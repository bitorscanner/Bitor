package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// URLScanService handles URL scanning operations using httpx
type URLScanService struct {
	app    *pocketbase.PocketBase
	logger *log.Logger
	// Track running scans
	activeScans map[string]*URLScanProgress
	mutex       sync.RWMutex
}

// URLScanProgress tracks the progress of a running URL scan
type URLScanProgress struct {
	ScanID       string     `json:"scan_id"`
	ClientID     string     `json:"client_id"`
	Status       string     `json:"status"`   // "running", "completed", "failed"
	Progress     int        `json:"progress"` // 0-100
	Message      string     `json:"message"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	TotalTargets int        `json:"total_targets"`
	LiveURLs     int        `json:"live_urls"`
	Error        string     `json:"error,omitempty"`
}

// URLScanRequest represents a URL scan request
type URLScanRequest struct {
	ClientID          string   `json:"client_id"`
	TargetURLs        []string `json:"target_urls,omitempty"`    // Manual URL list
	IncludePorts      bool     `json:"include_ports"`            // Include URLs from port scan results
	IncludeDomains    bool     `json:"include_domains"`          // Include URLs from discovered domains
	IncludeSubdomains bool     `json:"include_subdomains"`       // Include URLs from subdomains
	Schemes           []string `json:"schemes,omitempty"`        // URL schemes (http, https)
	Ports             []int    `json:"ports,omitempty"`          // Specific ports to scan
	OnlyWebPorts      bool     `json:"only_web_ports"`           // Only scan common web ports
	Threads           int      `json:"threads"`                  // Worker threads
	Timeout           int      `json:"timeout"`                  // Request timeout in seconds
	Retries           int      `json:"retries"`                  // Number of retries
	FollowRedirects   bool     `json:"follow_redirects"`         // Follow HTTP redirects
	TechDetection     bool     `json:"tech_detection"`           // Enable technology detection
	StatusCode        bool     `json:"status_code"`              // Include status codes
	ContentLength     bool     `json:"content_length"`           // Include content length
	ResponseTime      bool     `json:"response_time"`            // Include response time
	MatchRegex        string   `json:"match_regex,omitempty"`    // Custom regex to match
	FilterRegex       string   `json:"filter_regex,omitempty"`   // Custom regex to filter
	OutputAll         bool     `json:"output_all"`               // Output all URLs (even failed)
	Silent            bool     `json:"silent"`                   // Silent mode
	ExecutionMode     string   `json:"execution_mode"`           // "local" or "cloud"
	CloudProvider     string   `json:"cloud_provider,omitempty"` // Cloud provider for remote execution
}

// URLScanResult represents a discovered URL
type URLScanResult struct {
	URL           string            `json:"url"`
	StatusCode    int               `json:"status_code"`
	ContentLength int               `json:"content_length"`
	ResponseTime  string            `json:"response_time"`
	Title         string            `json:"title,omitempty"`
	Technologies  []string          `json:"technologies,omitempty"`
	Server        string            `json:"server,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	Location      string            `json:"location,omitempty"`
	FinalURL      string            `json:"final_url,omitempty"`
	Scheme        string            `json:"scheme"`
	Host          string            `json:"host"`
	Port          int               `json:"port"`
	Path          string            `json:"path"`
	Source        string            `json:"source"` // "ports", "domains", "subdomains", "manual"
	Hash          map[string]string `json:"hash,omitempty"`
	CDN           string            `json:"cdn,omitempty"`
	WebServer     string            `json:"webserver,omitempty"`
	IP            string            `json:"ip,omitempty"`
	CNames        []string          `json:"cnames,omitempty"`
	Chain         []string          `json:"chain,omitempty"`
}

// URLScanJobResult represents the complete result of a URL scan
type URLScanJobResult struct {
	ClientID      string          `json:"client_id"`
	ScanID        string          `json:"scan_id"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Duration      string          `json:"duration"`
	TotalTargets  int             `json:"total_targets"`
	LiveURLs      int             `json:"live_urls"`
	Results       []URLScanResult `json:"results"`
	Stats         map[string]int  `json:"stats"`
	ExecutionMode string          `json:"execution_mode"`
	CloudProvider string          `json:"cloud_provider,omitempty"`
	HttpxVersion  string          `json:"httpx_version,omitempty"`
	Error         string          `json:"error,omitempty"`
}

// HttpxOutput represents httpx JSON output format
type HttpxOutput struct {
	URL           string            `json:"url"`
	StatusCode    int               `json:"status_code"`
	ContentLength int               `json:"content_length"`
	ResponseTime  string            `json:"response_time"`
	Title         string            `json:"title,omitempty"`
	Technologies  []string          `json:"tech,omitempty"`
	Server        string            `json:"webserver,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	Location      string            `json:"location,omitempty"`
	FinalURL      string            `json:"final_url,omitempty"`
	Hash          map[string]string `json:"hash,omitempty"`
	CDN           bool              `json:"cdn,omitempty"`      // httpx outputs boolean, not string
	CDNName       string            `json:"cdn_name,omitempty"` // actual CDN name
	CDNType       string            `json:"cdn_type,omitempty"` // CDN type
	WebServer     string            `json:"webserver,omitempty"`
	IP            string            `json:"host,omitempty"`
	CNames        []string          `json:"cnames,omitempty"`
	Chain         []string          `json:"chain,omitempty"`
	Failed        bool              `json:"failed,omitempty"`    // httpx also outputs this as boolean
	Port          string            `json:"port,omitempty"`      // httpx outputs port as string
	Timestamp     string            `json:"timestamp,omitempty"` // httpx includes timestamp
	Method        string            `json:"method,omitempty"`    // HTTP method
	Input         string            `json:"input,omitempty"`     // Original input URL
	Time          string            `json:"time,omitempty"`      // Response time (different from response_time)
	Words         int               `json:"words,omitempty"`     // Word count
	Lines         int               `json:"lines,omitempty"`     // Line count
}

