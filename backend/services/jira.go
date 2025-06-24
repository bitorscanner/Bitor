package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"bitor/types"
	"strings"
	"time"
)

// JiraService handles all Jira-related operations
type JiraService struct {
	client *http.Client
}

// NewJiraService creates a new JiraService instance
func NewJiraService() *JiraService {
	return &JiraService{
		client: &http.Client{},
	}
}

// CreateJiraTicket creates a new ticket in Jira
func (s *JiraService) CreateJiraTicket(settings types.JiraSettings, clientID string, title string, description string) error {
	// Get the organization ID for the client
	var organizationID string
	for _, mapping := range settings.ClientMappings {
		if mapping.ClientID == clientID {
			organizationID = mapping.OrganizationID
			break
		}
	}

	// Create the Jira ticket payload
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": settings.ProjectKey,
			},
			"summary":     title,
			"description": description,
			"issuetype": map[string]string{
				"name": settings.IssueType,
			},
		},
	}

	// Add organization if one is mapped
	if organizationID != "" {
		payload["fields"].(map[string]interface{})["customfield_10002"] = []string{organizationID}
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling Jira payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/rest/api/2/issue", settings.JiraURL),
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.SetBasicAuth(settings.Username, settings.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetOrganizationForClient returns the organization ID for a given client
func (s *JiraService) GetOrganizationForClient(settings types.JiraSettings, clientID string) string {
	for _, mapping := range settings.ClientMappings {
		if mapping.ClientID == clientID {
			return mapping.OrganizationID
		}
	}
	return "" // Return empty string if no mapping found
}

type JiraProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type JiraIssueType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *JiraService) GetProjects(settings types.JiraSettings) ([]JiraProject, error) {
	// Build the API URL for fetching projects
	apiURL := fmt.Sprintf("%s/rest/api/3/project", settings.JiraURL)

	// Create the request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", settings.Username, settings.APIKey)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch projects: %s", string(body))
	}

	// Parse the response
	var projects []JiraProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return projects, nil
}

