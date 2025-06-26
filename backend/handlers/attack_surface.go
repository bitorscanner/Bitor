package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bitor/services"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// AttackSurfaceHandlers handles all attack surface related endpoints
type AttackSurfaceHandlers struct {
	app              *pocketbase.PocketBase
	tldService       *services.TLDService
	netblockSvc      *services.NetblockService
	portScanSvc      *services.PortScanService
	subfinderService *services.SubfinderService
}

// NewAttackSurfaceHandlers creates a new instance of AttackSurfaceHandlers
func NewAttackSurfaceHandlers(app *pocketbase.PocketBase) *AttackSurfaceHandlers {
	return &AttackSurfaceHandlers{
		app:              app,
		tldService:       services.NewTLDService(app),
		netblockSvc:      services.NewNetblockService(app),
		portScanSvc:      services.NewPortScanService(app),
		subfinderService: services.NewSubfinderService(app),
	}
}

// SubdomainScanRequest represents the request to start a subdomain scan
type SubdomainScanRequest struct {
	Domain      string                 `json:"domain"`
	ClientID    string                 `json:"client_id"`
	Sources     []string               `json:"sources,omitempty"`
	AllSources  bool                   `json:"all_sources,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	MaxTime     int                    `json:"max_time,omitempty"`
	RateLimit   int                    `json:"rate_limit,omitempty"`
	Recursive   bool                   `json:"recursive,omitempty"`
	IncludeTLDs bool                   `json:"include_tlds,omitempty"`
	SaveResults bool                   `json:"save_results,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// TLDDiscoveryRequest represents the request to start TLD discovery
type TLDDiscoveryRequest struct {
	Domain      string                 `json:"domain"`
	ClientID    string                 `json:"client_id"`
	SaveResults bool                   `json:"save_results,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// NetblockDiscoveryRequest represents the request to start netblock discovery
type NetblockDiscoveryRequest struct {
	ClientID     string   `json:"client_id"`
	OrgNames     []string `json:"org_names"`
	CustomRanges []string `json:"custom_ranges,omitempty"`
	UseDomainIPs bool     `json:"use_domain_ips"`
	FilterCloud  bool     `json:"filter_cloud"`
}

// PortScanRequest represents the request to start a port scan
type PortScanRequest struct {
	ClientID         string   `json:"client_id"`
	TargetIPs        []string `json:"target_ips,omitempty"`       // Manual IP list
	IncludeDomains   bool     `json:"include_domains"`            // Include IPs from discovered domains
	IncludeNetblocks bool     `json:"include_netblocks"`          // Include IPs from netblocks
	Ports            string   `json:"ports,omitempty"`            // Custom ports (e.g., "80,443,8080-8090")
	TopPorts         string   `json:"top_ports,omitempty"`        // Top ports preset (100, 1000, full)
	ExcludePorts     string   `json:"exclude_ports,omitempty"`    // Ports to exclude
	ScanType         string   `json:"scan_type"`                  // "SYN" or "CONNECT"
	Rate             int      `json:"rate"`                       // Packets per second
	Threads          int      `json:"threads"`                    // Worker threads
	Timeout          int      `json:"timeout"`                    // Timeout in milliseconds
	Retries          int      `json:"retries"`                    // Number of retries
	HostDiscovery    bool     `json:"host_discovery"`             // Enable host discovery
	ExcludeCDN       bool     `json:"exclude_cdn"`                // Skip full scans for CDN/WAF
	Verify           bool     `json:"verify"`                     // Verify ports with TCP
	ExecutionMode    string   `json:"execution_mode"`             // "local" or "cloud"
	CloudProvider    string   `json:"cloud_provider,omitempty"`   // Cloud provider for remote execution
	NmapIntegration  bool     `json:"nmap_integration,omitempty"` // Run nmap for service detection
	NmapCommand      string   `json:"nmap_command,omitempty"`     // Custom nmap command
}

// Note: API key management is handled through the existing providers system
// No separate API key structures needed here

// HandleStartSubdomainScan starts a new subdomain enumeration scan
func (h *AttackSurfaceHandlers) HandleStartSubdomainScan(c echo.Context) error {
	fmt.Println("DEBUG: HandleStartSubdomainScan called")

	// Read raw body for debugging
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Printf("DEBUG: Failed to read body: %v\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}
	fmt.Printf("DEBUG: Raw request body: %s\n", string(body))

	// Reset body for binding
	c.Request().Body = io.NopCloser(strings.NewReader(string(body)))

	fmt.Printf("DEBUG: Content-Type: %s\n", c.Request().Header.Get("Content-Type"))

	var req SubdomainScanRequest
	if err := c.Bind(&req); err != nil {
		fmt.Printf("DEBUG: Bind error: %v\n", err)
		fmt.Printf("DEBUG: Error type: %T\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request format: %v", err),
		})
	}

	fmt.Printf("DEBUG: Parsed request: %+v\n", req)

	// Validate required fields
	if req.Domain == "" {
		fmt.Println("DEBUG: Domain is empty")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Domain is required",
		})
	}

	if req.ClientID == "" {
		fmt.Println("DEBUG: ClientID is empty")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Client ID is required",
		})
	}

	fmt.Printf("DEBUG: Validated request - Domain: %s, ClientID: %s\n", req.Domain, req.ClientID)

	// Prepare options
	options := make(map[string]interface{})
	if req.Sources != nil {
		options["sources"] = req.Sources
	}
	if req.AllSources {
		options["all_sources"] = req.AllSources
	}
	if req.Timeout > 0 {
		options["timeout"] = req.Timeout
	}
	if req.MaxTime > 0 {
		options["max_time"] = req.MaxTime
	}
	if req.RateLimit > 0 {
		options["rate_limit"] = req.RateLimit
	}
	if req.Recursive {
		options["recursive"] = req.Recursive
	}
	if req.IncludeTLDs {
		options["include_tlds"] = req.IncludeTLDs
	}
	// Merge additional options
	for k, v := range req.Options {
		options[k] = v
	}

	// Start scan with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Run subfinder scan
	result, err := h.subfinderService.RunSubfinder(ctx, req.Domain, req.ClientID, options)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":  "Subfinder scan failed",
			"result": result, // Include partial results if available
		})
	}

	// Save results if requested
	if req.SaveResults {
		fmt.Printf("DEBUG: SaveResults is true, attempting to save %d subdomains\n", len(result.Subdomains))
		if saveErr := h.subfinderService.SaveResults(req.ClientID, result, ""); saveErr != nil {
			fmt.Printf("DEBUG: Save error: %v\n", saveErr)
			result.Error = fmt.Sprintf("Scan completed but failed to save: %v", saveErr)
		} else {
			fmt.Println("DEBUG: Results saved successfully")
		}
	} else {
		fmt.Println("DEBUG: SaveResults is false, not saving to database")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// HandleStartTLDDiscovery starts a new top-level domain discovery scan
func (h *AttackSurfaceHandlers) HandleStartTLDDiscovery(c echo.Context) error {
	fmt.Println("DEBUG: HandleStartTLDDiscovery called")

	// Read raw body for debugging
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Printf("DEBUG: Failed to read body: %v\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}
	fmt.Printf("DEBUG: Raw request body: %s\n", string(body))

	// Reset body for binding
	c.Request().Body = io.NopCloser(strings.NewReader(string(body)))

	fmt.Printf("DEBUG: Content-Type: %s\n", c.Request().Header.Get("Content-Type"))

	var req TLDDiscoveryRequest
	if err := c.Bind(&req); err != nil {
		fmt.Printf("DEBUG: Bind error: %v\n", err)
		fmt.Printf("DEBUG: Error type: %T\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request format: %v", err),
		})
	}

	fmt.Printf("DEBUG: Parsed request: %+v\n", req)

	// Validate required fields
	if req.Domain == "" {
		fmt.Println("DEBUG: Domain is empty")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Domain is required",
		})
	}

	if req.ClientID == "" {
		fmt.Println("DEBUG: ClientID is empty")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Client ID is required",
		})
	}

	fmt.Printf("DEBUG: Validated request - Domain: %s, ClientID: %s\n", req.Domain, req.ClientID)

	// Start TLD discovery with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := h.tldService.DiscoverTLDs(ctx, req.Domain, req.ClientID, req.Options)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":  "TLD discovery failed",
			"result": result, // Include partial results if available
		})
	}

	// Save results if requested
	if req.SaveResults {
		fmt.Printf("DEBUG: SaveResults is true, attempting to save %d TLD domains\n", len(result.DiscoveredDomains))
		if saveErr := h.tldService.SaveTLDResults(req.ClientID, result, ""); saveErr != nil {
			fmt.Printf("DEBUG: Save error: %v\n", saveErr)
			result.Error = fmt.Sprintf("Discovery completed but failed to save: %v", saveErr)
		} else {
			fmt.Println("DEBUG: TLD results saved successfully")
		}
	} else {
		fmt.Println("DEBUG: SaveResults is false, not saving to database")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// HandleStartNetblockDiscovery starts netblock discovery
func (h *AttackSurfaceHandlers) HandleStartNetblockDiscovery(c echo.Context) error {
	var req NetblockDiscoveryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
	}

	if req.ClientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	if len(req.OrgNames) == 0 && len(req.CustomRanges) == 0 && !req.UseDomainIPs {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "At least one discovery method must be specified",
		})
	}

	// Convert to service request format
	serviceReq := services.NetblockDiscoveryRequest{
		ClientID:     req.ClientID,
		OrgNames:     req.OrgNames,
		CustomRanges: req.CustomRanges,
		UseDomainIPs: req.UseDomainIPs,
		FilterCloud:  req.FilterCloud,
	}

	// Start discovery
	result, err := h.netblockSvc.DiscoverNetblocks(c.Request().Context(), serviceReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Netblock discovery failed: %v", err),
		})
	}

	// Save results to database
	scanID := fmt.Sprintf("netblock_%s_%d", req.ClientID, time.Now().Unix())
	if err := h.netblockSvc.SaveResults(req.ClientID, result, scanID); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to save netblock results: %v\n", err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// HandleGetTLDs retrieves saved TLD discovery results
func (h *AttackSurfaceHandlers) HandleGetTLDs(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	domain := c.QueryParam("domain")

	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Client ID is required",
		})
	}

	tlds, err := h.tldService.GetSavedTLDs(clientID, domain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve TLD results",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"tlds":    tlds,
		"total":   len(tlds),
	})
}

// HandleGetSubdomains retrieves saved subdomains for a domain
func (h *AttackSurfaceHandlers) HandleGetSubdomains(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	domain := c.QueryParam("domain")

	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Client ID is required",
		})
	}

	// Get saved subdomains
	records, err := h.subfinderService.GetSavedSubdomains(clientID, domain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve subdomains",
		})
	}

	// Transform records to expected frontend format
	var subdomains []map[string]interface{}
	for _, record := range records {
		subdomain := map[string]interface{}{
			"id":            record.Id,
			"subdomain":     record.GetString("domain"), // Map domain -> subdomain for frontend
			"parent_domain": record.GetString("parent_domain"),
			"source":        record.GetString("source"),
			"resolved":      record.GetBool("resolved"),
			"ip":            record.GetString("ip_address"), // Map ip_address -> ip for frontend
			"discovered_at": record.GetDateTime("discovered_at"),
			"scan_id":       record.GetString("scan_id"),
			"created":       record.Created,
			"updated":       record.Updated,
		}
		subdomains = append(subdomains, subdomain)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"subdomains": subdomains,
		"total":      len(subdomains),
	})
}

// HandleGetAvailableSources retrieves available subfinder sources
func (h *AttackSurfaceHandlers) HandleGetAvailableSources(c echo.Context) error {
	sources := h.subfinderService.GetAvailableSources()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"sources": sources,
		"total":   len(sources),
	})
}

// HandleGetDomainStats returns statistics about domains in the attack surface
func (h *AttackSurfaceHandlers) HandleGetDomainStats(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Client ID is required",
		})
	}

	// Get total domains count
	totalDomains, err := h.app.Dao().FindRecordsByFilter(
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
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get domain statistics",
		})
	}

	// Count unique parent domains
	parentDomains := make(map[string]bool)
	resolvedCount := 0
	sourceStats := make(map[string]int)

	for _, record := range totalDomains {
		if parent := record.GetString("parent_domain"); parent != "" {
			parentDomains[parent] = true
		}
		if record.GetBool("resolved") {
			resolvedCount++
		}
		if source := record.GetString("source"); source != "" {
			sourceStats[source]++
		}
	}

	stats := map[string]interface{}{
		"total_subdomains": len(totalDomains),
		"unique_domains":   len(parentDomains),
		"resolved_count":   resolvedCount,
		"unresolved_count": len(totalDomains) - resolvedCount,
		"source_breakdown": sourceStats,
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// HandleGetNetblocks retrieves stored netblock results
func (h *AttackSurfaceHandlers) HandleGetNetblocks(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get netblocks
	netblocks, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_netblocks",
		"client = {:client}",
		"-discovered_at",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve netblocks",
		})
	}

	// Convert to response format
	var results []map[string]interface{}
	for _, record := range netblocks {
		// Parse matched criteria from JSON
		var matchedCriteria []string
		if criteriaStr := record.GetString("matched_criteria"); criteriaStr != "" {
			json.Unmarshal([]byte(criteriaStr), &matchedCriteria)
		}

		result := map[string]interface{}{
			"id":               record.Id,
			"ip":               record.GetString("ip"),
			"cidr":             record.GetString("cidr"),
			"asn":              record.GetInt("asn"),
			"asn_name":         record.GetString("asn_name"),
			"organization":     record.GetString("organization"),
			"confidence":       record.GetFloat("confidence"),
			"source":           record.GetString("source"),
			"org_name":         record.GetString("org_name"),
			"country":          record.GetString("country"),
			"matched_criteria": matchedCriteria,
			"discovered_at":    record.GetDateTime("discovered_at"),
			"created":          record.Created,
			"updated":          record.Updated,
		}
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"netblocks": results,
		"total":     len(results),
	})
}

// HandleGetIPs retrieves stored IP results
func (h *AttackSurfaceHandlers) HandleGetIPs(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get IPs
	ips, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_ips",
		"client = {:client}",
		"-discovered_at",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve IPs",
		})
	}

	// Convert to response format
	var results []map[string]interface{}
	for _, record := range ips {
		result := map[string]interface{}{
			"id":            record.Id,
			"ip":            record.GetString("ip"),
			"source":        record.GetString("source"),
			"source_domain": record.GetString("source_domain"),
			"discovered_at": record.GetDateTime("discovered_at"),
			"created":       record.Created,
			"updated":       record.Updated,
		}
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"ips":     results,
		"total":   len(results),
	})
}

// HandleGetNetblockStats retrieves netblock statistics
func (h *AttackSurfaceHandlers) HandleGetNetblockStats(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get netblock count - use FindRecordsByFilter since the collection might not exist
	netblocks, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_netblocks",
		"client = {:client}",
		"",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	netblockCount := 0
	highConfidenceCount := 0
	uniqueASNs := make(map[int]bool)

	if err == nil {
		netblockCount = len(netblocks)
		for _, record := range netblocks {
			if record.GetFloat("confidence") >= 0.8 {
				highConfidenceCount++
			}
			if asn := record.GetInt("asn"); asn > 0 {
				uniqueASNs[asn] = true
			}
		}
	}

	// Get IP count
	ips, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_ips",
		"client = {:client}",
		"",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	ipCount := 0
	if err == nil {
		ipCount = len(ips)
	}

	stats := map[string]interface{}{
		"total_netblocks": netblockCount,
		"total_ips":       ipCount,
		"high_confidence": highConfidenceCount,
		"unique_asns":     len(uniqueASNs),
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// HandleStartPortScan starts a new port scan asynchronously
func (h *AttackSurfaceHandlers) HandleStartPortScan(c echo.Context) error {
	var req PortScanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Invalid request format",
		})
	}

	if req.ClientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Validate that at least one target source is specified
	if len(req.TargetIPs) == 0 && !req.IncludeDomains && !req.IncludeNetblocks {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "At least one target source must be specified",
		})
	}

	// Set defaults
	if req.ScanType == "" {
		req.ScanType = "CONNECT"
	}
	if req.ExecutionMode == "" {
		req.ExecutionMode = "local"
	}
	if req.TopPorts == "" && req.Ports == "" {
		req.TopPorts = "100"
	}

	// Convert to service request format
	serviceReq := services.PortScanRequest{
		ClientID:         req.ClientID,
		TargetIPs:        req.TargetIPs,
		IncludeDomains:   req.IncludeDomains,
		IncludeNetblocks: req.IncludeNetblocks,
		Ports:            req.Ports,
		TopPorts:         req.TopPorts,
		ExcludePorts:     req.ExcludePorts,
		ScanType:         req.ScanType,
		Rate:             req.Rate,
		Threads:          req.Threads,
		Timeout:          req.Timeout,
		Retries:          req.Retries,
		HostDiscovery:    req.HostDiscovery,
		ExcludeCDN:       req.ExcludeCDN,
		Verify:           req.Verify,
		ExecutionMode:    req.ExecutionMode,
		CloudProvider:    req.CloudProvider,
		NmapIntegration:  req.NmapIntegration,
		NmapCommand:      req.NmapCommand,
	}

	// Start the scan asynchronously
	scanID, err := h.portScanSvc.StartAsyncPortScan(serviceReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to start port scan: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"scan_id": scanID,
		"message": "Port scan started successfully",
	})
}

// HandleGetPortScanProgress retrieves the progress of a running port scan
func (h *AttackSurfaceHandlers) HandleGetPortScanProgress(c echo.Context) error {
	scanID := c.PathParam("scan_id")
	if scanID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Scan ID is required",
		})
	}

	progress, err := h.portScanSvc.GetScanProgress(scanID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Scan not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"progress": progress,
	})
}

// HandleGetPorts retrieves stored port scan results
func (h *AttackSurfaceHandlers) HandleGetPorts(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get ports
	ports, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_ports",
		"client = {:client}",
		"-discovered_at",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve ports",
		})
	}

	// Convert to response format
	var results []map[string]interface{}
	for _, record := range ports {
		result := map[string]interface{}{
			"id":            record.Id,
			"scan_id":       record.GetString("scan_id"),
			"ip":            record.GetString("ip"),
			"port":          record.GetInt("port"),
			"protocol":      record.GetString("protocol"),
			"service":       record.GetString("service"),
			"state":         record.GetString("state"),
			"host":          record.GetString("host"),
			"source":        record.GetString("source"),
			"discovered_at": record.GetDateTime("discovered_at"),
			"created":       record.Created,
			"updated":       record.Updated,
		}
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"ports":   results,
		"total":   len(results),
	})
}

// HandleGetPortScans retrieves port scan job history
func (h *AttackSurfaceHandlers) HandleGetPortScans(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get port scans
	scans, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_port_scans",
		"client = {:client}",
		"-start_time",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve port scans",
		})
	}

	// Convert to response format
	var results []map[string]interface{}
	for _, record := range scans {
		// Parse stats and target IPs from JSON
		var stats map[string]int
		var targetIPs []string
		if statsStr := record.GetString("stats"); statsStr != "" {
			json.Unmarshal([]byte(statsStr), &stats)
		}
		if targetsStr := record.GetString("target_ips"); targetsStr != "" {
			json.Unmarshal([]byte(targetsStr), &targetIPs)
		}

		result := map[string]interface{}{
			"id":             record.Id,
			"scan_id":        record.GetString("scan_id"),
			"start_time":     record.GetDateTime("start_time"),
			"end_time":       record.GetDateTime("end_time"),
			"duration":       record.GetString("duration"),
			"total_targets":  record.GetInt("total_targets"),
			"total_ports":    record.GetInt("total_ports"),
			"open_ports":     record.GetInt("open_ports"),
			"execution_mode": record.GetString("execution_mode"),
			"cloud_provider": record.GetString("cloud_provider"),
			"naabu_version":  record.GetString("naabu_version"),
			"error":          record.GetString("error"),
			"stats":          stats,
			"target_ips":     targetIPs,
			"created":        record.Created,
			"updated":        record.Updated,
		}
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"scans":   results,
		"total":   len(results),
	})
}

// HandleGetPortStats retrieves port scan statistics
func (h *AttackSurfaceHandlers) HandleGetPortStats(c echo.Context) error {
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Client ID is required",
		})
	}

	// Get total ports count
	totalPorts, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_ports",
		"client = {:client}",
		"",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to get port statistics",
		})
	}

	// Get unique IPs with open ports
	uniqueIPs := make(map[string]bool)
	portStats := make(map[int]int)
	serviceStats := make(map[string]int)
	sourceStats := make(map[string]int)

	for _, record := range totalPorts {
		ip := record.GetString("ip")
		port := record.GetInt("port")
		service := record.GetString("service")
		source := record.GetString("source")

		uniqueIPs[ip] = true
		portStats[port]++
		if service != "" {
			serviceStats[service]++
		}
		if source != "" {
			sourceStats[source]++
		}
	}

	// Get scan count
	scans, err := h.app.Dao().FindRecordsByFilter(
		"attack_surface_port_scans",
		"client = {:client}",
		"",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Failed to get scan count",
		})
	}

	// Find most recent scan
	var latestScan map[string]interface{}
	if len(scans) > 0 {
		latest := scans[0] // Already ordered by -start_time
		var stats map[string]int
		if statsStr := latest.GetString("stats"); statsStr != "" {
			json.Unmarshal([]byte(statsStr), &stats)
		}

		latestScan = map[string]interface{}{
			"scan_id":        latest.GetString("scan_id"),
			"start_time":     latest.GetDateTime("start_time"),
			"end_time":       latest.GetDateTime("end_time"),
			"duration":       latest.GetString("duration"),
			"total_targets":  latest.GetInt("total_targets"),
			"open_ports":     latest.GetInt("open_ports"),
			"execution_mode": latest.GetString("execution_mode"),
			"stats":          stats,
		}
	}

	// Top 5 ports
	type portCount struct {
		Port  int `json:"port"`
		Count int `json:"count"`
	}
	var topPorts []portCount
	for port, count := range portStats {
		topPorts = append(topPorts, portCount{Port: port, Count: count})
	}
	// Sort by count descending
	for i := 0; i < len(topPorts)-1; i++ {
		for j := i + 1; j < len(topPorts); j++ {
			if topPorts[j].Count > topPorts[i].Count {
				topPorts[i], topPorts[j] = topPorts[j], topPorts[i]
			}
		}
	}
	if len(topPorts) > 5 {
		topPorts = topPorts[:5]
	}

	stats := map[string]interface{}{
		"total_open_ports":  len(totalPorts),
		"unique_hosts":      len(uniqueIPs),
		"total_scans":       len(scans),
		"top_ports":         topPorts,
		"service_breakdown": serviceStats,
		"source_breakdown":  sourceStats,
		"latest_scan":       latestScan,
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// Register registers all attack surface routes
func (h *AttackSurfaceHandlers) Register(app *pocketbase.PocketBase) {
	fmt.Println("DEBUG: AttackSurfaceHandlers.Register() called")

	// Try registering in the existing OnBeforeServe event instead of adding a new one
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		fmt.Println("DEBUG: OnBeforeServe callback executing for attack surface")

		// Add a simple test endpoint first
		e.Router.GET("/api/test-attack-surface", func(c echo.Context) error {
			fmt.Println("DEBUG: Test endpoint called!")
			return c.JSON(200, map[string]string{"message": "Attack surface handler is working!"})
		})

		// Debug: Register routes directly first to test
		e.Router.GET("/api/attack-surface/sources", func(c echo.Context) error {
			fmt.Println("DEBUG: Sources endpoint called!")
			return h.HandleGetAvailableSources(c)
		})
		e.Router.GET("/api/attack-surface/subdomains", func(c echo.Context) error {
			fmt.Println("DEBUG: Subdomains endpoint called!")
			return h.HandleGetSubdomains(c)
		})
		e.Router.GET("/api/attack-surface/subdomains/stats", func(c echo.Context) error {
			fmt.Println("DEBUG: Stats endpoint called!")
			return h.HandleGetDomainStats(c)
		})
		e.Router.POST("/api/attack-surface/subdomains/scan", func(c echo.Context) error {
			fmt.Println("DEBUG: Scan endpoint called!")
			return h.HandleStartSubdomainScan(c)
		})
		e.Router.POST("/api/attack-surface/tld/discover", func(c echo.Context) error {
			fmt.Println("DEBUG: TLD Discovery endpoint called!")
			return h.HandleStartTLDDiscovery(c)
		})
		e.Router.GET("/api/attack-surface/tld", func(c echo.Context) error {
			fmt.Println("DEBUG: TLD Get endpoint called!")
			return h.HandleGetTLDs(c)
		})
		e.Router.POST("/api/attack-surface/netblock/discover", func(c echo.Context) error {
			fmt.Println("DEBUG: Netblock Discovery endpoint called!")
			return h.HandleStartNetblockDiscovery(c)
		})
		e.Router.GET("/api/attack-surface/netblocks", func(c echo.Context) error {
			fmt.Println("DEBUG: Netblocks Get endpoint called!")
			return h.HandleGetNetblocks(c)
		})
		e.Router.GET("/api/attack-surface/ips", func(c echo.Context) error {
			fmt.Println("DEBUG: IPs Get endpoint called!")
			return h.HandleGetIPs(c)
		})
		e.Router.GET("/api/attack-surface/netblocks/stats", func(c echo.Context) error {
			fmt.Println("DEBUG: Netblock Stats endpoint called!")
			return h.HandleGetNetblockStats(c)
		})
		e.Router.POST("/api/attack-surface/ports/scan", func(c echo.Context) error {
			fmt.Println("DEBUG: Port Scan endpoint called!")
			return h.HandleStartPortScan(c)
		})
		e.Router.GET("/api/attack-surface/ports", func(c echo.Context) error {
			fmt.Println("DEBUG: Ports Get endpoint called!")
			return h.HandleGetPorts(c)
		})
		e.Router.GET("/api/attack-surface/ports/scans", func(c echo.Context) error {
			fmt.Println("DEBUG: Port Scans Get endpoint called!")
			return h.HandleGetPortScans(c)
		})
		e.Router.GET("/api/attack-surface/ports/stats", func(c echo.Context) error {
			fmt.Println("DEBUG: Port Stats endpoint called!")
			return h.HandleGetPortStats(c)
		})

		fmt.Println("DEBUG: Attack surface routes registered successfully")
		fmt.Println("DEBUG: GET /api/test-attack-surface -> simple test")
		fmt.Println("DEBUG: GET /api/attack-surface/sources -> HandleGetAvailableSources")
		fmt.Println("DEBUG: GET /api/attack-surface/subdomains -> HandleGetSubdomains")
		fmt.Println("DEBUG: GET /api/attack-surface/subdomains/stats -> HandleGetDomainStats")
		fmt.Println("DEBUG: POST /api/attack-surface/subdomains/scan -> HandleStartSubdomainScan")
		fmt.Println("DEBUG: POST /api/attack-surface/tld/discover -> HandleStartTLDDiscovery")
		fmt.Println("DEBUG: GET /api/attack-surface/tld -> HandleGetTLDs")
		fmt.Println("DEBUG: POST /api/attack-surface/netblock/discover -> HandleStartNetblockDiscovery")
		fmt.Println("DEBUG: GET /api/attack-surface/netblocks -> HandleGetNetblocks")
		fmt.Println("DEBUG: GET /api/attack-surface/ips -> HandleGetIPs")
		fmt.Println("DEBUG: GET /api/attack-surface/netblocks/stats -> HandleGetNetblockStats")
		fmt.Println("DEBUG: POST /api/attack-surface/ports/scan -> HandleStartPortScan")
		fmt.Println("DEBUG: GET /api/attack-surface/ports -> HandleGetPorts")
		fmt.Println("DEBUG: GET /api/attack-surface/ports/scans -> HandleGetPortScans")
		fmt.Println("DEBUG: GET /api/attack-surface/ports/stats -> HandleGetPortStats")

		return nil
	})
}