// AttackSurfaceTargetRequest represents a request for collecting attack surface targets for nuclei
type AttackSurfaceTargetRequest struct {
	ClientID          string   `json:"client_id"`
	IncludeDomains    bool     `json:"include_domains"`    // Include discovered domains
	IncludeSubdomains bool     `json:"include_subdomains"` // Include discovered subdomains
	IncludePorts      bool     `json:"include_ports"`      // Include IPs with open ports
	IncludeNetblocks  bool     `json:"include_netblocks"`  // Include IPs from netblocks
	IncludeURLs       bool     `json:"include_urls"`       // Include discovered URLs
	Schemes           []string `json:"schemes,omitempty"`  // URL schemes for domains/IPs (http, https)
	Ports             []int    `json:"ports,omitempty"`    // Specific ports to include for IPs/domains
	OnlyWebPorts      bool     `json:"only_web_ports"`     // Only include common web ports
	ManualTargets     []string `json:"manual_targets"`     // Additional manual targets
}

// AttackSurfaceTargetResult represents the collected targets from attack surface
type AttackSurfaceTargetResult struct {
	ClientID     string              `json:"client_id"`
	TotalTargets int                 `json:"total_targets"`
	Targets      []string            `json:"targets"`
	Sources      map[string][]string `json:"sources"`
	Stats        map[string]int      `json:"stats"`
}

// NewURLScanService creates a new instance of URLScanService
func NewURLScanService(app *pocketbase.PocketBase) *URLScanService {
	return &URLScanService{
		app:         app,
		logger:      log.New(log.Writer(), "[URLScanService] ", log.LstdFlags),
		activeScans: make(map[string]*URLScanProgress),
	}
}

// createUserMessage creates a user notification message for scan events
func (s *URLScanService) createUserMessage(clientID, scanID, message, messageType string) error {
	// Get the current authenticated user from the context
	// For now, we'll get the client creator, but ideally this should come from the request context
	clientRecord, err := s.app.Dao().FindRecordById("clients", clientID)
	if err != nil {
		s.logger.Printf("Failed to find client %s: %v", clientID, err)
		// If we can't find the client, try to create a general admin notification
		return s.createAdminMessage(scanID, message, messageType)
	}

	// Get the user who owns/created the client
	userID := clientRecord.GetString("created_by")
	if userID == "" {
		s.logger.Printf("No user found for client %s, creating admin message", clientID)
		return s.createAdminMessage(scanID, message, messageType)
	}

	// Create user message
	collection, err := s.app.Dao().FindCollectionByNameOrId("user_messages")
	if err != nil {
		return fmt.Errorf("failed to find user_messages collection: %v", err)
	}

	record := models.NewRecord(collection)
	record.Set("message", fmt.Sprintf("URL Scan: %s", message))
	record.Set("type", messageType)
	record.Set("read", false)

	// Try to find if this is an admin user first
	_, err = s.app.Dao().FindRecordById("_pb_users_auth_", userID)
	if err == nil {
		// It's an admin user
		record.Set("admin_id", userID)
	} else {
		// It's a regular user
		record.Set("user", userID)
	}

	if err := s.app.Dao().SaveRecord(record); err != nil {
		s.logger.Printf("Failed to save user message: %v", err)
		// Fallback to admin message
		return s.createAdminMessage(scanID, message, messageType)
	}

	s.logger.Printf("Created user message for scan %s: %s", scanID, message)
	return nil
}

// createAdminMessage creates a notification for admin users when user-specific notification fails
func (s *URLScanService) createAdminMessage(scanID, message, messageType string) error {
	collection, err := s.app.Dao().FindCollectionByNameOrId("user_messages")
	if err != nil {
		return fmt.Errorf("failed to find user_messages collection: %v", err)
	}

	// Get the first admin user
	adminRecords, err := s.app.Dao().FindRecordsByFilter("_pb_users_auth_", "", "", 1, 0)
	if err != nil || len(adminRecords) == 0 {
		s.logger.Printf("No admin users found for notification")
		return fmt.Errorf("no admin users found")
	}

	record := models.NewRecord(collection)
	record.Set("message", fmt.Sprintf("URL Scan: %s", message))
	record.Set("type", messageType)
	record.Set("read", false)
	record.Set("admin_id", adminRecords[0].Id)

	if err := s.app.Dao().SaveRecord(record); err != nil {
		return fmt.Errorf("failed to save admin message: %v", err)
	}

	s.logger.Printf("Created admin message for scan %s: %s", scanID, message)
	return nil
}

// StartAsyncURLScan starts a URL scan in the background and returns the scan ID
func (s *URLScanService) StartAsyncURLScan(req URLScanRequest) (string, error) {
	scanID := fmt.Sprintf("urlscan_%s_%d", req.ClientID, time.Now().Unix())

	// Initialize progress tracking
	s.mutex.Lock()
	s.activeScans[scanID] = &URLScanProgress{
		ScanID:    scanID,
		ClientID:  req.ClientID,
		Status:    "starting",
		Progress:  0,
		Message:   "Preparing URL scan...",
		StartTime: time.Now(),
	}
	s.mutex.Unlock()

	// Create initial notification
	s.createUserMessage(req.ClientID, scanID, "started successfully", "info")

	// Start scan in goroutine
	go func() {
		defer func() {
			s.mutex.Lock()
			if scan, exists := s.activeScans[scanID]; exists {
				if scan.Status == "running" {
					scan.Status = "completed"
					scan.Progress = 100
					endTime := time.Now()
					scan.EndTime = &endTime
				}
			}
			s.mutex.Unlock()
		}()

		// Update status to running
		s.updateScanProgress(scanID, "running", 10, "Starting URL scan...")

		// Run the actual scan
		ctx := context.Background()
		result, err := s.RunURLScan(ctx, req)
		if err != nil {
			s.updateScanProgress(scanID, "failed", 0, fmt.Sprintf("Scan failed: %v", err))
			// Create failure notification
			s.createUserMessage(req.ClientID, scanID, fmt.Sprintf("failed with error: %v", err), "error")
			return
		}

		// Save results to database
		s.updateScanProgress(scanID, "running", 90, "Saving results...")
		if err := s.SaveResults(req.ClientID, result, scanID); err != nil {
			s.logger.Printf("Failed to save URL scan results: %v", err)
			// Still notify about completion even if save failed
			s.createUserMessage(req.ClientID, scanID, fmt.Sprintf("completed but failed to save results: %v", err), "warning")
		} else {
			// Create completion notification only if save was successful
			s.createUserMessage(req.ClientID, scanID, fmt.Sprintf("completed successfully! Found %d live URLs from %d targets", result.LiveURLs, result.TotalTargets), "success")
		}

		s.updateScanProgress(scanID, "completed", 100, fmt.Sprintf("Scan completed. Found %d live URLs", result.LiveURLs))
	}()

	return scanID, nil
}

