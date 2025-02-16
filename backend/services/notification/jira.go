package notification

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
)

type JiraService struct {
	config *JiraConfig
}

type JiraIssue struct {
	Fields struct {
		Project struct {
			Key string `json:"key"`
		} `json:"project"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		IssueType   struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Organization []struct {
			ID string `json:"id"`
		} `json:"organization"`
	} `json:"fields"`
}

func NewJiraService(config *JiraConfig) *JiraService {
	return &JiraService{
		config: config,
	}
}

func (s *JiraService) CreateIssue(ctx context.Context, subject, message, organizationID string) error {
	if !s.config.Enabled {
		return nil
	}

	// Parse template if provided, otherwise use default format
	var description string
	if s.config.Template != "" {
		tmpl, err := template.New("jira").Parse(s.config.Template)
		if err != nil {
			return fmt.Errorf("failed to parse template: %v", err)
		}

		data := struct {
			Subject string
			Message string
		}{
			Subject: subject,
			Message: message,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template: %v", err)
		}
		description = buf.String()
	} else {
		description = fmt.Sprintf("%s\n\n%s", subject, message)
	}

	issue := JiraIssue{}
	issue.Fields.Project.Key = s.config.ProjectKey
	issue.Fields.Summary = subject
	issue.Fields.Description = description
	issue.Fields.IssueType.Name = s.config.IssueType

	// Only add organization field if a valid ID is provided
	if organizationID != "" && organizationID != "0" && organizationID != "undefined" {
		log.Printf("Adding organization ID to Jira issue: %s", organizationID)
		// Use customfield_10002 instead of organization field
		jsonData, err := json.Marshal(map[string]interface{}{
			"fields": map[string]interface{}{
				"project": map[string]string{
					"key": s.config.ProjectKey,
				},
				"summary":     subject,
				"description": description,
				"issuetype": map[string]string{
					"name": s.config.IssueType,
				},
				"customfield_10002": []string{organizationID},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to marshal issue: %v", err)
		}
		log.Printf("Creating Jira issue with payload: %s", string(jsonData))
		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/rest/api/2/issue", s.config.URL), bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}
		// Add headers
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.config.Username, s.config.APIToken)))
		req.Header.Set("Authorization", "Basic "+auth)
		req.Header.Set("Content-Type", "application/json")
		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create issue: status code %d, body: %s", resp.StatusCode, string(body))
		}
		return nil
	} else {
		log.Printf("Skipping organization field as no valid ID was provided")
		// Create regular issue without organization field
		jsonData, err := json.Marshal(map[string]interface{}{
			"fields": map[string]interface{}{
				"project": map[string]string{
					"key": s.config.ProjectKey,
				},
				"summary":     subject,
				"description": description,
				"issuetype": map[string]string{
					"name": s.config.IssueType,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to marshal issue: %v", err)
		}
		log.Printf("Creating Jira issue with payload: %s", string(jsonData))
		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/rest/api/2/issue", s.config.URL), bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}
		// Add headers
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.config.Username, s.config.APIToken)))
		req.Header.Set("Authorization", "Basic "+auth)
		req.Header.Set("Content-Type", "application/json")
		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to create issue: status code %d, body: %s", resp.StatusCode, string(body))
		}
		return nil
	}
}
