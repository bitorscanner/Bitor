package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// NetblockService handles IP and netblock discovery operations
type NetblockService struct {
	app    *pocketbase.PocketBase
	logger *log.Logger
}

// IPSource represents an IP address and where it was discovered
type IPSource struct {
	IP           string `json:"ip"`
	Source       string `json:"source"`        // "dns", "mx", "ns", "web", "manual"
	SourceDomain string `json:"source_domain"` // which domain this came from
}

// CloudProvider represents a cloud or CDN provider and their IP ranges
type CloudProvider struct {
	Name   string       `json:"name"`
	Ranges []*net.IPNet `json:"-"`
	Source string       `json:"source"`
}

// NetblockConfidence represents how confident we are that an IP belongs to the organization
type NetblockConfidence struct {
	IP              string    `json:"ip"`
	CIDR            string    `json:"cidr"`
	ASN             int       `json:"asn"`
	ASNName         string    `json:"asn_name"`
	Organization    string    `json:"organization"`
	Confidence      float64   `json:"confidence"` // 0.0 to 1.0
	MatchedCriteria []string  `json:"matched_criteria"`
	LastChecked     time.Time `json:"last_checked"`
	Source          string    `json:"source"`
	OrgName         string    `json:"org_name"`
	Country         string    `json:"country"`
	LastUpdated     time.Time `json:"last_updated"`
}

// WhoisXMLAPIResponse represents the response from the WhoisXML API
type WhoisXMLAPIResponse struct {
	Search string `json:"search"`
	Result struct {
		Count    int `json:"count"`
		Limit    int `json:"limit"`
		Inetnums []struct {
			Inetnum     string   `json:"inetnum"`
			NetName     string   `json:"netname"`
			Description []string `json:"description"`
			Country     string   `json:"country"`
			Org         struct {
				Name    string `json:"name"`
				Country string `json:"country"`
			} `json:"org"`
		} `json:"inetnums"`
	} `json:"result"`
}

// WhoisIPData represents the response from WhoisXML API's IP WHOIS endpoint
type WhoisIPData struct {
	ASN          int      `json:"asn"`
	ASNName      string   `json:"asnName"`
	Organization string   `json:"organization"`
	CIDR         string   `json:"cidr"`
	Emails       []string `json:"emails"`
	Address      string   `json:"address"`
}

// NetblockDiscoveryRequest represents a request to discover netblocks
type NetblockDiscoveryRequest struct {
	ClientID     string   `json:"client_id"`
	OrgNames     []string `json:"org_names"`
	CustomRanges []string `json:"custom_ranges,omitempty"` // Manual IP ranges provided by user
	UseDomainIPs bool     `json:"use_domain_ips"`          // Whether to collect IPs from discovered domains
	FilterCloud  bool     `json:"filter_cloud"`            // Whether to filter out cloud provider IPs
}

// NetblockDiscoveryResult represents the result of netblock discovery
type NetblockDiscoveryResult struct {
	ClientID        string                `json:"client_id"`
	CollectedIPs    []IPSource            `json:"collected_ips"`
	FilteredIPs     []IPSource            `json:"filtered_ips"`
	CloudMatches    map[string][]string   `json:"cloud_matches"`
	NetblockResults []*NetblockConfidence `json:"netblock_results"`
	StartTime       time.Time             `json:"start_time"`
	EndTime         time.Time             `json:"end_time"`
	Duration        string                `json:"duration"`
	Error           string                `json:"error,omitempty"`
}

// Common cloud providers and CDNs to filter
var commonProviders = []struct {
	Name   string
	Source string
}{
	{"AWS", "https://ip-ranges.amazonaws.com/ip-ranges.json"},
	{"Azure", "https://download.microsoft.com/download/7/1/D/71D86715-5596-4529-9B13-DA13A5DE5B63/ServiceTags_Public_20220516.json"},
	{"Cloudflare", "https://www.cloudflare.com/ips-v4"},
	{"Fastly", "https://api.fastly.com/public-ip-list"},
}