// GetScanProgress returns the current progress of a scan
func (s *URLScanService) GetScanProgress(scanID string) (*URLScanProgress, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if progress, exists := s.activeScans[scanID]; exists {
		return progress, nil
	}

	return nil, fmt.Errorf("scan not found")
}

// updateScanProgress updates the progress of a running scan
func (s *URLScanService) updateScanProgress(scanID, status string, progress int, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if scan, exists := s.activeScans[scanID]; exists {
		scan.Status = status
		scan.Progress = progress
		scan.Message = message
		if status == "completed" || status == "failed" {
			endTime := time.Now()
			scan.EndTime = &endTime
		}
	}
}

// CollectTargetURLs collects target URLs from various sources
func (s *URLScanService) CollectTargetURLs(clientID string, includeports, includeDomains, includeSubdomains bool, manualURLs []string, schemes []string, ports []int, onlyWebPorts bool) ([]string, map[string][]string, error) {
	var allURLs []string
	sources := make(map[string][]string)
	seen := make(map[string]bool)

	// Default schemes if not specified
	if len(schemes) == 0 {
		schemes = []string{"http", "https"}
	}

	// Default web ports if only web ports is enabled
	webPorts := []int{80, 443, 8080, 8443, 8000, 8888, 9000, 9001, 3000, 5000}
	if onlyWebPorts && len(ports) == 0 {
		ports = webPorts
	}

	// Add manual URLs
	if len(manualURLs) > 0 {
		var validURLs []string
		for _, url := range manualURLs {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			if !seen[url] {
				validURLs = append(validURLs, url)
				seen[url] = true
			}
		}
		sources["manual"] = validURLs
		allURLs = append(allURLs, validURLs...)
	}

	// Collect URLs from port scan results
	if includeports {
		portURLs, err := s.collectPortURLs(clientID, schemes, ports)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect port URLs: %v", err)
		} else {
			var uniquePortURLs []string
			for _, url := range portURLs {
				if !seen[url] {
					uniquePortURLs = append(uniquePortURLs, url)
					seen[url] = true
				}
			}
			sources["ports"] = uniquePortURLs
			allURLs = append(allURLs, uniquePortURLs...)
		}
	}

	// Collect URLs from discovered domains
	if includeDomains {
		domainURLs, err := s.collectDomainURLs(clientID, schemes, ports)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect domain URLs: %v", err)
		} else {
			var uniqueDomainURLs []string
			for _, url := range domainURLs {
				if !seen[url] {
					uniqueDomainURLs = append(uniqueDomainURLs, url)
					seen[url] = true
				}
			}
			sources["domains"] = uniqueDomainURLs
			allURLs = append(allURLs, uniqueDomainURLs...)
		}
	}

	// Collect URLs from subdomains
	if includeSubdomains {
		subdomainURLs, err := s.collectSubdomainURLs(clientID, schemes, ports)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect subdomain URLs: %v", err)
		} else {
			var uniqueSubdomainURLs []string
			for _, url := range subdomainURLs {
				if !seen[url] {
					uniqueSubdomainURLs = append(uniqueSubdomainURLs, url)
					seen[url] = true
				}
			}
			sources["subdomains"] = uniqueSubdomainURLs
			allURLs = append(allURLs, uniqueSubdomainURLs...)
		}
	}

	// Sort URLs for consistent output
	sort.Strings(allURLs)

	s.logger.Printf("Collected %d unique target URLs: %d manual, %d from ports, %d from domains, %d from subdomains",
		len(allURLs), len(sources["manual"]), len(sources["ports"]), len(sources["domains"]), len(sources["subdomains"]))

	return allURLs, sources, nil
}

// collectPortURLs gets URLs from port scan results
func (s *URLScanService) collectPortURLs(clientID string, schemes []string, ports []int) ([]string, error) {
	// Get all open ports for this client
	portRecords, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_ports",
		"client = {:client}",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get port records: %v", err)
	}

	var urls []string
	seen := make(map[string]bool)

	for _, portRecord := range portRecords {
		ip := portRecord.GetString("ip")
		port := portRecord.GetInt("port")

		if ip == "" || port == 0 {
			continue
		}

		// Filter by specific ports if specified
		if len(ports) > 0 {
			found := false
			for _, p := range ports {
				if p == port {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Generate URLs for each scheme
		for _, scheme := range schemes {
			var url string
			if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
				url = fmt.Sprintf("%s://%s", scheme, ip)
			} else {
				url = fmt.Sprintf("%s://%s:%d", scheme, ip, port)
			}

			if !seen[url] {
				urls = append(urls, url)
				seen[url] = true
			}
		}
	}

	return urls, nil
}

// collectDomainURLs gets URLs from discovered domains
func (s *URLScanService) collectDomainURLs(clientID string, schemes []string, ports []int) ([]string, error) {
	// Get all domains for this client (parent domains only)
	domains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client} && parent_domain = ''",
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

	var urls []string
	seen := make(map[string]bool)

	for _, domainRecord := range domains {
		domain := domainRecord.GetString("domain")
		if domain == "" {
			continue
		}

		// Generate URLs for each scheme and port combination
		for _, scheme := range schemes {
			if len(ports) > 0 {
				for _, port := range ports {
					var url string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						url = fmt.Sprintf("%s://%s", scheme, domain)
					} else {
						url = fmt.Sprintf("%s://%s:%d", scheme, domain, port)
					}

					if !seen[url] {
						urls = append(urls, url)
						seen[url] = true
					}
				}
			} else {
				// Use default ports
				url := fmt.Sprintf("%s://%s", scheme, domain)
				if !seen[url] {
					urls = append(urls, url)
					seen[url] = true
				}
			}
		}
	}

	return urls, nil
}

