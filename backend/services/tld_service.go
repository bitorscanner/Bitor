package services

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// TLDService handles top-level domain discovery
type TLDService struct {
	app *pocketbase.PocketBase
}

// NewTLDService creates a new TLD service instance
func NewTLDService(app *pocketbase.PocketBase) *TLDService {
	return &TLDService{
		app: app,
	}
}

// UserRealmResponse represents the response from getuserrealm endpoint
type UserRealmResponse struct {
	XMLName             xml.Name `xml:"RealmInfo"`
	IsFederated         bool     `xml:"IsFederated"`
	DomainName          string   `xml:"DomainName"`
	CloudInstance       string   `xml:"CloudInstance"`
	FederationBrandName string   `xml:"FederationBrandName"`
	NameSpaceType       string   `xml:"NameSpaceType"`
	AuthURL             string   `xml:"AuthURL"`
}

// SOAP template for GetFederationInformation request
const soapTemplate = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:exm="http://schemas.microsoft.com/exchange/services/2006/messages" 
               xmlns:ext="http://schemas.microsoft.com/exchange/services/2006/types" 
               xmlns:a="http://www.w3.org/2005/08/addressing" 
               xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<soap:Header>
		<a:Action soap:mustUnderstand="1">http://schemas.microsoft.com/exchange/2010/Autodiscover/Autodiscover/GetFederationInformation</a:Action>
		<a:To soap:mustUnderstand="1">https://autodiscover-s.outlook.com/autodiscover/autodiscover.svc</a:To>
		<a:ReplyTo>
			<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		</a:ReplyTo>
	</soap:Header>
	<soap:Body>
		<GetFederationInformationRequestMessage xmlns="http://schemas.microsoft.com/exchange/2010/Autodiscover">
			<Request>
				<Domain>%s</Domain>
			</Request>
		</GetFederationInformationRequestMessage>
	</soap:Body>
</soap:Envelope>`

// federationResponse represents the SOAP response structure
type federationResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetFedInfoResponse struct {
			Response struct {
				Domains struct {
					Domain []string `xml:"Domain"`
				} `xml:"Domains"`
			} `xml:"Response"`
		} `xml:"GetFederationInformationResponseMessage"`
	} `xml:"Body"`
}

// TLDDiscoveryResult represents the result of TLD discovery
type TLDDiscoveryResult struct {
	Domain            string             `json:"domain"`
	StartTime         time.Time          `json:"start_time"`
	EndTime           time.Time          `json:"end_time"`
	Duration          string             `json:"duration"`
	TenantInfo        *UserRealmResponse `json:"tenant_info,omitempty"`
	TenantID          string             `json:"tenant_id,omitempty"`
	DiscoveredDomains []string           `json:"discovered_domains"`
	TotalDomains      int                `json:"total_domains"`
	Error             string             `json:"error,omitempty"`
}

// DiscoverTLDs performs top-level domain discovery for Microsoft tenants
func (s *TLDService) DiscoverTLDs(ctx context.Context, domain, clientID string, options map[string]interface{}) (*TLDDiscoveryResult, error) {
	startTime := time.Now()

	result := &TLDDiscoveryResult{
		Domain:    domain,
		StartTime: startTime,
	}

	fmt.Printf("DEBUG: Starting TLD discovery for domain: %s\n", domain)

	// Get user realm info
	realmInfo, err := s.getUserRealmInfo(domain)
	if err != nil {
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		result.Error = fmt.Sprintf("Failed to get tenant info: %v", err)
		return result, err
	}

	result.TenantInfo = realmInfo

	// Get tenant ID
	tenantID, err := s.getTenantID(domain)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get tenant ID: %v\n", err)
	} else {
		result.TenantID = tenantID
	}

	// Check if this is a Microsoft tenant
	if realmInfo.FederationBrandName == "" {
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		result.DiscoveredDomains = []string{domain}
		result.TotalDomains = 1
		return result, nil
	}

	fmt.Printf("DEBUG: Found Microsoft tenant: %s\n", realmInfo.FederationBrandName)

	// Get tenant domains using SOAP request
	domains, err := s.getTenantDomains(domain)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get tenant domains: %v\n", err)
		result.DiscoveredDomains = []string{domain}
	} else {
		// Filter domains based on basic exclusion patterns
		excludePatterns := []string{
			"*.onmicrosoft.com",
			"*.microsoftonline.com",
			"*.mail.onmicrosoft.com",
		}

		var filteredDomains []string
		for _, d := range domains {
			if !s.shouldExcludeDomain(d, excludePatterns) {
				filteredDomains = append(filteredDomains, d)
			} else {
				fmt.Printf("DEBUG: Excluding domain: %s\n", d)
			}
		}

		result.DiscoveredDomains = filteredDomains
	}

	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()
	result.TotalDomains = len(result.DiscoveredDomains)

	fmt.Printf("DEBUG: TLD discovery completed. Found %d domains\n", result.TotalDomains)

	return result, nil
}

// getUserRealmInfo gets Microsoft tenant realm information
func (s *TLDService) getUserRealmInfo(domain string) (*UserRealmResponse, error) {
	url := fmt.Sprintf("https://login.microsoftonline.com/getuserrealm.srf?login=user@%s&xml=1", domain)

	resp, err := s.httpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserRealmResponse
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode user realm response: %w", err)
	}

	return &result, nil
}

// getTenantID extracts the tenant ID from Microsoft's OpenID configuration
func (s *TLDService) getTenantID(domain string) (string, error) {
	url := fmt.Sprintf("https://login.microsoftonline.com/%s/.well-known/openid-configuration", domain)
	resp, err := s.httpGet(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Extract tenant ID from authorization endpoint
	if result.AuthorizationEndpoint != "" {
		parts := strings.Split(result.AuthorizationEndpoint, "/")
		if len(parts) > 3 {
			return parts[3], nil
		}
	}
	return "", fmt.Errorf("tenant ID not found")
}

// getTenantDomains uses SOAP request to get all tenant domains
func (s *TLDService) getTenantDomains(domain string) ([]string, error) {
	soapBody := fmt.Sprintf(soapTemplate, domain)

	req, err := http.NewRequest("POST", "https://autodiscover-s.outlook.com/autodiscover/autodiscover.svc", bytes.NewBufferString(soapBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"http://schemas.microsoft.com/exchange/2010/Autodiscover/Autodiscover/GetFederationInformation"`)
	req.Header.Set("User-Agent", "AutodiscoverClient")

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result federationResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	return result.Body.GetFedInfoResponse.Response.Domains.Domain, nil
}