// NewNetblockService creates a new instance of NetblockService
func NewNetblockService(app *pocketbase.PocketBase) *NetblockService {
	return &NetblockService{
		app:    app,
		logger: log.New(log.Writer(), "[NetblockService] ", log.LstdFlags),
	}
}

// GetWhoisXMLAPIKey retrieves WhoisXML API key from providers
func (s *NetblockService) GetWhoisXMLAPIKey() (string, error) {
	// Get enabled WhoisXML provider
	records, err := s.app.Dao().FindRecordsByFilter(
		"providers",
		"enabled = true && provider_type = 'whoisxml'",
		"created",
		0,
		1,
		map[string]interface{}{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to find WhoisXML provider: %v", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("no enabled WhoisXML provider found")
	}

	provider := records[0]

	// Get API keys for this provider
	apiKeys, err := s.app.Dao().FindRecordsByFilter(
		"api_keys",
		"provider = {:provider} && key_type = 'api_key'",
		"created",
		0,
		1,
		map[string]interface{}{
			"provider": provider.Id,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get API keys: %v", err)
	}

	if len(apiKeys) == 0 {
		return "", fmt.Errorf("no API key found for WhoisXML provider")
	}

	return apiKeys[0].GetString("key"), nil
}

// CollectDomainIPs gathers IPs from discovered domains for a client
func (s *NetblockService) CollectDomainIPs(clientID string) ([]IPSource, error) {
	var results []IPSource
	seen := make(map[string]bool) // Track unique IPs

	// Get all discovered domains for this client
	domains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client}",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %v", err)
	}

	s.logger.Printf("Collecting IPs from %d discovered domains", len(domains))

	for _, domainRecord := range domains {
		domain := domainRecord.GetString("domain")
		if domain == "" {
			continue
		}

		// Get A/AAAA records
		ips, err := net.LookupIP(domain)
		if err != nil {
			s.logger.Printf("Failed to lookup IPs for %s: %v", domain, err)
			continue
		}

		for _, ip := range ips {
			// Skip IPv6 for now
			if ip.To4() == nil {
				continue
			}

			ipStr := ip.String()
			if !seen[ipStr] {
				results = append(results, IPSource{
					IP:           ipStr,
					Source:       "dns",
					SourceDomain: domain,
				})
				seen[ipStr] = true
			}
		}

		// Get MX records
		mxRecords, err := net.LookupMX(domain)
		if err == nil {
			for _, mx := range mxRecords {
				mxIPs, err := net.LookupIP(mx.Host)
				if err != nil {
					continue
				}

				for _, ip := range mxIPs {
					if ip.To4() == nil {
						continue
					}

					ipStr := ip.String()
					if !seen[ipStr] {
						results = append(results, IPSource{
							IP:           ipStr,
							Source:       "mx",
							SourceDomain: domain,
						})
						seen[ipStr] = true
					}
				}
			}
		}

		// Get NS records
		nsRecords, err := net.LookupNS(domain)
		if err == nil {
			for _, ns := range nsRecords {
				nsIPs, err := net.LookupIP(ns.Host)
				if err != nil {
					continue
				}

				for _, ip := range nsIPs {
					if ip.To4() == nil {
						continue
					}

					ipStr := ip.String()
					if !seen[ipStr] {
						results = append(results, IPSource{
							IP:           ipStr,
							Source:       "ns",
							SourceDomain: domain,
						})
						seen[ipStr] = true
					}
				}
			}
		}
	}

	// Log summary
	var summary = make(map[string]int)
	for _, result := range results {
		summary[result.Source]++
	}

	s.logger.Printf("Collected IPs summary:")
	s.logger.Printf("- DNS A records: %d", summary["dns"])
	s.logger.Printf("- MX records: %d", summary["mx"])
	s.logger.Printf("- NS records: %d", summary["ns"])
	s.logger.Printf("Total unique IPs: %d", len(results))

	return results, nil
}