// collectSubdomainURLs gets URLs from discovered subdomains
func (s *URLScanService) collectSubdomainURLs(clientID string, schemes []string, ports []int) ([]string, error) {
	// Get all subdomains for this client
	subdomains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client} && parent_domain != ''",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get subdomains: %v", err)
	}

	var urls []string
	seen := make(map[string]bool)

	for _, subdomainRecord := range subdomains {
		subdomain := subdomainRecord.GetString("domain")
		if subdomain == "" {
			continue
		}

		// Generate URLs for each scheme and port combination
		for _, scheme := range schemes {
			if len(ports) > 0 {
				for _, port := range ports {
					var url string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						url = fmt.Sprintf("%s://%s", scheme, subdomain)
					} else {
						url = fmt.Sprintf("%s://%s:%d", scheme, subdomain, port)
					}

					if !seen[url] {
						urls = append(urls, url)
						seen[url] = true
					}
				}
			} else {
				// Use default ports
				url := fmt.Sprintf("%s://%s", scheme, subdomain)
				if !seen[url] {
					urls = append(urls, url)
					seen[url] = true
				}
			}
		}
	}

	return urls, nil
}

// RunURLScan executes httpx URL scan locally or in the cloud
func (s *URLScanService) RunURLScan(ctx context.Context, req URLScanRequest) (*URLScanJobResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("urlscan_%s_%d", req.ClientID, startTime.Unix())

	result := &URLScanJobResult{
		ClientID:      req.ClientID,
		ScanID:        scanID,
		StartTime:     startTime,
		ExecutionMode: req.ExecutionMode,
		CloudProvider: req.CloudProvider,
		Stats:         make(map[string]int),
	}

	// Ensure httpx is installed (for local scans)
	if req.ExecutionMode != "cloud" {
		if err := s.ensureHttpxInstalled(); err != nil {
			result.Error = fmt.Sprintf("Failed to ensure httpx is installed: %v", err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime).String()
			return result, err
		}
	}

	// Collect target URLs
	targetURLs, sources, err := s.CollectTargetURLs(req.ClientID, req.IncludePorts, req.IncludeDomains, req.IncludeSubdomains, req.TargetURLs, req.Schemes, req.Ports, req.OnlyWebPorts)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to collect target URLs: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(targetURLs) == 0 {
		result.Error = "No target URLs found"
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, fmt.Errorf("no target URLs found")
	}

	result.TotalTargets = len(targetURLs)

	s.logger.Printf("Starting URL scan: %d targets", len(targetURLs))

	// Execute scan based on mode
	var scanResults []URLScanResult
	if req.ExecutionMode == "cloud" {
		scanResults, err = s.runCloudScan(ctx, req, targetURLs)
	} else {
		scanResults, err = s.runLocalScan(ctx, req, targetURLs)
	}

	if err != nil {
		result.Error = fmt.Sprintf("URL scan failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Enhance results with source information
	for i := range scanResults {
		scanResults[i].Source = s.determineURLSource(scanResults[i].URL, sources)
	}

	result.Results = scanResults
	result.LiveURLs = len(scanResults)
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	// Calculate statistics
	result.Stats = s.calculateScanStats(scanResults, sources)

	s.logger.Printf("URL scan completed: %d live URLs found in %s", result.LiveURLs, result.Duration)

	return result, nil
}

// runLocalScan executes httpx locally
func (s *URLScanService) runLocalScan(ctx context.Context, req URLScanRequest, targetURLs []string) ([]URLScanResult, error) {
	// Create temporary target file
	targetFile, err := s.createTargetFile(targetURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to create target file: %v", err)
	}
	defer os.Remove(targetFile)

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "httpx_output_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Build httpx command
	args := s.buildHttpxArgs(req, targetFile, outputFile.Name())

	s.logger.Printf("Running httpx command: httpx %s", strings.Join(args, " "))

	// Execute httpx
	cmd := exec.CommandContext(ctx, "httpx", args...)

	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start httpx: %v", err)
	}

	// Log stderr output
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.logger.Printf("httpx: %s", scanner.Text())
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("httpx execution failed: %v", err)
	}

	// Parse results
	results, err := s.parseHttpxOutput(outputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse httpx output: %v", err)
	}

	return results, nil
}

// runCloudScan executes httpx on a cloud instance
func (s *URLScanService) runCloudScan(ctx context.Context, req URLScanRequest, targetURLs []string) ([]URLScanResult, error) {
	// TODO: Implement cloud execution
	return nil, fmt.Errorf("cloud scanning not yet implemented")
}

// buildHttpxArgs constructs command line arguments for httpx
func (s *URLScanService) buildHttpxArgs(req URLScanRequest, targetFile, outputFile string) []string {
	var args []string

	// Input file
	args = append(args, "-list", targetFile)

	// Output format
	args = append(args, "-json", "-o", outputFile)

	// Performance options
	if req.Threads > 0 {
		args = append(args, "-threads", fmt.Sprintf("%d", req.Threads))
	}
	if req.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", req.Timeout))
	}
	if req.Retries > 0 {
		args = append(args, "-retries", fmt.Sprintf("%d", req.Retries))
	}

	// Follow redirects
	if req.FollowRedirects {
		args = append(args, "-follow-redirects")
	}

	// Technology detection
	if req.TechDetection {
		args = append(args, "-tech-detect")
	}

	// Response information
	if req.StatusCode {
		args = append(args, "-status-code")
	}
	if req.ContentLength {
		args = append(args, "-content-length")
	}
	if req.ResponseTime {
		args = append(args, "-response-time")
	}

	// Additional response data
	args = append(args, "-title", "-server", "-content-type", "-location")

	// Regex filters
	if req.MatchRegex != "" {
		args = append(args, "-match-regex", req.MatchRegex)
	}
	if req.FilterRegex != "" {
		args = append(args, "-filter-regex", req.FilterRegex)
	}

	// Output options
	if req.OutputAll {
		args = append(args, "-no-fallback")
	}

	// Silent mode for cleaner output
	if req.Silent {
		args = append(args, "-silent")
	}

	return args
}

