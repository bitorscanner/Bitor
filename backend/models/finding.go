package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Finding represents the unified finding structure used throughout the application
type Finding struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Severity         string                 `json:"severity"`
	SeverityOrder    int                    `json:"severity_order"`
	Host             string                 `json:"host"`
	Type             string                 `json:"type"`
	Tool             string                 `json:"tool"`
	ScanID           string                 `json:"scan_id"`
	ClientID         string                 `json:"client"`
	TemplateID       string                 `json:"template_id"`
	IP               string                 `json:"ip"`
	Port             string                 `json:"port"`
	Protocol         string                 `json:"protocol"`
	Timestamp        string                 `json:"timestamp"`
	Info             map[string]interface{} `json:"info"`
	MatchedAt        string                 `json:"matched_at"`
	MatcherName      string                 `json:"matcher_name"`
	CurlCommand      string                 `json:"curl_command"`
	Request          string                 `json:"request"`
	Response         string                 `json:"response"`
	ExtractedResults []string               `json:"extracted_results"`
	URL              string                 `json:"url"`
	CreatedBy        string                 `json:"created_by"`
}

// NewFindingFromNuclei converts a NucleiFinding to our unified Finding structure
func NewFindingFromNuclei(nucleiFinding NucleiFinding, clientID, scanID string, userID string) (*Finding, error) {
	// Ensure template_id is not empty and properly set
	templateID := nucleiFinding.TemplateID
	if templateID == "" {
		if nucleiFinding.TemplatePath != "" {
			base := filepath.Base(nucleiFinding.TemplatePath)
			templateID = strings.TrimSuffix(base, filepath.Ext(base))
		} else {
			templateID = nucleiFinding.Info.Name
		}
	}

	// Get severity order
	severityOrder := getSeverityOrder(nucleiFinding.Info.Severity)

	// Create the finding
	finding := &Finding{
		Name:             nucleiFinding.Info.Name,
		Description:      nucleiFinding.Info.Description,
		Severity:         nucleiFinding.Info.Severity,
		SeverityOrder:    severityOrder,
		Host:             nucleiFinding.Host,
		Type:             nucleiFinding.Type,
		Tool:             "nuclei",
		ScanID:           scanID,
		ClientID:         clientID,
		TemplateID:       templateID,
		IP:               nucleiFinding.IP,
		Port:             nucleiFinding.Port,
		Protocol:         nucleiFinding.Scheme,
		Timestamp:        nucleiFinding.Timestamp,
		MatchedAt:        nucleiFinding.MatchedAt,
		MatcherName:      nucleiFinding.MatcherName,
		CurlCommand:      nucleiFinding.CurlCommand,
		Request:          nucleiFinding.Request,
		Response:         nucleiFinding.Response,
		ExtractedResults: nucleiFinding.ExtractedResults,
		URL:              nucleiFinding.URL,
		CreatedBy:        userID,
	}

	// Convert Info to map
	infoJSON, err := json.Marshal(nucleiFinding.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal info: %v", err)
	}
	if err := json.Unmarshal(infoJSON, &finding.Info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal info: %v", err)
	}

	return finding, nil
}

// GenerateHash creates a unique hash for a finding
func (f *Finding) GenerateHash() string {
	// Create a normalized string for hashing that captures all unique elements
	normalizedString := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		strings.TrimSpace(strings.ToLower(f.Name)),        // Template name
		strings.TrimSpace(strings.ToLower(f.Host)),        // Target host
		strings.TrimSpace(strings.ToLower(f.Type)),        // Finding type
		strings.TrimSpace(strings.ToLower(f.Description)), // Finding description
		strings.TrimSpace(strings.ToLower(f.Tool)),        // Tool
		strings.TrimSpace(strings.ToLower(f.ClientID)),    // Client ID
		strings.TrimSpace(strings.ToLower(f.TemplateID)),  // Template ID
		strings.TrimSpace(strings.ToLower(f.MatchedAt)),   // Matched location (important for HTTP headers)
		strings.TrimSpace(strings.ToLower(f.URL)),         // Specific URL/endpoint
		strings.TrimSpace(strings.ToLower(f.Port)),        // Port number
		strings.TrimSpace(strings.ToLower(f.Response)),    // Response headers (important for HTTP header findings)
		strings.TrimSpace(strings.ToLower(f.MatcherName)), // Matcher name (important for distinguishing different missing headers)
		strings.Join(f.ExtractedResults, "|"),             // Include extracted results for TLS versions and WHOIS data
	)

	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(normalizedString))
	return hex.EncodeToString(hash[:])
}

// getSeverityOrder returns the numeric order for a severity string
func getSeverityOrder(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	case "info":
		return 5
	default:
		return 0 // unknown severity
	}
}

// ToMap converts a Finding to a map for database storage
func (f *Finding) ToMap() map[string]interface{} {
	data := map[string]interface{}{
		"hash":           f.GenerateHash(),
		"name":           f.Name,
		"description":    f.Description,
		"severity":       f.Severity,
		"severity_order": f.SeverityOrder,
		"type":           f.Type,
		"tool":           f.Tool,
		"host":           f.Host,
		"status":         "open",
		"client":         f.ClientID,
		"scan_id":        f.ScanID,
		"template_id":    f.TemplateID,
		"ip":             f.IP,
		"port":           f.Port,
		"protocol":       f.Protocol,
		"timestamp":      f.Timestamp,
		"matched_at":     f.MatchedAt,
		"matcher_name":   f.MatcherName,
		"curl_command":   f.CurlCommand,
		"request":        f.Request,
		"response":       f.Response,
		"url":            f.URL,
		"first_seen":     time.Now().Format(time.RFC3339),
		"created_by":     f.CreatedBy,
	}

	// Handle Info field
	if f.Info != nil {
		if infoJSON, err := json.Marshal(f.Info); err == nil {
			data["info"] = string(infoJSON)
		} else {
			data["info"] = "{}"
		}
	} else {
		data["info"] = "{}"
	}

	// Handle ExtractedResults
	if len(f.ExtractedResults) > 0 {
		if extractedBytes, err := json.Marshal(f.ExtractedResults); err == nil {
			data["extracted_results"] = string(extractedBytes)
		} else {
			data["extracted_results"] = "[]"
		}
	} else {
		data["extracted_results"] = "[]"
	}

	return data
}