// FilterCloudIPs removes IPs belonging to major cloud providers and CDNs
func (s *NetblockService) FilterCloudIPs(ips []IPSource) ([]IPSource, map[string][]string, error) {
	// Initialize providers
	providers, err := s.initializeCloudProviders()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize cloud providers: %w", err)
	}

	var filtered []IPSource
	matches := make(map[string][]string) // Track which IPs matched which providers

	for _, ip := range ips {
		isCloudIP := false
		parsedIP := net.ParseIP(ip.IP)
		if parsedIP == nil {
			s.logger.Printf("Invalid IP address: %s", ip.IP)
			continue
		}

		// Check against each provider
		for _, provider := range providers {
			for _, ipNet := range provider.Ranges {
				if ipNet.Contains(parsedIP) {
					isCloudIP = true
					matches[provider.Name] = append(matches[provider.Name], ip.IP)
					break
				}
			}
			if isCloudIP {
				break
			}
		}

		if !isCloudIP {
			filtered = append(filtered, ip)
		}
	}

	// Log summary
	s.logger.Printf("Cloud IP filtering results:")
	for provider, matchedIPs := range matches {
		s.logger.Printf("- %s: %d IPs", provider, len(matchedIPs))
	}
	s.logger.Printf("Remaining IPs after filtering: %d", len(filtered))

	return filtered, matches, nil
}

