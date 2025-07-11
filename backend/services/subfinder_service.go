package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// SubfinderService handles subdomain enumeration using subfinder
type SubfinderService struct {
	app    *pocketbase.PocketBase
	logger *log.Logger
}

// SubfinderResult represents the result of a subfinder scan
type SubfinderResult struct {
	Domain           string    `json:"domain"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Duration         string    `json:"duration"`
	Subdomains       []string  `json:"subdomains"`
	TotalSubdomains  int       `json:"total_subdomains"`
	UniqueSubdomains int       `json:"unique_subdomains"`
	Sources          []string  `json:"sources_used"`
	Error            string    `json:"error,omitempty"`
	ClientID         string    `json:"client_id"`
}

// SubfinderOutput represents subfinder JSON output format
type SubfinderOutput struct {
	Host   string `json:"host"`
	Source string `json:"source"`
}

// SourceInfo represents information about a subfinder source
type SourceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RequiresKey bool   `json:"requires_key"`
	Category    string `json:"category"`
}

// NewSubfinderService creates a new instance of SubfinderService
func NewSubfinderService(app *pocketbase.PocketBase) *SubfinderService {
	return &SubfinderService{
		app:    app,
		logger: log.New(log.Writer(), "[SubfinderService] ", log.LstdFlags),
	}
}

// RunSubfinder executes subfinder for subdomain enumeration
func (s *SubfinderService) RunSubfinder(ctx context.Context, domain, clientID string, options map[string]interface{}) (*SubfinderResult, error) {
	startTime := time.Now()

	result := &SubfinderResult{
		Domain:    domain,
		StartTime: startTime,
		ClientID:  clientID,
	}

	s.logger.Printf("Starting subfinder scan for domain: %s", domain)

	// Handle TLD-only scanning
	if includeTLDs, ok := options["include_tlds"].(bool); ok && includeTLDs && domain == "tld-only-scan" {
		s.logger.Printf("TLD-only scan requested, getting TLD domains for client: %s", clientID)
		return s.runTLDScan(ctx, clientID, options, result, startTime)
	}

	// Regular domain scan
	return s.runRegularScan(ctx, domain, options, result, startTime)
}

// runTLDScan handles scanning of discovered TLD domains
func (s *SubfinderService) runTLDScan(ctx context.Context, clientID string, options map[string]interface{}, result *SubfinderResult, startTime time.Time) (*SubfinderResult, error) {
	// Get TLD domains from database - these are domains discovered via TLD discovery
	// Look for domains where the source indicates TLD discovery or where they are root domains
	tldDomains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client} && (source ~ 'tld' || source ~ 'ms_tenant' || source = 'manual')",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get TLD domains: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(tldDomains) == 0 {
		result.Error = "No TLD domains found. Please run TLD discovery first."
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, fmt.Errorf("no TLD domains found")
	}

	s.logger.Printf("Found %d TLD domains to scan", len(tldDomains))

	// Collect unique domains to scan (avoid duplicates)
	uniqueDomains := make(map[string]bool)
	for _, tldRecord := range tldDomains {
		domain := tldRecord.GetString("domain")
		if domain != "" {
			uniqueDomains[domain] = true
		}
	}

	s.logger.Printf("Unique domains to scan: %v", getKeysFromMap(uniqueDomains))

	// Collect all subdomains and sources from all TLD domains
	var allSubdomains []string
	sourcesMap := make(map[string]bool)

	for domain := range uniqueDomains {
		s.logger.Printf("Scanning TLD domain: %s", domain)

		// Run subfinder for this TLD domain
		tldResult, err := s.runRegularScan(ctx, domain, options, &SubfinderResult{
			Domain:    domain,
			StartTime: startTime,
			ClientID:  clientID,
		}, startTime)

		if err != nil {
			s.logger.Printf("Failed to scan TLD domain %s: %v", domain, err)
			continue
		}

		// Collect results
		allSubdomains = append(allSubdomains, tldResult.Subdomains...)
		for _, source := range tldResult.Sources {
			sourcesMap[source] = true
		}
	}

	// Convert sources map to slice
	var sources []string
	for source := range sourcesMap {
		sources = append(sources, source)
	}

	// Update result
	result.Domain = fmt.Sprintf("TLD scan (%d domains)", len(uniqueDomains))
	result.Subdomains = allSubdomains
	result.TotalSubdomains = len(allSubdomains)
	result.UniqueSubdomains = len(allSubdomains)
	result.Sources = sources
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	s.logger.Printf("TLD scan completed: %d subdomains found across %d TLD domains in %s", result.TotalSubdomains, len(uniqueDomains), result.Duration)

	return result, nil
}

// getKeysFromMap is a helper function to get keys from a map for logging
func getKeysFromMap(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// runRegularScan handles normal domain scanning
func (s *SubfinderService) runRegularScan(ctx context.Context, domain string, options map[string]interface{}, result *SubfinderResult, startTime time.Time) (*SubfinderResult, error) {
	// Ensure subfinder is installed
	if err := s.ensureSubfinderInstalled(); err != nil {
		result.Error = fmt.Sprintf("Failed to ensure subfinder is installed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "subfinder_output_*.json")
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create output file: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Build subfinder command
	args := s.buildSubfinderArgs(domain, outputFile.Name(), options)

	s.logger.Printf("Running subfinder command: subfinder %s", strings.Join(args, " "))

	// Execute subfinder
	cmd := exec.CommandContext(ctx, "subfinder", args...)

	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create stderr pipe: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Sprintf("Failed to start subfinder: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Log stderr output
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.logger.Printf("subfinder: %s", scanner.Text())
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		result.Error = fmt.Sprintf("Subfinder execution failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Parse results
	subdomains, sources, err := s.parseSubfinderOutput(outputFile.Name())
	if err != nil {
		result.Error = fmt.Sprintf("Failed to parse subfinder output: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.Subdomains = subdomains
	result.TotalSubdomains = len(subdomains)
	result.UniqueSubdomains = len(subdomains) // Already unique from subfinder
	result.Sources = sources
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	s.logger.Printf("Subfinder scan completed: %d subdomains found in %s", result.TotalSubdomains, result.Duration)

	return result, nil
}

// buildSubfinderArgs constructs command line arguments for subfinder
func (s *SubfinderService) buildSubfinderArgs(domain, outputFile string, options map[string]interface{}) []string {
	var args []string

	// Target domain
	args = append(args, "-d", domain)

	// Output format
	args = append(args, "-json", "-o", outputFile)

	// Sources configuration
	if sources, ok := options["sources"].([]string); ok && len(sources) > 0 {
		args = append(args, "-sources", strings.Join(sources, ","))
	}

	if allSources, ok := options["all_sources"].(bool); ok && allSources {
		args = append(args, "-all")
	}

	// Timeout options
	if timeout, ok := options["timeout"].(int); ok && timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", timeout))
	}

	if maxTime, ok := options["max_time"].(int); ok && maxTime > 0 {
		args = append(args, "-max-time", fmt.Sprintf("%d", maxTime))
	}

	// Rate limiting
	if rateLimit, ok := options["rate_limit"].(int); ok && rateLimit > 0 {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", rateLimit))
	}

	// Recursive enumeration
	if recursive, ok := options["recursive"].(bool); ok && recursive {
		args = append(args, "-recursive")
	}

	// Silent mode for cleaner output
	args = append(args, "-silent")

	return args
}

// parseSubfinderOutput parses subfinder JSON output
func (s *SubfinderService) parseSubfinderOutput(outputFile string) ([]string, []string, error) {
	s.logger.Printf("Parsing subfinder output from file: %s", outputFile)

	file, err := os.Open(outputFile)
	if err != nil {
		s.logger.Printf("Failed to open output file: %v", err)
		return nil, nil, err
	}
	defer file.Close()

	// Check file size for debugging
	if stat, err := file.Stat(); err == nil {
		s.logger.Printf("Output file size: %d bytes", stat.Size())
	}

	var subdomains []string
	sourcesMap := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if strings.TrimSpace(line) == "" {
			continue
		}

		s.logger.Printf("Processing line %d: %s", lineCount, line)

		var subfinderResult SubfinderOutput
		if err := json.Unmarshal([]byte(line), &subfinderResult); err != nil {
			s.logger.Printf("Failed to parse JSON on line %d, trying as plain text: %v", lineCount, err)
			// Try parsing as plain text (fallback)
			if strings.TrimSpace(line) != "" {
				subdomains = append(subdomains, strings.TrimSpace(line))
				s.logger.Printf("Added plain text subdomain: %s", strings.TrimSpace(line))
			}
			continue
		}

		if subfinderResult.Host != "" {
			subdomains = append(subdomains, subfinderResult.Host)
			s.logger.Printf("Added JSON subdomain: %s from source: %s", subfinderResult.Host, subfinderResult.Source)
			if subfinderResult.Source != "" {
				sourcesMap[subfinderResult.Source] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		s.logger.Printf("Scanner error: %v", err)
		return nil, nil, err
	}

	// Convert sources map to slice
	var sources []string
	for source := range sourcesMap {
		sources = append(sources, source)
	}

	s.logger.Printf("Parsing complete: found %d subdomains from %d sources", len(subdomains), len(sources))
	if len(subdomains) > 0 {
		s.logger.Printf("First few subdomains: %v", subdomains[:min(len(subdomains), 3)])
	}

	return subdomains, sources, nil
}

// min is a helper function since Go doesn't have a built-in min for ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ensureSubfinderInstalled checks if subfinder is installed and installs it if needed
func (s *SubfinderService) ensureSubfinderInstalled() error {
	// First check if subfinder is already available in PATH
	if _, err := exec.LookPath("subfinder"); err == nil {
		return nil
	}

	// If not found, install it using go install
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "install", "-v", "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install subfinder: %w", err)
	}

	// Verify installation
	if _, err := exec.LookPath("subfinder"); err != nil {
		return fmt.Errorf("subfinder installation failed - not found in PATH after installation")
	}

	return nil
}

// SaveResults saves subfinder results to the database
func (s *SubfinderService) SaveResults(clientID string, result *SubfinderResult, scanID string) error {
	collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_domains")
	if err != nil {
		return fmt.Errorf("collection not found: %w", err)
	}

	for _, subdomain := range result.Subdomains {
		record := models.NewRecord(collection)
		record.Set("client", clientID)
		record.Set("domain", subdomain)
		record.Set("parent_domain", result.Domain)
		record.Set("source", "subfinder")
		record.Set("resolved", false)
		record.Set("discovered_at", result.StartTime)
		record.Set("scan_id", scanID)

		// Add subfinder metadata
		metadata := map[string]interface{}{
			"discovery_method": "subfinder",
			"sources_used":     result.Sources,
			"scan_duration":    result.Duration,
		}
		record.Set("metadata", metadata)

		if err := s.app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save subdomain result: %w", err)
		}
	}

	s.logger.Printf("Saved %d subfinder results to database", len(result.Subdomains))
	return nil
}

// GetSavedSubdomains retrieves saved subdomain results from the database
func (s *SubfinderService) GetSavedSubdomains(clientID, domain string) ([]*models.Record, error) {
	filter := "client = {:client}"
	params := map[string]interface{}{
		"client": clientID,
	}

	if domain != "" {
		filter += " && (domain ~ {:domain} || parent_domain ~ {:parent_domain})"
		params["domain"] = domain
		params["parent_domain"] = domain
	}

	filter += " && source = 'subfinder'"

	records, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		filter,
		"created",
		0,
		-1,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subdomain results: %w", err)
	}

	return records, nil
}

// GetAvailableSources returns the list of available subfinder sources with metadata
func (s *SubfinderService) GetAvailableSources() []SourceInfo {
	return []SourceInfo{
		{Name: "alienvault", Description: "AlienVault OTX", RequiresKey: true, Category: "Threat Intelligence"},
		{Name: "anubis", Description: "Anubis", RequiresKey: false, Category: "Certificate Transparency"},
		{Name: "bevigil", Description: "BeVigil", RequiresKey: true, Category: "Mobile App Intelligence"},
		{Name: "binaryedge", Description: "BinaryEdge", RequiresKey: true, Category: "Internet Scanning"},
		{Name: "bufferover", Description: "BufferOver", RequiresKey: false, Category: "DNS"},
		{Name: "c99", Description: "C99.nl", RequiresKey: true, Category: "Subdomain Finder"},
		{Name: "censys", Description: "Censys", RequiresKey: true, Category: "Internet Scanning"},
		{Name: "certspotter", Description: "CertSpotter", RequiresKey: false, Category: "Certificate Transparency"},
		{Name: "chaos", Description: "Chaos", RequiresKey: true, Category: "ProjectDiscovery"},
		{Name: "chinaz", Description: "ChinaZ", RequiresKey: false, Category: "DNS"},
		{Name: "crtsh", Description: "crt.sh", RequiresKey: false, Category: "Certificate Transparency"},
		{Name: "dnsdb", Description: "Farsight DNSDB", RequiresKey: true, Category: "DNS Intelligence"},
		{Name: "dnsdumpster", Description: "DNSDumpster", RequiresKey: false, Category: "DNS"},
		{Name: "dnsrepo", Description: "DNS Repo", RequiresKey: false, Category: "DNS"},
		{Name: "fofa", Description: "FOFA", RequiresKey: true, Category: "Internet Scanning"},
		{Name: "fullhunt", Description: "FullHunt", RequiresKey: true, Category: "Attack Surface"},
		{Name: "github", Description: "GitHub", RequiresKey: true, Category: "Code Repository"},
		{Name: "hackertarget", Description: "HackerTarget", RequiresKey: false, Category: "Security Tools"},
		{Name: "hunter", Description: "Hunter.io", RequiresKey: true, Category: "Email Finding"},
		{Name: "intelx", Description: "Intelligence X", RequiresKey: true, Category: "Search Engine"},
		{Name: "passivetotal", Description: "PassiveTotal", RequiresKey: true, Category: "Threat Intelligence"},
		{Name: "quake", Description: "Quake", RequiresKey: true, Category: "Internet Scanning"},
		{Name: "rapiddns", Description: "RapidDNS", RequiresKey: false, Category: "DNS"},
		{Name: "reconcloud", Description: "ReconCloud", RequiresKey: false, Category: "Reconnaissance"},
		{Name: "riddler", Description: "Riddler", RequiresKey: false, Category: "DNS"},
		{Name: "robtex", Description: "Robtex", RequiresKey: false, Category: "DNS"},
		{Name: "securitytrails", Description: "SecurityTrails", RequiresKey: true, Category: "DNS Intelligence"},
		{Name: "shodan", Description: "Shodan", RequiresKey: true, Category: "Internet Scanning"},
		{Name: "spyse", Description: "Spyse", RequiresKey: true, Category: "Internet Intelligence"},
		{Name: "sublist3r", Description: "Sublist3r", RequiresKey: false, Category: "Subdomain Enumeration"},
		{Name: "threatbook", Description: "ThreatBook", RequiresKey: true, Category: "Threat Intelligence"},
		{Name: "threatcrowd", Description: "ThreatCrowd", RequiresKey: false, Category: "Threat Intelligence"},
		{Name: "threatminer", Description: "ThreatMiner", RequiresKey: false, Category: "Threat Intelligence"},
		{Name: "virustotal", Description: "VirusTotal", RequiresKey: true, Category: "Threat Intelligence"},
		{Name: "waybackarchive", Description: "Wayback Machine", RequiresKey: false, Category: "Web Archive"},
		{Name: "whoisxmlapi", Description: "WhoisXML API", RequiresKey: true, Category: "WHOIS/DNS"},
		{Name: "zoomeye", Description: "ZoomEye", RequiresKey: true, Category: "Internet Scanning"},
	}
}