// createTargetFile creates a temporary file with target URLs
func (s *URLScanService) createTargetFile(targetURLs []string) (string, error) {
	tempFile, err := os.CreateTemp("", "httpx_targets_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	for _, url := range targetURLs {
		if _, err := tempFile.WriteString(url + "\n"); err != nil {
			os.Remove(tempFile.Name())
			return "", err
		}
	}

	return tempFile.Name(), nil
}

// parseHttpxOutput parses httpx JSON output
func (s *URLScanService) parseHttpxOutput(outputFile string) ([]URLScanResult, error) {
	file, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []URLScanResult
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var httpxResult HttpxOutput
		if err := json.Unmarshal([]byte(line), &httpxResult); err != nil {
			s.logger.Printf("Failed to parse httpx output line: %s, error: %v", line, err)
			continue
		}

		// Parse URL components
		scheme, host, port, path := s.parseURL(httpxResult.URL)

		// Convert CDN boolean to string for storage
		cdnStr := ""
		if httpxResult.CDN {
			if httpxResult.CDNName != "" {
				cdnStr = httpxResult.CDNName
			} else {
				cdnStr = "true"
			}
		}

		result := URLScanResult{
			URL:           httpxResult.URL,
			StatusCode:    httpxResult.StatusCode,
			ContentLength: httpxResult.ContentLength,
			ResponseTime:  httpxResult.ResponseTime,
			Title:         httpxResult.Title,
			Technologies:  httpxResult.Technologies,
			Server:        httpxResult.Server,
			ContentType:   httpxResult.ContentType,
			Location:      httpxResult.Location,
			FinalURL:      httpxResult.FinalURL,
			Scheme:        scheme,
			Host:          host,
			Port:          port,
			Path:          path,
			Hash:          httpxResult.Hash,
			CDN:           cdnStr,
			WebServer:     httpxResult.WebServer,
			IP:            httpxResult.IP,
			CNames:        httpxResult.CNames,
			Chain:         httpxResult.Chain,
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// parseURL extracts components from a URL string
func (s *URLScanService) parseURL(urlStr string) (scheme, host string, port int, path string) {
	// Simple URL parsing
	parts := strings.Split(urlStr, "://")
	if len(parts) != 2 {
		return "", "", 0, ""
	}

	scheme = parts[0]
	remainder := parts[1]

	// Split host/port from path
	hostPathParts := strings.SplitN(remainder, "/", 2)
	hostPort := hostPathParts[0]
	if len(hostPathParts) > 1 {
		path = "/" + hostPathParts[1]
	} else {
		path = "/"
	}

	// Split host from port
	if strings.Contains(hostPort, ":") {
		hostPortParts := strings.Split(hostPort, ":")
		host = hostPortParts[0]
		if len(hostPortParts) > 1 {
			fmt.Sscanf(hostPortParts[1], "%d", &port)
		}
	} else {
		host = hostPort
		// Default ports
		if scheme == "https" {
			port = 443
		} else if scheme == "http" {
			port = 80
		}
	}

	return scheme, host, port, path
}

// determineURLSource identifies which source a URL came from
func (s *URLScanService) determineURLSource(url string, sources map[string][]string) string {
	for source, urls := range sources {
		for _, sourceURL := range urls {
			if sourceURL == url {
				return source
			}
		}
	}
	return "unknown"
}

// calculateScanStats generates statistics about the scan results
func (s *URLScanService) calculateScanStats(results []URLScanResult, sources map[string][]string) map[string]int {
	stats := make(map[string]int)

	// Source statistics
	for source := range sources {
		stats[source+"_targets"] = len(sources[source])
	}

	// Status code statistics
	statusCodes := make(map[int]int)
	schemes := make(map[string]int)
	ports := make(map[int]int)
	techCount := make(map[string]int)

	for _, result := range results {
		statusCodes[result.StatusCode]++
		schemes[result.Scheme]++
		ports[result.Port]++

		for _, tech := range result.Technologies {
			techCount[tech]++
		}
	}

	// Convert to stats
	stats["total_urls"] = len(results)
	stats["unique_hosts"] = len(s.getUniqueHosts(results))
	stats["unique_ports"] = len(ports)
	stats["unique_technologies"] = len(techCount)

	// Status code ranges
	for code, count := range statusCodes {
		if code >= 200 && code < 300 {
			stats["2xx_responses"] += count
		} else if code >= 300 && code < 400 {
			stats["3xx_responses"] += count
		} else if code >= 400 && code < 500 {
			stats["4xx_responses"] += count
		} else if code >= 500 {
			stats["5xx_responses"] += count
		}
	}

	return stats
}

// getUniqueHosts returns unique hosts from results
func (s *URLScanService) getUniqueHosts(results []URLScanResult) []string {
	seen := make(map[string]bool)
	var hosts []string

	for _, result := range results {
		if !seen[result.Host] {
			hosts = append(hosts, result.Host)
			seen[result.Host] = true
		}
	}

	return hosts
}

// SaveResults saves URL scan results to the database
func (s *URLScanService) SaveResults(clientID string, result *URLScanJobResult, scanID string) error {
	collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_urls")
	if err != nil {
		return fmt.Errorf("collection not found: %w", err)
	}

	for _, urlResult := range result.Results {
		record := models.NewRecord(collection)
		record.Set("client", clientID)
		record.Set("url", urlResult.URL)
		record.Set("scheme", urlResult.Scheme)
		record.Set("host", urlResult.Host)
		record.Set("port", urlResult.Port)
		record.Set("path", urlResult.Path)
		record.Set("status_code", urlResult.StatusCode)
		record.Set("content_length", urlResult.ContentLength)
		record.Set("response_time", urlResult.ResponseTime)
		record.Set("title", urlResult.Title)
		record.Set("server", urlResult.Server)
		record.Set("content_type", urlResult.ContentType)
		record.Set("final_url", urlResult.FinalURL)
		record.Set("source", urlResult.Source)
		record.Set("ip", urlResult.IP)
		record.Set("cdn", urlResult.CDN)
		record.Set("webserver", urlResult.WebServer)
		record.Set("scan_id", scanID)
		record.Set("discovered_at", result.StartTime)

		// Store technologies as JSON
		if len(urlResult.Technologies) > 0 {
			techJSON, _ := json.Marshal(urlResult.Technologies)
			record.Set("technologies", string(techJSON))
		}

		// Store hash as JSON
		if len(urlResult.Hash) > 0 {
			hashJSON, _ := json.Marshal(urlResult.Hash)
			record.Set("hash", string(hashJSON))
		}

		// Store cnames as JSON
		if len(urlResult.CNames) > 0 {
			cnamesJSON, _ := json.Marshal(urlResult.CNames)
			record.Set("cnames", string(cnamesJSON))
		}

		// Store chain as JSON
		if len(urlResult.Chain) > 0 {
			chainJSON, _ := json.Marshal(urlResult.Chain)
			record.Set("chain", string(chainJSON))
		}

		if err := s.app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save URL result: %w", err)
		}
	}

	// Save scan summary
	scanCollection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_url_scans")
	if err != nil {
		return fmt.Errorf("scan collection not found: %w", err)
	}

	scanRecord := models.NewRecord(scanCollection)
	scanRecord.Set("client", clientID)
	scanRecord.Set("scan_id", result.ScanID)
	scanRecord.Set("start_time", result.StartTime)
	scanRecord.Set("end_time", result.EndTime)
	scanRecord.Set("duration", result.Duration)
	scanRecord.Set("total_targets", result.TotalTargets)
	scanRecord.Set("live_urls", result.LiveURLs)
	scanRecord.Set("execution_mode", result.ExecutionMode)
	scanRecord.Set("cloud_provider", result.CloudProvider)
	scanRecord.Set("httpx_version", result.HttpxVersion)

	// Store stats as JSON
	statsJSON, _ := json.Marshal(result.Stats)
	scanRecord.Set("stats", string(statsJSON))

	if err := s.app.Dao().SaveRecord(scanRecord); err != nil {
		return fmt.Errorf("failed to save scan summary: %v", err)
	}

	s.logger.Printf("Saved URL scan results: %d URLs, scan ID: %s", len(result.Results), result.ScanID)
	return nil
}

// ensureHttpxInstalled checks if httpx is installed and installs it if needed
func (s *URLScanService) ensureHttpxInstalled() error {
	// First check if httpx is already available in PATH
	if _, err := exec.LookPath("httpx"); err == nil {
		return nil
	}

	// If not found, install it using go install
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "install", "-v", "github.com/projectdiscovery/httpx/cmd/httpx@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install httpx: %w", err)
	}

	// Verify installation
	if _, err := exec.LookPath("httpx"); err != nil {
		return fmt.Errorf("httpx installation failed - not found in PATH after installation")
	}

	return nil
}

