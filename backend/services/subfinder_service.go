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
	file, err := os.Open(outputFile)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var subdomains []string
	sourcesMap := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var subfinderResult SubfinderOutput
		if err := json.Unmarshal([]byte(line), &subfinderResult); err != nil {
			// Try parsing as plain text (fallback)
			if strings.TrimSpace(line) != "" {
				subdomains = append(subdomains, strings.TrimSpace(line))
			}
			continue
		}

		if subfinderResult.Host != "" {
			subdomains = append(subdomains, subfinderResult.Host)
			if subfinderResult.Source != "" {
				sourcesMap[subfinderResult.Source] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	// Convert sources map to slice
	var sources []string
	for source := range sourcesMap {
		sources = append(sources, source)
	}

	return subdomains, sources, nil
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

// GetAvailableSources returns the list of available subfinder sources
func (s *SubfinderService) GetAvailableSources() []string {
	// Default subfinder sources
	return []string{
		"alienvault",
		"anubis",
		"bevigil",
		"binaryedge",
		"bufferover",
		"c99",
		"censys",
		"certspotter",
		"chaos",
		"chinaz",
		"crtsh",
		"dnsdb",
		"dnsdumpster",
		"dnsrepo",
		"fofa",
		"fullhunt",
		"github",
		"hackertarget",
		"hunter",
		"intelx",
		"passivetotal",
		"quake",
		"rapiddns",
		"reconcloud",
		"riddler",
		"robtex",
		"securitytrails",
		"shodan",
		"spyse",
		"sublist3r",
		"threatbook",
		"threatcrowd",
		"threatminer",
		"virustotal",
		"waybackarchive",
		"whoisxmlapi",
		"zoomeye",
	}
}
