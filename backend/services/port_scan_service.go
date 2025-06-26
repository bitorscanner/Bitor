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

// PortScanService handles port scanning operations using naabu
type PortScanService struct {
	app    *pocketbase.PocketBase
	logger *log.Logger
	// Track running scans
	activeScans map[string]*ScanProgress
	mutex       sync.RWMutex
}

// ScanProgress tracks the progress of a running scan
type ScanProgress struct {
	ScanID       string     `json:"scan_id"`
	ClientID     string     `json:"client_id"`
	Status       string     `json:"status"`   // "running", "completed", "failed"
	Progress     int        `json:"progress"` // 0-100
	Message      string     `json:"message"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	TotalTargets int        `json:"total_targets"`
	TotalPorts   int        `json:"total_ports"`
	OpenPorts    int        `json:"open_ports"`
	Error        string     `json:"error,omitempty"`
}

// PortScanResult represents a discovered open port
type PortScanResult struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"`
	Service  string `json:"service,omitempty"`
	State    string `json:"state"`
	Host     string `json:"host,omitempty"`
	Source   string `json:"source"` // "domains", "netblocks", "manual"
}

// PortScanRequest represents a port scan request
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

// PortScanJobResult represents the complete result of a port scan
type PortScanJobResult struct {
	ClientID      string           `json:"client_id"`
	ScanID        string           `json:"scan_id"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	Duration      string           `json:"duration"`
	TotalTargets  int              `json:"total_targets"`
	TotalPorts    int              `json:"total_ports"`
	OpenPorts     int              `json:"open_ports"`
	TargetIPs     []string         `json:"target_ips"`
	Results       []PortScanResult `json:"results"`
	Stats         map[string]int   `json:"stats"`
	ExecutionMode string           `json:"execution_mode"`
	CloudProvider string           `json:"cloud_provider,omitempty"`
	NaabuVersion  string           `json:"naabu_version,omitempty"`
	Error         string           `json:"error,omitempty"`
}

// NaabuOutput represents naabu JSON output format
type NaabuOutput struct {
	IP    string `json:"ip"`
	Port  int    `json:"port"`
	Proto string `json:"proto,omitempty"`
}

// NewPortScanService creates a new instance of PortScanService
func NewPortScanService(app *pocketbase.PocketBase) *PortScanService {
	return &PortScanService{
		app:         app,
		logger:      log.New(log.Writer(), "[PortScanService] ", log.LstdFlags),
		activeScans: make(map[string]*ScanProgress),
	}
}

// StartAsyncPortScan starts a port scan in the background and returns the scan ID
func (s *PortScanService) StartAsyncPortScan(req PortScanRequest) (string, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("portscan_%s_%d", req.ClientID, startTime.Unix())

	// Create progress tracker
	progress := &ScanProgress{
		ScanID:    scanID,
		ClientID:  req.ClientID,
		Status:    "running",
		Progress:  0,
		Message:   "Initializing scan...",
		StartTime: startTime,
	}

	// Store progress tracker
	s.mutex.Lock()
	s.activeScans[scanID] = progress
	s.mutex.Unlock()

	// Start scan in background
	go func() {
		defer func() {
			// Clean up progress tracker after some time
			time.AfterFunc(1*time.Hour, func() {
				s.mutex.Lock()
				delete(s.activeScans, scanID)
				s.mutex.Unlock()
			})
		}()

		// Update progress
		s.updateScanProgress(scanID, 5, "Collecting target IPs...")

		// Run the actual scan
		result, err := s.RunPortScan(context.Background(), req)

		endTime := time.Now()

		if err != nil {
			s.updateScanProgress(scanID, 0, fmt.Sprintf("Scan failed: %v", err))
			s.mutex.Lock()
			if progress, ok := s.activeScans[scanID]; ok {
				progress.Status = "failed"
				progress.Error = err.Error()
				progress.EndTime = &endTime
			}
			s.mutex.Unlock()
			s.logger.Printf("Port scan %s failed: %v", scanID, err)
			return
		}

		// Save results
		s.updateScanProgress(scanID, 95, "Saving results...")
		if err := s.SaveResults(result); err != nil {
			s.logger.Printf("Failed to save port scan results for %s: %v", scanID, err)
		}

		// Mark as completed
		s.mutex.Lock()
		if progress, ok := s.activeScans[scanID]; ok {
			progress.Status = "completed"
			progress.Progress = 100
			progress.Message = fmt.Sprintf("Scan completed: %d open ports found", result.OpenPorts)
			progress.EndTime = &endTime
			progress.OpenPorts = result.OpenPorts
		}
		s.mutex.Unlock()

		s.logger.Printf("Port scan %s completed successfully", scanID)
	}()

	return scanID, nil
}

// GetScanProgress returns the current progress of a scan
func (s *PortScanService) GetScanProgress(scanID string) (*ScanProgress, error) {
	s.mutex.RLock()
	progress, exists := s.activeScans[scanID]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("scan not found")
	}

	// Return a copy to avoid race conditions
	progressCopy := *progress
	return &progressCopy, nil
}

// updateScanProgress updates the progress of a running scan
func (s *PortScanService) updateScanProgress(scanID string, progress int, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if scan, ok := s.activeScans[scanID]; ok {
		scan.Progress = progress
		scan.Message = message
	}
}

// CollectTargetIPs gathers all IP addresses from domains and netblocks
func (s *PortScanService) CollectTargetIPs(clientID string, includeDomains, includeNetblocks bool, manualIPs []string) ([]string, map[string][]string, error) {
	var allIPs []string
	sources := make(map[string][]string)
	seen := make(map[string]bool)

	// Add manual IPs
	if len(manualIPs) > 0 {
		var validIPs []string
		for _, ip := range manualIPs {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}

			// Validate IP or resolve hostname
			if net.ParseIP(ip) != nil {
				if !seen[ip] {
					validIPs = append(validIPs, ip)
					seen[ip] = true
				}
			} else {
				// Try to resolve as hostname
				ips, err := net.LookupIP(ip)
				if err == nil {
					for _, resolvedIP := range ips {
						if resolvedIP.To4() != nil { // IPv4 only for now
							ipStr := resolvedIP.String()
							if !seen[ipStr] {
								validIPs = append(validIPs, ipStr)
								seen[ipStr] = true
							}
						}
					}
				}
			}
		}
		sources["manual"] = validIPs
		allIPs = append(allIPs, validIPs...)
	}

	// Collect IPs from discovered domains
	if includeDomains {
		domainIPs, err := s.collectDomainIPs(clientID)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect domain IPs: %v", err)
		} else {
			var uniqueDomainIPs []string
			for _, ip := range domainIPs {
				if !seen[ip] {
					uniqueDomainIPs = append(uniqueDomainIPs, ip)
					seen[ip] = true
				}
			}
			sources["domains"] = uniqueDomainIPs
			allIPs = append(allIPs, uniqueDomainIPs...)
		}
	}

	// Collect IPs from netblocks
	if includeNetblocks {
		netblockIPs, err := s.collectNetblockIPs(clientID)
		if err != nil {
			s.logger.Printf("Warning: Failed to collect netblock IPs: %v", err)
		} else {
			var uniqueNetblockIPs []string
			for _, ip := range netblockIPs {
				if !seen[ip] {
					uniqueNetblockIPs = append(uniqueNetblockIPs, ip)
					seen[ip] = true
				}
			}
			sources["netblocks"] = uniqueNetblockIPs
			allIPs = append(allIPs, uniqueNetblockIPs...)
		}
	}

	// Sort IPs for consistent output
	sort.Strings(allIPs)

	s.logger.Printf("Collected %d unique target IPs: %d manual, %d from domains, %d from netblocks",
		len(allIPs), len(sources["manual"]), len(sources["domains"]), len(sources["netblocks"]))

	return allIPs, sources, nil
}

// collectDomainIPs gets all IP addresses from discovered domains and subdomains
func (s *PortScanService) collectDomainIPs(clientID string) ([]string, error) {
	// Get all domains for this client
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

	var ips []string
	seen := make(map[string]bool)

	for _, domainRecord := range domains {
		domain := domainRecord.GetString("domain")
		if domain == "" {
			continue
		}

		// Resolve domain to IPs
		resolvedIPs, err := net.LookupIP(domain)
		if err != nil {
			s.logger.Printf("Failed to resolve %s: %v", domain, err)
			continue
		}

		for _, ip := range resolvedIPs {
			if ip.To4() != nil { // IPv4 only
				ipStr := ip.String()
				if !seen[ipStr] {
					ips = append(ips, ipStr)
					seen[ipStr] = true
				}
			}
		}
	}

	return ips, nil
}

// collectNetblockIPs expands CIDR ranges from discovered netblocks
func (s *PortScanService) collectNetblockIPs(clientID string) ([]string, error) {
	// Get high-confidence netblocks for this client
	netblocks, err := s.app.Dao().FindRecordsByFilter(
		"attack_surface_netblocks",
		"client = {:client} && confidence >= {:confidence}",
		"confidence DESC",
		0,
		100, // Limit to top 100 netblocks to avoid massive scans
		map[string]interface{}{
			"client":     clientID,
			"confidence": 0.7, // Only include medium+ confidence netblocks
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get netblocks: %v", err)
	}

	var ips []string
	seen := make(map[string]bool)

	for _, netblockRecord := range netblocks {
		cidr := netblockRecord.GetString("cidr")
		if cidr == "" {
			continue
		}

		// Parse CIDR
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			s.logger.Printf("Invalid CIDR %s: %v", cidr, err)
			continue
		}

		// Expand CIDR to individual IPs (with safety limits)
		expandedIPs := s.expandCIDR(ipNet, 1000) // Limit to 1000 IPs per CIDR
		for _, ip := range expandedIPs {
			if !seen[ip] {
				ips = append(ips, ip)
				seen[ip] = true
			}
		}
	}

	return ips, nil
}

// expandCIDR converts a CIDR to individual IP addresses with a limit
func (s *PortScanService) expandCIDR(ipNet *net.IPNet, limit int) []string {
	var ips []string

	// Calculate network size
	ones, bits := ipNet.Mask.Size()
	if bits-ones > 20 { // More than /12 would be too many IPs
		s.logger.Printf("CIDR %s too large, skipping expansion", ipNet.String())
		return ips
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		return ips // Skip IPv6 for now
	}

	// Convert IP to uint32 for iteration
	ipUint := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	// Calculate network and broadcast addresses
	maskUint := uint32(0xFFFFFFFF) << uint32(32-ones)
	networkUint := ipUint & maskUint
	broadcastUint := networkUint | (^maskUint)

	count := 0
	for i := networkUint; i <= broadcastUint && count < limit; i++ {
		// Skip network and broadcast addresses for subnets smaller than /31
		if ones < 31 && (i == networkUint || i == broadcastUint) {
			continue
		}

		// Convert back to IP
		resultIP := net.IPv4(
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)

		ips = append(ips, resultIP.String())
		count++
	}

	return ips
}

// RunPortScan executes naabu port scan locally or in the cloud
func (s *PortScanService) RunPortScan(ctx context.Context, req PortScanRequest) (*PortScanJobResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("portscan_%s_%d", req.ClientID, startTime.Unix())

	result := &PortScanJobResult{
		ClientID:      req.ClientID,
		ScanID:        scanID,
		StartTime:     startTime,
		ExecutionMode: req.ExecutionMode,
		CloudProvider: req.CloudProvider,
		Stats:         make(map[string]int),
	}

	// Ensure naabu is installed (for local scans)
	if req.ExecutionMode != "cloud" {
		if err := s.ensureNaabuInstalled(); err != nil {
			result.Error = fmt.Sprintf("Failed to ensure naabu is installed: %v", err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime).String()
			return result, err
		}
	}

	// Collect target IPs
	targetIPs, sources, err := s.CollectTargetIPs(req.ClientID, req.IncludeDomains, req.IncludeNetblocks, req.TargetIPs)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to collect target IPs: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(targetIPs) == 0 {
		result.Error = "No target IPs found"
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, fmt.Errorf("no target IPs found")
	}

	result.TargetIPs = targetIPs
	result.TotalTargets = len(targetIPs)

	// Calculate estimated ports to scan
	portCount := s.estimatePortCount(req.Ports, req.TopPorts)
	result.TotalPorts = portCount * len(targetIPs)

	s.logger.Printf("Starting port scan: %d targets, ~%d total port checks", len(targetIPs), result.TotalPorts)

	// Execute scan based on mode
	var scanResults []PortScanResult
	if req.ExecutionMode == "cloud" {
		scanResults, err = s.runCloudScan(ctx, req, targetIPs)
	} else {
		scanResults, err = s.runLocalScan(ctx, req, targetIPs)
	}

	if err != nil {
		result.Error = fmt.Sprintf("Port scan failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	// Enhance results with source information
	for i := range scanResults {
		scanResults[i].Source = s.determineIPSource(scanResults[i].IP, sources)
	}

	result.Results = scanResults
	result.OpenPorts = len(scanResults)
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime).String()

	// Calculate statistics
	result.Stats = s.calculateScanStats(scanResults, sources)

	s.logger.Printf("Port scan completed: %d open ports found in %s", result.OpenPorts, result.Duration)

	return result, nil
}

// runLocalScan executes naabu locally
func (s *PortScanService) runLocalScan(ctx context.Context, req PortScanRequest, targetIPs []string) ([]PortScanResult, error) {
	// naabu installation is handled by the caller

	// Create temporary target file
	targetFile, err := s.createTargetFile(targetIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to create target file: %v", err)
	}
	defer os.Remove(targetFile)

	// Create temporary output file
	outputFile, err := os.CreateTemp("", "naabu_output_*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Build naabu command
	args := s.buildNaabuArgs(req, targetFile, outputFile.Name())

	s.logger.Printf("Running naabu command: naabu %s", strings.Join(args, " "))

	// Execute naabu
	cmd := exec.CommandContext(ctx, "naabu", args...)

	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start naabu: %v", err)
	}

	// Log stderr output
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.logger.Printf("naabu: %s", scanner.Text())
		}
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("naabu execution failed: %v", err)
	}

	// Parse results
	results, err := s.parseNaabuOutput(outputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse naabu output: %v", err)
	}

	return results, nil
}

// runCloudScan executes naabu on a cloud instance
func (s *PortScanService) runCloudScan(ctx context.Context, req PortScanRequest, targetIPs []string) ([]PortScanResult, error) {
	// TODO: Implement cloud execution
	// This would involve:
	// 1. Spinning up a cloud instance (AWS/GCP/Azure/DigitalOcean)
	// 2. Installing naabu on the instance
	// 3. Uploading target list
	// 4. Running the scan remotely
	// 5. Downloading results
	// 6. Cleaning up the instance

	return nil, fmt.Errorf("cloud scanning not yet implemented")
}

// buildNaabuArgs constructs command line arguments for naabu
func (s *PortScanService) buildNaabuArgs(req PortScanRequest, targetFile, outputFile string) []string {
	var args []string

	// Target file
	args = append(args, "-list", targetFile)

	// Output format
	args = append(args, "-json", "-output", outputFile)

	// Port configuration
	if req.TopPorts != "" {
		args = append(args, "-top-ports", req.TopPorts)
	} else if req.Ports != "" {
		args = append(args, "-port", req.Ports)
	} else {
		args = append(args, "-top-ports", "100") // Default
	}

	if req.ExcludePorts != "" {
		args = append(args, "-exclude-ports", req.ExcludePorts)
	}

	// Scan type
	if req.ScanType == "SYN" {
		args = append(args, "-scan-type", "s")
	} else {
		args = append(args, "-scan-type", "c") // CONNECT scan (default)
	}

	// Performance options
	if req.Rate > 0 {
		args = append(args, "-rate", fmt.Sprintf("%d", req.Rate))
	}
	if req.Threads > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", req.Threads))
	}
	if req.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", req.Timeout))
	}
	if req.Retries > 0 {
		args = append(args, "-retries", fmt.Sprintf("%d", req.Retries))
	}

	// Host discovery
	if req.HostDiscovery {
		args = append(args, "-wn")
	} else {
		args = append(args, "-Pn") // Skip host discovery
	}

	// Additional options
	if req.ExcludeCDN {
		args = append(args, "-exclude-cdn")
	}
	if req.Verify {
		args = append(args, "-verify")
	}

	// Nmap integration
	if req.NmapIntegration && req.NmapCommand != "" {
		args = append(args, "-nmap-cli", req.NmapCommand)
	}

	// Silent mode for cleaner output
	args = append(args, "-silent")

	return args
}

// createTargetFile creates a temporary file with target IPs
func (s *PortScanService) createTargetFile(targetIPs []string) (string, error) {
	tempFile, err := os.CreateTemp("", "naabu_targets_*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	for _, ip := range targetIPs {
		if _, err := tempFile.WriteString(ip + "\n"); err != nil {
			os.Remove(tempFile.Name())
			return "", err
		}
	}

	return tempFile.Name(), nil
}

// parseNaabuOutput parses naabu JSON output
func (s *PortScanService) parseNaabuOutput(outputFile string) ([]PortScanResult, error) {
	file, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []PortScanResult
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var naabuResult NaabuOutput
		if err := json.Unmarshal([]byte(line), &naabuResult); err != nil {
			s.logger.Printf("Failed to parse naabu output line: %s, error: %v", line, err)
			continue
		}

		result := PortScanResult{
			IP:       naabuResult.IP,
			Port:     naabuResult.Port,
			Protocol: naabuResult.Proto,
			State:    "open",
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// determineIPSource identifies which source an IP came from
func (s *PortScanService) determineIPSource(ip string, sources map[string][]string) string {
	for source, ips := range sources {
		for _, sourceIP := range ips {
			if sourceIP == ip {
				return source
			}
		}
	}
	return "unknown"
}

// calculateScanStats generates statistics about the scan results
func (s *PortScanService) calculateScanStats(results []PortScanResult, sources map[string][]string) map[string]int {
	stats := make(map[string]int)

	// Basic counts
	stats["total_open_ports"] = len(results)

	// Count unique IPs with open ports
	uniqueIPs := make(map[string]bool)
	for _, result := range results {
		uniqueIPs[result.IP] = true
	}
	stats["hosts_with_open_ports"] = len(uniqueIPs)

	// Count by source
	for source := range sources {
		stats[fmt.Sprintf("targets_from_%s", source)] = len(sources[source])
	}

	// Port statistics
	portCounts := make(map[int]int)
	for _, result := range results {
		portCounts[result.Port]++
	}

	// Most common ports
	var topPorts []struct {
		Port  int
		Count int
	}
	for port, count := range portCounts {
		topPorts = append(topPorts, struct {
			Port  int
			Count int
		}{port, count})
	}

	// Sort by count and take top 5
	if len(topPorts) > 0 {
		sort.Slice(topPorts, func(i, j int) bool {
			return topPorts[i].Count > topPorts[j].Count
		})

		for i, portStat := range topPorts {
			if i >= 5 {
				break
			}
			stats[fmt.Sprintf("top_port_%d", i+1)] = portStat.Port
			stats[fmt.Sprintf("top_port_%d_count", i+1)] = portStat.Count
		}
	}

	return stats
}

// estimatePortCount estimates how many ports will be scanned
func (s *PortScanService) estimatePortCount(ports, topPorts string) int {
	if topPorts != "" {
		switch topPorts {
		case "100":
			return 100
		case "1000":
			return 1000
		case "full":
			return 65535
		default:
			return 100
		}
	}

	if ports != "" {
		// Simple estimation - count commas and ranges
		count := strings.Count(ports, ",") + 1
		ranges := strings.Count(ports, "-")
		return count + (ranges * 50) // Rough estimate for ranges
	}

	return 100 // Default
}

// SaveResults saves port scan results to the database
func (s *PortScanService) SaveResults(result *PortScanJobResult) error {
	// Save individual port scan results
	for _, portResult := range result.Results {
		collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_ports")
		if err != nil {
			s.logger.Printf("Failed to find attack_surface_ports collection: %v", err)
			continue
		}

		record := models.NewRecord(collection)
		record.Set("client", result.ClientID)
		record.Set("scan_id", result.ScanID)
		record.Set("ip", portResult.IP)
		record.Set("port", portResult.Port)
		record.Set("protocol", portResult.Protocol)
		record.Set("service", portResult.Service)
		record.Set("state", portResult.State)
		record.Set("host", portResult.Host)
		record.Set("source", portResult.Source)
		record.Set("discovered_at", result.StartTime.Format(time.RFC3339))

		if err := s.app.Dao().SaveRecord(record); err != nil {
			s.logger.Printf("Failed to save port result %s:%d: %v", portResult.IP, portResult.Port, err)
		}
	}

	// Save scan job summary
	collection, err := s.app.Dao().FindCollectionByNameOrId("attack_surface_port_scans")
	if err != nil {
		s.logger.Printf("Failed to find attack_surface_port_scans collection: %v", err)
		return err
	}

	record := models.NewRecord(collection)
	record.Set("client", result.ClientID)
	record.Set("scan_id", result.ScanID)
	record.Set("start_time", result.StartTime.Format(time.RFC3339))
	record.Set("end_time", result.EndTime.Format(time.RFC3339))
	record.Set("duration", result.Duration)
	record.Set("total_targets", result.TotalTargets)
	record.Set("total_ports", result.TotalPorts)
	record.Set("open_ports", result.OpenPorts)
	record.Set("execution_mode", result.ExecutionMode)
	record.Set("cloud_provider", result.CloudProvider)
	record.Set("naabu_version", result.NaabuVersion)
	record.Set("error", result.Error)

	// Store stats as JSON
	statsJSON, _ := json.Marshal(result.Stats)
	record.Set("stats", string(statsJSON))

	// Store target IPs as JSON
	targetsJSON, _ := json.Marshal(result.TargetIPs)
	record.Set("target_ips", string(targetsJSON))

	if err := s.app.Dao().SaveRecord(record); err != nil {
		return fmt.Errorf("failed to save scan summary: %v", err)
	}

	s.logger.Printf("Saved port scan results: %d ports, scan ID: %s", len(result.Results), result.ScanID)
	return nil
}

// ensureNaabuInstalled checks if naabu is installed and installs it if needed
func (s *PortScanService) ensureNaabuInstalled() error {
	// First check if naabu is already available in PATH
	if _, err := exec.LookPath("naabu"); err == nil {
		return nil
	}

	// If not found, install it using go install
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "install", "-v", "github.com/projectdiscovery/naabu/v2/cmd/naabu@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install naabu: %w", err)
	}

	// Verify installation
	if _, err := exec.LookPath("naabu"); err != nil {
		return fmt.Errorf("naabu installation failed - not found in PATH after installation")
	}

	return nil
}