// GetSavedURLs retrieves saved URL scan results from the database
func (s *URLScanService) GetSavedURLs(clientID, host string) ([]*models.Record, error) {
	filter := "client = {:client}"
	params := map[string]interface{}{
		"client": clientID,
	}

	if host != "" {
		filter += " && host ~ {:host}"
		params["host"] = host
	}

	records, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_urls",
		filter,
		"created",
		0,
		-1,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve URL results: %w", err)
	}

	return records, nil
}

// GetSavedScans retrieves saved URL scan summaries from the database
func (s *URLScanService) GetSavedScans(clientID string) ([]*models.Record, error) {
	records, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_url_scans",
		"client = {:client}",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve URL scan summaries: %w", err)
	}

	return records, nil
}

// CollectAttackSurfaceTargets collects all relevant targets from the attack surface for nuclei scanning
func (s *URLScanService) CollectAttackSurfaceTargets(req AttackSurfaceTargetRequest) (*AttackSurfaceTargetResult, error) {
	var allTargets []string
	sources := make(map[string][]string)
	seen := make(map[string]bool)

	// Default schemes if not specified
	schemes := req.Schemes
	if len(schemes) == 0 {
		schemes = []string{"http", "https"}
	}

	// Collect manual targets
	if len(req.ManualTargets) > 0 {
		var uniqueManualTargets []string
		for _, target := range req.ManualTargets {
			if !seen[target] {
				uniqueManualTargets = append(uniqueManualTargets, target)
				seen[target] = true
			}
		}
		sources["manual"] = uniqueManualTargets
		allTargets = append(allTargets, uniqueManualTargets...)
	}

	// Collect domain targets
	if req.IncludeDomains {
		domainTargets, err := s.collectDomainTargetsForNuclei(req.ClientID, schemes, req.Ports, req.OnlyWebPorts)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect domain targets: %v", err)
		} else {
			var uniqueDomainTargets []string
			for _, target := range domainTargets {
				if !seen[target] {
					uniqueDomainTargets = append(uniqueDomainTargets, target)
					seen[target] = true
				}
			}
			sources["domains"] = uniqueDomainTargets
			allTargets = append(allTargets, uniqueDomainTargets...)
		}
	}

	// Collect subdomain targets
	if req.IncludeSubdomains {
		subdomainTargets, err := s.collectSubdomainTargetsForNuclei(req.ClientID, schemes, req.Ports, req.OnlyWebPorts)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect subdomain targets: %v", err)
		} else {
			var uniqueSubdomainTargets []string
			for _, target := range subdomainTargets {
				if !seen[target] {
					uniqueSubdomainTargets = append(uniqueSubdomainTargets, target)
					seen[target] = true
				}
			}
			sources["subdomains"] = uniqueSubdomainTargets
			allTargets = append(allTargets, uniqueSubdomainTargets...)
		}
	}

	// Collect port-based targets (IPs with open ports)
	if req.IncludePorts {
		portTargets, err := s.collectPortTargetsForNuclei(req.ClientID, schemes, req.Ports, req.OnlyWebPorts)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect port targets: %v", err)
		} else {
			var uniquePortTargets []string
			for _, target := range portTargets {
				if !seen[target] {
					uniquePortTargets = append(uniquePortTargets, target)
					seen[target] = true
				}
			}
			sources["ports"] = uniquePortTargets
			allTargets = append(allTargets, uniquePortTargets...)
		}
	}

	// Collect netblock-based targets
	if req.IncludeNetblocks {
		netblockTargets, err := s.collectNetblockTargetsForNuclei(req.ClientID, schemes, req.Ports, req.OnlyWebPorts)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect netblock targets: %v", err)
		} else {
			var uniqueNetblockTargets []string
			for _, target := range netblockTargets {
				if !seen[target] {
					uniqueNetblockTargets = append(uniqueNetblockTargets, target)
					seen[target] = true
				}
			}
			sources["netblocks"] = uniqueNetblockTargets
			allTargets = append(allTargets, uniqueNetblockTargets...)
		}
	}

	// Collect live URLs from previous scans
	if req.IncludeURLs {
		urlTargets, err := s.collectURLTargetsForNuclei(req.ClientID)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect URL targets: %v", err)
		} else {
			var uniqueURLTargets []string
			for _, target := range urlTargets {
				if !seen[target] {
					uniqueURLTargets = append(uniqueURLTargets, target)
					seen[target] = true
				}
			}
			sources["urls"] = uniqueURLTargets
			allTargets = append(allTargets, uniqueURLTargets...)
		}
	}

	// Calculate statistics
	stats := map[string]int{
		"total_targets":     len(allTargets),
		"manual_targets":    len(sources["manual"]),
		"domain_targets":    len(sources["domains"]),
		"subdomain_targets": len(sources["subdomains"]),
		"port_targets":      len(sources["ports"]),
		"netblock_targets":  len(sources["netblocks"]),
		"url_targets":       len(sources["urls"]),
	}

	s.logger.Printf("Collected %d unique nuclei targets: %d manual, %d domains, %d subdomains, %d ports, %d netblocks, %d URLs",
		len(allTargets), stats["manual_targets"], stats["domain_targets"], stats["subdomain_targets"],
		stats["port_targets"], stats["netblock_targets"], stats["url_targets"])

	return &AttackSurfaceTargetResult{
		ClientID:     req.ClientID,
		TotalTargets: len(allTargets),
		Targets:      allTargets,
		Sources:      sources,
		Stats:        stats,
	}, nil
}