func (s *JiraService) GetIssueTypes(settings types.JiraSettings, projectKey string) ([]JiraIssueType, error) {
	// Build the API URL for fetching issue types
	apiURL := fmt.Sprintf("%s/rest/api/3/project/%s", settings.JiraURL, projectKey)

	// Create the request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", settings.Username, settings.APIKey)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch project data: %s", string(body))
	}

	// Parse the response
	var projectData struct {
		IssueTypes []JiraIssueType `json:"issueTypes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projectData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return projectData.IssueTypes, nil
}

type JiraOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *JiraService) GetOrganizations(settings types.JiraSettings, projectKey string) ([]JiraOrganization, error) {
	// Use the Service Desk API to get organizations
	apiURL := fmt.Sprintf("%s/rest/servicedeskapi/organization", settings.JiraURL)

	// Create the request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication headers
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", settings.Username, settings.APIKey)))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations: %s", string(body))
	}

	// Parse the response
	var response struct {
		Values []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"values"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Convert to JiraOrganization slice
	var organizations []JiraOrganization
	for _, org := range response.Values {
		organizations = append(organizations, JiraOrganization{
			ID:   org.ID,
			Name: org.Name,
		})
	}

	return organizations, nil
}

// GetCustomerFromIssue extracts customer information from a Jira issue
func (s *JiraService) GetCustomerFromIssue(issue types.JiraIssue) *types.JiraCustomer {
	if len(issue.Fields.CustomField10002) == 0 {
		return nil
	}

	return &types.JiraCustomer{
		ID:   issue.Fields.CustomField10002[0].ID,
		Name: issue.Fields.CustomField10087,
	}
}

const (
	scanStartedTemplate = `A new scan has been initiated in Bitor.

Scan Details:
- Scan ID: {{scan_id}}
- Scan Name: {{scan_name}}
- Start Time: {{start_time}}
- Client: {{client_name}}
- Target Count: {{total_targets}}

This notification was automatically generated by Bitor.
For more details on this automation, see {{jira_link}}`

	scanFinishedTemplate = `A scan was conducted on externally accessible web applications using {{tool}}. This scan was automatically executed from the following IP addresses:

{{scan_ips}}

For more details on this automation, see {{jira_link}}

Tool output is attached to this ticket as a compressed archive format.

Statistics:
{{tool}} Version: {{tool_version}}
Total Targets: {{total_targets}}
Total Skipped Targets: {{skipped_targets}}
Critical Findings: {{critical_findings}}
High Findings: {{high_findings}}
Medium Findings: {{medium_findings}}
Low Findings: {{low_findings}}
Informational Findings: {{info_findings}}
Unknown Findings: {{unknown_findings}}
Total Scan Time: {{scan_time}}

{{#if findings}}
Findings Details:
{{#each findings}}
* {{severity}} - {{title}}
  Target: {{target}}
  Description: {{description}}
{{/each}}
{{/if}}

Scan Details:
- Scan ID: {{scan_id}}
- Scan Name: {{scan_name}}
- Start Time: {{start_time}}
- End Time: {{end_time}}
- Client: {{client_name}}

{{#if additional_notes}}
Additional Notes:
{{additional_notes}}
{{/if}}`

	findingTemplate = `A new {{severity}} finding has been detected in scan {{scan_name}}.

Finding Details:
- Title: {{title}}
- Severity: {{severity}}
- Target: {{target}}

Description:
{{description}}

Scan Details:
- Scan ID: {{scan_id}}
- Client: {{client_name}}
- Detection Time: {{time}}

For more details, please see {{jira_link}}`

	scanFailedTemplate = `A scan has failed in Bitor.

Error Details:
{{error}}

Scan Details:
- Scan ID: {{scan_id}}
- Scan Name: {{scan_name}}
- Client: {{client_name}}
- Start Time: {{start_time}}
- Failure Time: {{time}}

For more details on this automation, see {{jira_link}}`
)

// SendTestNotification creates a test issue in Jira
func (s *JiraService) SendTestNotification(settings types.JiraSettings) error {
	// First, fetch available organizations to validate the ID
	orgs, err := s.GetOrganizations(settings, settings.ProjectKey)
	if err != nil {
		log.Printf("Warning: Failed to fetch organizations: %v", err)
	}

	// Create example data for template
	testData := map[string]string{
		"scan_id":       "TEST-001",
		"scan_name":     "Test Scan",
		"start_time":    time.Now().Format(time.RFC3339),
		"client_name":   "Test Client",
		"total_targets": "5",
		"jira_link":     fmt.Sprintf("%s/browse/%s", settings.JiraURL, settings.ProjectKey),
	}

	// Initialize description variable with template
	var description string = scanStartedTemplate
	for key, value := range testData {
		description = strings.ReplaceAll(description, "{{"+key+"}}", value)
	}

	// Create the Jira ticket payload
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": settings.ProjectKey,
			},
			"summary":     "Test Notification: Scan Started",
			"description": description,
			"issuetype": map[string]string{
				"name": settings.IssueType,
			},
		},
	}

	// Add client mapping information
	if len(settings.ClientMappings) > 0 {
		description = payload["fields"].(map[string]interface{})["description"].(string)
		description += "\n\nConfiguration Details:\n"
		description += fmt.Sprintf("- Jira Project: %s\n", settings.ProjectKey)
		description += fmt.Sprintf("- Issue Type: %s\n", settings.IssueType)
		description += "\nClient Mappings:\n"

		for _, mapping := range settings.ClientMappings {
			description += fmt.Sprintf("- Client ID: %s -> Organization ID: %s\n", mapping.ClientID, mapping.OrganizationID)
			if mapping.OrganizationID != "" {
				var validID bool
				for _, org := range orgs {
					if org.ID == mapping.OrganizationID {
						validID = true
						break
					}
				}

				if !validID {
					log.Printf("Warning: Organization ID %s not found in available organizations", mapping.OrganizationID)
				}

				payload["fields"].(map[string]interface{})["customfield_10002"] = []string{mapping.OrganizationID}
				break
			}
		}
		payload["fields"].(map[string]interface{})["description"] = description
	}

	description = payload["fields"].(map[string]interface{})["description"].(string)
	description += "\n\nThis is a test issue and can be safely deleted."
	payload["fields"].(map[string]interface{})["description"] = description

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling Jira payload: %v", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/rest/api/2/issue", settings.JiraURL)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.SetBasicAuth(settings.Username, settings.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create test issue: status code %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