// initializeCloudProviders fetches and parses IP ranges for known providers
func (s *NetblockService) initializeCloudProviders() ([]CloudProvider, error) {
	var providers []CloudProvider

	for _, info := range commonProviders {
		provider := CloudProvider{
			Name:   info.Name,
			Source: info.Source,
		}

		switch info.Name {
		case "AWS":
			ranges, err := s.getAWSRanges(info.Source)
			if err != nil {
				s.logger.Printf("Failed to get AWS ranges: %v", err)
				continue
			}
			provider.Ranges = ranges

		case "Cloudflare":
			ranges, err := s.getCloudflareRanges(info.Source)
			if err != nil {
				s.logger.Printf("Failed to get Cloudflare ranges: %v", err)
				continue
			}
			provider.Ranges = ranges

		case "Azure":
			ranges, err := s.getAzureRanges(info.Source)
			if err != nil {
				s.logger.Printf("Failed to get Azure ranges: %v", err)
				continue
			}
			provider.Ranges = ranges

		case "Fastly":
			ranges, err := s.getFastlyRanges(info.Source)
			if err != nil {
				s.logger.Printf("Failed to get Fastly ranges: %v", err)
				continue
			}
			provider.Ranges = ranges
		}

		if len(provider.Ranges) > 0 {
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// Helper functions to fetch and parse provider-specific IP ranges
func (s *NetblockService) getAWSRanges(url string) ([]*net.IPNet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Prefixes []struct {
			IPPrefix string `json:"ip_prefix"`
		} `json:"prefixes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var ranges []*net.IPNet
	for _, prefix := range data.Prefixes {
		_, ipNet, err := net.ParseCIDR(prefix.IPPrefix)
		if err != nil {
			continue
		}
		ranges = append(ranges, ipNet)
	}

	return ranges, nil
}

// getCloudflareRanges fetches Cloudflare's IPv4 ranges
func (s *NetblockService) getCloudflareRanges(url string) ([]*net.IPNet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Cloudflare returns one CIDR per line
	var ranges []*net.IPNet
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		_, ipNet, err := net.ParseCIDR(scanner.Text())
		if err != nil {
			s.logger.Printf("Invalid Cloudflare CIDR: %s", scanner.Text())
			continue
		}
		ranges = append(ranges, ipNet)
	}

	return ranges, scanner.Err()
}

// getAzureRanges fetches Microsoft Azure's IP ranges
func (s *NetblockService) getAzureRanges(url string) ([]*net.IPNet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Values []struct {
			Properties struct {
				AddressPrefixes []string `json:"addressPrefixes"`
			} `json:"properties"`
		} `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var ranges []*net.IPNet
	for _, value := range data.Values {
		for _, prefix := range value.Properties.AddressPrefixes {
			// Skip IPv6 addresses
			if strings.Contains(prefix, ":") {
				continue
			}
			_, ipNet, err := net.ParseCIDR(prefix)
			if err != nil {
				s.logger.Printf("Invalid Azure CIDR: %s", prefix)
				continue
			}
			ranges = append(ranges, ipNet)
		}
	}

	return ranges, nil
}

// getFastlyRanges fetches Fastly's IP ranges
func (s *NetblockService) getFastlyRanges(url string) ([]*net.IPNet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Addresses []string `json:"addresses"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var ranges []*net.IPNet
	for _, cidr := range data.Addresses {
		// Skip IPv6 addresses
		if strings.Contains(cidr, ":") {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			s.logger.Printf("Invalid Fastly CIDR: %s", cidr)
			continue
		}
		ranges = append(ranges, ipNet)
	}

	return ranges, nil
}

// ParseCustomRanges validates and parses custom IP ranges provided by user
func (s *NetblockService) ParseCustomRanges(customRanges []string) ([]IPSource, error) {
	var results []IPSource

	for _, rangeStr := range customRanges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}

		// Try parsing as CIDR
		if strings.Contains(rangeStr, "/") {
			_, ipNet, err := net.ParseCIDR(rangeStr)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR range: %s", rangeStr)
			}

			// For storage, we'll use the network address
			results = append(results, IPSource{
				IP:           ipNet.IP.String(),
				Source:       "manual",
				SourceDomain: rangeStr, // Store the original range string
			})
		} else if strings.Contains(rangeStr, "-") {
			// Try parsing as IP range (e.g., "192.168.1.1-192.168.1.100")
			parts := strings.Split(rangeStr, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid IP range format: %s", rangeStr)
			}

			startIP := net.ParseIP(strings.TrimSpace(parts[0]))
			endIP := net.ParseIP(strings.TrimSpace(parts[1]))

			if startIP == nil || endIP == nil {
				return nil, fmt.Errorf("invalid IP addresses in range: %s", rangeStr)
			}

			// For storage, we'll use the start IP
			results = append(results, IPSource{
				IP:           startIP.String(),
				Source:       "manual",
				SourceDomain: rangeStr, // Store the original range string
			})
		} else {
			// Try parsing as single IP
			ip := net.ParseIP(rangeStr)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address: %s", rangeStr)
			}

			results = append(results, IPSource{
				IP:           ip.String(),
				Source:       "manual",
				SourceDomain: rangeStr,
			})
		}
	}

	return results, nil
}

// DiscoverNetblocks performs comprehensive netblock discovery
func (s *NetblockService) DiscoverNetblocks(ctx context.Context, req NetblockDiscoveryRequest) (*NetblockDiscoveryResult, error) {
	startTime := time.Now()

	result := &NetblockDiscoveryResult{
		ClientID:     req.ClientID,
		StartTime:    startTime,
		CloudMatches: make(map[string][]string),
	}

	s.logger.Printf("Starting netblock discovery for client: %s", req.ClientID)

	var allIPs []IPSource

	// 1. Collect IPs from discovered domains if requested
	if req.UseDomainIPs {
		s.logger.Printf("Collecting IPs from discovered domains...")
		domainIPs, err := s.CollectDomainIPs(req.ClientID)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to collect domain IPs: %v", err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime).String()
			return result, err
		}
		allIPs = append(allIPs, domainIPs...)
		s.logger.Printf("Collected %d IPs from domains", len(domainIPs))
	}

	// 2. Add custom ranges if provided
	if len(req.CustomRanges) > 0 {
		s.logger.Printf("Processing %d custom IP ranges...", len(req.CustomRanges))
		customIPs, err := s.ParseCustomRanges(req.CustomRanges)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to parse custom ranges: %v", err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime).String()
			return result, err
		}
		allIPs = append(allIPs, customIPs...)
		s.logger.Printf("Added %d IPs from custom ranges", len(customIPs))
	}

	result.CollectedIPs = allIPs

	// 3. Filter cloud IPs if requested
	var filteredIPs []IPSource
	if req.FilterCloud {
		s.logger.Printf("Filtering cloud provider IPs...")
		var err error
		filteredIPs, result.CloudMatches, err = s.FilterCloudIPs(allIPs)
		if err != nil {
			s.logger.Printf("Warning: Cloud filtering failed: %v", err)
			filteredIPs = allIPs // Use unfiltered IPs
		}
		s.logger.Printf("Filtered out %d cloud IPs, %d remaining", len(allIPs)-len(filteredIPs), len(filteredIPs))
	} else {
		filteredIPs = allIPs
	}

	result.FilteredIPs = filteredIPs

	// 4. Discover netblocks using WhoisXML API
	if len(req.OrgNames) > 0 {
		s.logger.Printf("Discovering netblocks for %d organizations...", len(req.OrgNames))
		netblocks, err := s.GetNetblocksByOrganization(req.OrgNames)
		if err != nil {
			s.logger.Printf("Warning: Netblock discovery failed: %v", err)
		} else {
			result.NetblockResults = netblocks
			s.logger.Printf("Discovered %d netblocks", len(netblocks))
		}
	}

	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	s.logger.Printf("Netblock discovery completed in %s", result.Duration)

	return result, nil
}

// GetNetblocksByOrganization discovers netblocks using WhoisXML API
func (s *NetblockService) GetNetblocksByOrganization(orgNames []string) ([]*NetblockConfidence, error) {
	// Get API key
	apiKey, err := s.GetWhoisXMLAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get WhoisXML API key: %v", err)
	}

	var allResults []*NetblockConfidence
	seenCIDRs := make(map[string]bool) // Track unique CIDRs

	// Try each organization name
	for _, orgName := range orgNames {
		if orgName == "" {
			continue
		}

		s.logger.Printf("Fetching netblocks for organization: %s", orgName)
		var from string       // Used for pagination
		const maxLimit = 1000 // Maximum records per request

		for {
			// Construct API URL with pagination
			baseURL := fmt.Sprintf("https://ip-netblocks.whoisxmlapi.com/api/v2?apiKey=%s&org[]=%s&limit=%d",
				apiKey, url.QueryEscape(orgName), maxLimit)
			if from != "" {
				baseURL += "&from=" + url.QueryEscape(from)
			}

			// Make HTTP request
			resp, err := http.Get(baseURL)
			if err != nil {
				s.logger.Printf("API request failed for org '%s': %v", orgName, err)
				break
			}

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				s.logger.Printf("API returned status %d for org '%s': %s", resp.StatusCode, orgName, string(body))
				break
			}

			// Parse response
			var apiResp WhoisXMLAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				resp.Body.Close()
				s.logger.Printf("Failed to parse response for org '%s': %v", orgName, err)
				break
			}
			resp.Body.Close()

			// Process results
			for _, inetnum := range apiResp.Result.Inetnums {
				// Parse IP range
				ipRange := strings.Split(inetnum.Inetnum, " - ")
				if len(ipRange) != 2 {
					s.logger.Printf("Invalid IP range format for org '%s': %s", orgName, inetnum.Inetnum)
					continue
				}

				// Try to get CIDR notation
				cidr := s.getCIDRFromRange(ipRange[0], ipRange[1])
				if cidr == "" {
					s.logger.Printf("Failed to convert IP range to CIDR for org '%s': %s", orgName, inetnum.Inetnum)
					continue
				}

				// Skip if we've already seen this CIDR
				if seenCIDRs[cidr] {
					continue
				}
				seenCIDRs[cidr] = true

				// Create NetblockConfidence entry
				confidence := &NetblockConfidence{
					IP:          ipRange[0],
					CIDR:        cidr,
					Confidence:  0.9, // High confidence since it's from official WHOIS data
					Source:      "whoisxml",
					OrgName:     inetnum.Org.Name,
					Country:     inetnum.Country,
					LastUpdated: time.Now(),
					MatchedCriteria: []string{
						fmt.Sprintf("Found via organization search: %s", orgName),
					},
				}

				allResults = append(allResults, confidence)
				s.logger.Printf("Found netblock %s for org '%s'", cidr, orgName)
			}

			// Check if there are more results
			if apiResp.Result.Count < maxLimit {
				break // No more results
			}

			// Get the next range for pagination
			if len(apiResp.Result.Inetnums) > 0 {
				lastInetnum := apiResp.Result.Inetnums[len(apiResp.Result.Inetnums)-1]
				from = lastInetnum.Inetnum
			} else {
				break // No more results
			}

			s.logger.Printf("Fetched %d netblocks for org '%s', getting next page...", len(allResults), orgName)
		}
	}

	s.logger.Printf("Total unique netblocks found across all organizations: %d", len(allResults))
	return allResults, nil
}

// Helper function to convert IP range to CIDR
func (s *NetblockService) getCIDRFromRange(startIP, endIP string) string {
	// Parse IPs
	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)
	if start == nil || end == nil {
		s.logger.Printf("Invalid IP address in range: %s - %s", startIP, endIP)
		return ""
	}

	// Convert to integers
	startInt, err := s.ipToUint32(start)
	if err != nil {
		s.logger.Printf("Failed to convert start IP to int: %v", err)
		return ""
	}

	endInt, err := s.ipToUint32(end)
	if err != nil {
		s.logger.Printf("Failed to convert end IP to int: %v", err)
		return ""
	}

	// Find the number of addresses in the range
	size := endInt - startInt + 1

	// Find the largest CIDR that fits
	bits := 32 - int(math.Floor(math.Log2(float64(size))))
	if bits < 0 || bits > 32 {
		s.logger.Printf("Invalid CIDR bits calculated: %d", bits)
		return ""
	}

	// Convert back to CIDR notation
	return fmt.Sprintf("%s/%d", startIP, bits)
}

// Helper function to convert IP to uint32
func (s *NetblockService) ipToUint32(ip net.IP) (uint32, error) {
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("invalid IPv4 address")
	}

	if len(ip) != 4 {
		return 0, fmt.Errorf("IP address must be exactly 4 bytes")
	}

	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3]), nil
}

// SaveResults saves netblock discovery results to the database
func (s *NetblockService) SaveResults(clientID string, result *NetblockDiscoveryResult, scanID string) error {
	// Save to attack_surface_ips collection for collected/filtered IPs
	for _, ip := range result.FilteredIPs {
		collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_ips")
		if err != nil {
			s.logger.Printf("Failed to find attack_surface_ips collection: %v", err)
			continue
		}

		record := models.NewRecord(collection)
		record.Set("client", clientID)
		record.Set("ip", ip.IP)
		record.Set("source", ip.Source)
		record.Set("source_domain", ip.SourceDomain)
		record.Set("discovered_at", result.StartTime.Format(time.RFC3339))

		if scanID != "" {
			record.Set("scan_id", scanID)
		}

		if err := s.app.Dao().SaveRecord(record); err != nil {
			s.logger.Printf("Failed to save IP %s: %v", ip.IP, err)
		}
	}

	// Save netblock results
	for _, nb := range result.NetblockResults {
		collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_netblocks")
		if err != nil {
			s.logger.Printf("Failed to find attack_surface_netblocks collection: %v", err)
			continue
		}

		record := models.NewRecord(collection)
		record.Set("client", clientID)
		record.Set("ip", nb.IP)
		record.Set("cidr", nb.CIDR)
		record.Set("asn", nb.ASN)
		record.Set("asn_name", nb.ASNName)
		record.Set("organization", nb.Organization)
		record.Set("confidence", nb.Confidence)
		record.Set("source", nb.Source)
		record.Set("org_name", nb.OrgName)
		record.Set("country", nb.Country)
		record.Set("discovered_at", result.StartTime.Format(time.RFC3339))

		// Store matched criteria as JSON
		criteriaJSON, _ := json.Marshal(nb.MatchedCriteria)
		record.Set("matched_criteria", string(criteriaJSON))

		if scanID != "" {
			record.Set("scan_id", scanID)
		}

		if err := s.app.Dao().SaveRecord(record); err != nil {
			s.logger.Printf("Failed to save netblock %s: %v", nb.CIDR, err)
		}
	}

	s.logger.Printf("Saved %d IPs and %d netblocks to database", len(result.FilteredIPs), len(result.NetblockResults))
	return nil
}