// collectDomainTargetsForNuclei gets targets from discovered domains for nuclei scanning
func (s *URLScanService) collectDomainTargetsForNuclei(clientID string, schemes []string, ports []int, onlyWebPorts bool) ([]string, error) {
	// Get all domains for this client (parent domains only)
	domains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client} && parent_domain = ''",
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

	var targets []string
	seen := make(map[string]bool)

	// Web ports for nuclei scanning
	webPorts := []int{80, 443, 8080, 8443, 3000, 5000, 8000, 9000}

	for _, domainRecord := range domains {
		domain := domainRecord.GetString("domain")
		if domain == "" {
			continue
		}

		// Generate targets for each scheme and port combination
		for _, scheme := range schemes {
			if len(ports) > 0 {
				for _, port := range ports {
					var target string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						target = fmt.Sprintf("%s://%s", scheme, domain)
					} else {
						target = fmt.Sprintf("%s://%s:%d", scheme, domain, port)
					}

					if !seen[target] {
						targets = append(targets, target)
						seen[target] = true
					}
				}
			} else if onlyWebPorts {
				for _, port := range webPorts {
					var target string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						target = fmt.Sprintf("%s://%s", scheme, domain)
					} else {
						target = fmt.Sprintf("%s://%s:%d", scheme, domain, port)
					}

					if !seen[target] {
						targets = append(targets, target)
						seen[target] = true
					}
				}
			} else {
				// Just the basic domain with scheme
				target := fmt.Sprintf("%s://%s", scheme, domain)
				if !seen[target] {
					targets = append(targets, target)
					seen[target] = true
				}
			}
		}
	}

	return targets, nil
}

// collectSubdomainTargetsForNuclei gets targets from discovered subdomains for nuclei scanning
func (s *URLScanService) collectSubdomainTargetsForNuclei(clientID string, schemes []string, ports []int, onlyWebPorts bool) ([]string, error) {
	// Get all subdomains for this client
	subdomains, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_domains",
		"client = {:client} && parent_domain != ''",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get subdomains: %v", err)
	}

	var targets []string
	seen := make(map[string]bool)

	// Web ports for nuclei scanning
	webPorts := []int{80, 443, 8080, 8443, 3000, 5000, 8000, 9000}

	for _, subdomainRecord := range subdomains {
		subdomain := subdomainRecord.GetString("domain")
		if subdomain == "" {
			continue
		}

		// Generate targets for each scheme and port combination
		for _, scheme := range schemes {
			if len(ports) > 0 {
				for _, port := range ports {
					var target string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						target = fmt.Sprintf("%s://%s", scheme, subdomain)
					} else {
						target = fmt.Sprintf("%s://%s:%d", scheme, subdomain, port)
					}

					if !seen[target] {
						targets = append(targets, target)
						seen[target] = true
					}
				}
			} else if onlyWebPorts {
				for _, port := range webPorts {
					var target string
					if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
						target = fmt.Sprintf("%s://%s", scheme, subdomain)
					} else {
						target = fmt.Sprintf("%s://%s:%d", scheme, subdomain, port)
					}

					if !seen[target] {
						targets = append(targets, target)
						seen[target] = true
					}
				}
			} else {
				// Just the basic subdomain with scheme
				target := fmt.Sprintf("%s://%s", scheme, subdomain)
				if !seen[target] {
					targets = append(targets, target)
					seen[target] = true
				}
			}
		}
	}

	return targets, nil
}

