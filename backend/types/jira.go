package types

// JiraClientMapping represents the mapping between a client and their Jira project
type JiraClientMapping struct {
	ClientID       string `json:"client_id"`
	OrganizationID string `json:"organization_id"`
}

// JiraSettings contains the configuration for Jira integration
type JiraSettings struct {
	JiraURL        string              `json:"jira_url"`
	Username       string              `json:"username"`
	APIKey         string              `json:"api_key"`
	ProjectKey     string              `json:"project_key"`
	IssueType      string              `json:"issue_type"`
	ClientMappings []JiraClientMapping `json:"client_mappings,omitempty"`
}

// JiraProject represents a Jira project
type JiraProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// JiraIssueType represents a Jira issue type
type JiraIssueType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// JiraCredentials represents the credentials needed for Jira API access
type JiraCredentials struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	APIKey   string `json:"api_key"`
}

// JiraProjectRequest represents a request to fetch Jira projects
type JiraProjectRequest struct {
	JiraCredentials
}

// JiraIssueTypeRequest represents a request to fetch Jira issue types
type JiraIssueTypeRequest struct {
	JiraCredentials
	ProjectKey string `json:"project_key"`
}

type JiraOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// JiraCustomer represents a Jira customer with ID and name from custom fields
type JiraCustomer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// JiraIssueFields represents the fields in a Jira issue
type JiraIssueFields struct {
	CustomField10002 []struct {
		ID string `json:"id"`
	} `json:"customfield_10002"`
	CustomField10087 string `json:"customfield_10087"`
}

// JiraIssue represents a Jira issue with its fields
type JiraIssue struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Fields JiraIssueFields `json:"fields"`
}