// shouldExcludeDomain checks if a domain should be excluded based on patterns
func (s *TLDService) shouldExcludeDomain(domain string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		// Convert glob pattern to regex
		regexPattern := strings.ReplaceAll(pattern, ".", "\\.")
		regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"

		if matched, _ := regexp.MatchString(regexPattern, domain); matched {
			return true
		}
	}
	return false
}

// httpGet performs HTTP GET request with timeout
func (s *TLDService) httpGet(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// SaveTLDResults saves discovered TLD results to the database
func (s *TLDService) SaveTLDResults(clientID string, result *TLDDiscoveryResult, scanID string) error {
	collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_domains")
	if err != nil {
		return fmt.Errorf("collection not found: %w", err)
	}

	for _, domain := range result.DiscoveredDomains {
		record := models.NewRecord(collection)
		record.Set("client", clientID)
		record.Set("domain", domain)
		record.Set("parent_domain", result.Domain)
		record.Set("source", "ms_tenant_discovery")
		record.Set("resolved", false) // Will be resolved later by subfinder
		record.Set("discovered_at", result.StartTime)
		record.Set("scan_id", scanID)

		// Add tenant metadata
		metadata := map[string]interface{}{
			"tenant_id":        result.TenantID,
			"tenant_name":      "",
			"discovery_method": "microsoft_tenant",
		}
		if result.TenantInfo != nil {
			metadata["tenant_name"] = result.TenantInfo.FederationBrandName
			metadata["namespace_type"] = result.TenantInfo.NameSpaceType
			metadata["is_federated"] = result.TenantInfo.IsFederated
		}
		record.Set("metadata", metadata)

		if err := s.app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save TLD result: %w", err)
		}
	}

	return nil
}

// GetSavedTLDs retrieves saved TLD results from the database
func (s *TLDService) GetSavedTLDs(clientID, domain string) ([]*models.Record, error) {
	filter := "client = {:client}"
	params := map[string]interface{}{
		"client": clientID,
	}

	if domain != "" {
		filter += " && (domain ~ {:domain} || parent_domain ~ {:parent_domain})"
		params["domain"] = domain
		params["parent_domain"] = domain
	}

	filter += " && source = 'ms_tenant_discovery'"

	records, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		filter,
		"created",
		0,
		-1,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve TLD results: %w", err)
	}

	return records, nil
}