// collectPortTargetsForNuclei gets targets from port scan results for nuclei scanning
func (s *URLScanService) collectPortTargetsForNuclei(clientID string, schemes []string, ports []int, onlyWebPorts bool) ([]string, error) {
	// Get all open ports for this client
	portRecords, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_ports",
		"client = {:client}",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get port records: %v", err)
	}

	var targets []string
	seen := make(map[string]bool)

	// Web ports for nuclei scanning
	webPorts := []int{80, 443, 8080, 8443, 3000, 5000, 8000, 9000}

	for _, portRecord := range portRecords {
		ip := portRecord.GetString("ip")
		port := portRecord.GetInt("port")

		if ip == "" || port == 0 {
			continue
		}

		// Filter by specific ports if specified
		if len(ports) > 0 {
			found := false
			for _, p := range ports {
				if p == port {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by web ports if specified
		if onlyWebPorts {
			found := false
			for _, webPort := range webPorts {
				if webPort == port {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if this is a web port
		isWebPort := false
		for _, webPort := range webPorts {
			if webPort == port {
				isWebPort = true
				break
			}
		}

		if isWebPort {
			// Generate HTTP/HTTPS targets for web ports
			for _, scheme := range schemes {
				var target string
				if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
					target = fmt.Sprintf("%s://%s", scheme, ip)
				} else {
					target = fmt.Sprintf("%s://%s:%d", scheme, ip, port)
				}

				if !seen[target] {
					targets = append(targets, target)
					seen[target] = true
				}
			}
		} else {
			// For non-web ports (SSH, FTP, databases, etc.), use IP:port format
			// This allows nuclei to scan services like SSH (22), FTP (21), MySQL (3306), etc.
			target := fmt.Sprintf("%s:%d", ip, port)
			if !seen[target] {
				targets = append(targets, target)
				seen[target] = true
			}
		}
	}

	return targets, nil
}

// collectNetblockTargetsForNuclei gets targets from netblocks for nuclei scanning
func (s *URLScanService) collectNetblockTargetsForNuclei(clientID string, schemes []string, ports []int, onlyWebPorts bool) ([]string, error) {
	// Get all netblocks for this client
	netblocks, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_netblocks",
		"client = {:client}",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get netblocks: %v", err)
	}

	var targets []string
	seen := make(map[string]bool)

	// Web ports for nuclei scanning
	webPorts := []int{80, 443, 8080, 8443, 3000, 5000, 8000, 9000}

	// Default ports if none specified
	targetPorts := ports
	if len(targetPorts) == 0 {
		if onlyWebPorts {
			targetPorts = webPorts
		} else {
			targetPorts = []int{80, 443} // Just basic web ports
		}
	}

	for _, netblockRecord := range netblocks {
		cidr := netblockRecord.GetString("cidr")
		if cidr == "" {
			continue
		}

		// Parse the CIDR to get individual IPs
		ips, err := s.expandCIDR(cidr)
		if err != nil {
			s.logger.Printf("Warning: Failed to expand CIDR %s: %v", cidr, err)
			continue
		}

		// Limit the number of IPs from each netblock to avoid too many targets
		maxIPs := 100
		if len(ips) > maxIPs {
			ips = ips[:maxIPs]
		}

		// Generate targets for each IP and port combination
		for _, ip := range ips {
			for _, port := range targetPorts {
				// Check if this is a web port
				isWebPort := false
				for _, webPort := range webPorts {
					if webPort == port {
						isWebPort = true
						break
					}
				}

				if isWebPort {
					// Generate HTTP/HTTPS targets for web ports
					for _, scheme := range schemes {
						var target string
						if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
							target = fmt.Sprintf("%s://%s", scheme, ip)
						} else {
							target = fmt.Sprintf("%s://%s:%d", scheme, ip, port)
						}

						if !seen[target] {
							targets = append(targets, target)
							seen[target] = true
						}
					}
				} else {
					// For non-web ports, use IP:port format
					target := fmt.Sprintf("%s:%d", ip, port)
					if !seen[target] {
						targets = append(targets, target)
						seen[target] = true
					}
				}
			}
		}
	}

	return targets, nil
}

// collectURLTargetsForNuclei gets targets from discovered URLs for nuclei scanning
func (s *URLScanService) collectURLTargetsForNuclei(clientID string) ([]string, error) {
	// Get all live URLs for this client
	urlRecords, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_urls",
		"client = {:client} && status_code >= 200 && status_code < 400",
		"created",
		0,
		-1,
		map[string]interface{}{
			"client": clientID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL records: %v", err)
	}

	var targets []string
	seen := make(map[string]bool)

	for _, urlRecord := range urlRecords {
		url := urlRecord.GetString("url")
		if url == "" {
			continue
		}

		if !seen[url] {
			targets = append(targets, url)
			seen[url] = true
		}
	}

	return targets, nil
}

// expandCIDR expands a CIDR notation to individual IP addresses
func (s *URLScanService) expandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses for small subnets
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// CreateNucleiTargetFromAttackSurface creates a nuclei target record with attack surface data
func (s *URLScanService) CreateNucleiTargetFromAttackSurface(name string, req AttackSurfaceTargetRequest) (string, error) {
	// Collect targets from attack surface
	result, err := s.CollectAttackSurfaceTargets(req)
	if err != nil {
		return "", fmt.Errorf("failed to collect attack surface targets: %v", err)
	}

	if len(result.Targets) == 0 {
		return "", fmt.Errorf("no targets found from attack surface")
	}

	// Create nuclei target record
	collection, err := s.app.Dao().FindCollectionByNameOrId("nuclei_targets")
	if err != nil {
		return "", fmt.Errorf("nuclei_targets collection not found: %v", err)
	}

	record := models.NewRecord(collection)
	record.Set("name", name)
	record.Set("client", req.ClientID)
	record.Set("count", len(result.Targets))

	// Store targets as JSON
	targetsJSON, err := json.Marshal(result.Targets)
	if err != nil {
		return "", fmt.Errorf("failed to marshal targets: %v", err)
	}
	record.Set("targets", string(targetsJSON))

	// Store metadata about sources
	metadataJSON, err := json.Marshal(map[string]interface{}{
		"sources": result.Sources,
		"stats":   result.Stats,
		"config":  req,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %v", err)
	}
	record.Set("metadata", string(metadataJSON))

	if err := s.app.Dao().SaveRecord(record); err != nil {
		return "", fmt.Errorf("failed to save nuclei target: %v", err)
	}

	s.logger.Printf("Created nuclei target '%s' with %d targets from attack surface", name, len(result.Targets))
	return record.Id, nil
}
