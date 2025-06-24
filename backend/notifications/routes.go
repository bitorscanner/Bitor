package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"bitor/providers/jira"
	"bitor/services"
	"bitor/types"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

// providerIDMiddleware extracts provider_id from request body and sets it in context
func providerIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Read the body
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return apis.NewBadRequestError("Failed to read request body", err)
		}

		// Parse the request body
		var req struct {
			ProviderID string `json:"provider_id"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return apis.NewBadRequestError("Invalid request body", err)
		}

		// Set the provider ID in context
		c.Set("provider_id", req.ProviderID)

		// Reset the body for the next middleware/handler
		c.Request().Body = ioutil.NopCloser(bytes.NewReader(body))

		return next(c)
	}
}

func RegisterRoutes(app *pocketbase.PocketBase, g *echo.Group) {
	log.Printf("Registering notification routes...")

	// Initialize Jira service and handler
	jiraService := services.NewJiraService()
	jiraHandler := NewJiraHandler(app, jiraService)

	// Register Jira routes with provider ID middleware
	g.POST("/jira/projects", providerIDMiddleware(jiraHandler.GetProjects), apis.RequireAdminOrRecordAuth())
	g.POST("/jira/issuetypes", providerIDMiddleware(jiraHandler.GetIssueTypes), apis.RequireAdminOrRecordAuth())
	g.POST("/jira/organizations", providerIDMiddleware(jiraHandler.GetOrganizations), apis.RequireAdminOrRecordAuth())
	log.Printf("Registered Jira routes")

	// Add route to get notification templates
	g.GET("/notification-templates", func(c echo.Context) error {
		templates := map[string]string{
			"scan_started":  services.ScanStartedTemplate,
			"scan_finished": services.ScanFinishedTemplate,
			"scan_failed":   services.ScanFailedTemplate,
			"scan_stopped":  services.ScanStoppedTemplate,
			"finding":       services.FindingTemplate,
		}
		return c.JSON(http.StatusOK, templates)
	}, apis.RequireAdminOrRecordAuth())

	// Apply email settings handler
	g.POST("/apply-email-settings", func(c echo.Context) error {
		if err := ApplyEmailSettings(app); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to apply email settings", err)
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Email settings applied successfully",
		})
	}, apis.RequireAdminOrRecordAuth())

	// Test notification handler
	g.POST("/test-notification", func(c echo.Context) error {
		var req struct {
			RuleID string `json:"rule_id"`
		}

		if err := c.Bind(&req); err != nil {
			return apis.NewBadRequestError("Invalid request body", err)
		}

		// Get notification settings
		records, err := app.Dao().FindRecordsByExpr("notification_settings")
		if err != nil || len(records) == 0 {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to get notification settings", err)
		}

		log.Printf("Found notification settings record: %v", records[0].Id)

		// Get rules from settings
		var rulesData struct {
			Rules []map[string]interface{} `json:"rules"`
		}

		// Get the data field
		dataField := records[0].Get("data")
		log.Printf("Data field raw value: %+v", dataField)

		if dataField != nil {
			// Convert the data to JSON bytes
			var jsonBytes []byte
			switch v := dataField.(type) {
			case string:
				jsonBytes = []byte(v)
			case []byte:
				jsonBytes = v
			default:
				// For types.JsonRaw or any other type, marshal it back to JSON
				var err error
				jsonBytes, err = json.Marshal(v)
				if err != nil {
					log.Printf("Error marshaling data field: %v", err)
					return apis.NewApiError(http.StatusInternalServerError, "Failed to process rules data", err)
				}
			}

			// Unmarshal the JSON bytes into our rules structure
			if err := json.Unmarshal(jsonBytes, &rulesData); err != nil {
				log.Printf("Error unmarshaling rules data: %v", err)
				return apis.NewApiError(http.StatusInternalServerError, "Failed to parse rules data", err)
			}
			log.Printf("Successfully parsed rules: %+v", rulesData)
		}

		// Find the specific rule
		var targetRule map[string]interface{}
		for _, rule := range rulesData.Rules {
			if ruleID, ok := rule["id"].(string); ok {
				log.Printf("Comparing rule ID %s with requested ID %s", ruleID, req.RuleID)
				if ruleID == req.RuleID {
					targetRule = rule
					break
				}
			}
		}

		if targetRule == nil {
			log.Printf("Rule with ID %s not found in rules list", req.RuleID)
			return apis.NewNotFoundError("Rule not found", nil)
		}

		log.Printf("Found target rule: %v", targetRule)

		// Get channels from the rule
		channels, ok := targetRule["channels"].([]interface{})
		if !ok {
			return apis.NewApiError(http.StatusInternalServerError, "Invalid channels configuration", nil)
		}

		// For each channel, get the provider and send a test notification
		for _, channel := range channels {
			providerID, ok := channel.(string)
			if !ok {
				continue
			}

			// Get provider details
			provider, err := app.Dao().FindRecordById("providers", providerID)
			if err != nil {
				log.Printf("Error finding provider %s: %v", providerID, err)
				continue
			}

			switch provider.Get("provider_type").(string) {
			case "jira":
				// Get provider settings
				settingsRaw := provider.Get("settings")
				log.Printf("Provider settings (raw): %+v", settingsRaw)

				// Parse settings
				var settingsMap map[string]interface{}
				switch v := settingsRaw.(type) {
				case string:
					if err := json.Unmarshal([]byte(v), &settingsMap); err != nil {
						log.Printf("Error parsing settings JSON: %v", err)
						continue
					}
				case map[string]interface{}:
					settingsMap = v
				default:
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						log.Printf("Error marshaling settings: %v", err)
						continue
					}
					if err := json.Unmarshal(jsonBytes, &settingsMap); err != nil {
						log.Printf("Error parsing settings: %v", err)
						continue
					}
				}

				log.Printf("Parsed settings map: %+v", settingsMap)

				// Get credentials
				username, apiKey, err := jira.GetCredentials(app, provider)
				if err != nil {
					log.Printf("Error getting credentials: %v", err)
					return apis.NewApiError(http.StatusInternalServerError, fmt.Sprintf("Failed to get Jira credentials: %v", err), err)
				}

				// Extract client mappings
				var clientMappings []types.JiraClientMapping
				if mappingsRaw, ok := settingsMap["client_mappings"].([]interface{}); ok {
					for _, mapping := range mappingsRaw {
						if mappingMap, ok := mapping.(map[string]interface{}); ok {
							clientMapping := types.JiraClientMapping{
								ClientID:       fmt.Sprint(mappingMap["client_id"]),
								OrganizationID: fmt.Sprint(mappingMap["organization_id"]),
							}
							clientMappings = append(clientMappings, clientMapping)
						}
					}
				}

				// Create Jira settings
				jiraSettings := types.JiraSettings{
					JiraURL:        fmt.Sprint(settingsMap["jira_url"]),
					Username:       username,
					APIKey:         apiKey,
					ProjectKey:     fmt.Sprint(settingsMap["project_key"]),
					IssueType:      fmt.Sprint(settingsMap["issue_type"]),
					ClientMappings: clientMappings,
				}

				log.Printf("Sending test notification with settings: %+v", jiraSettings)
				if err := jiraService.SendTestNotification(jiraSettings); err != nil {
					log.Printf("Error sending test Jira notification: %v", err)
					return apis.NewApiError(http.StatusInternalServerError, fmt.Sprintf("Failed to send Jira notification: %v", err), err)
				}
				log.Printf("Successfully sent Jira test notification")

			case "email":
				fromAddress := provider.Get("settings.from_address")
				if fromAddress == nil {
					log.Printf("Error: from_address is not set for email provider %s", providerID)
					continue
				}
				if err := SendTestEmail(app, fromAddress.(string)); err != nil {
					log.Printf("Error sending test email: %v", err)
				}
			case "slack", "discord", "telegram":
				// TODO: Implement other providers
				log.Printf("%s test notification not implemented yet", provider.Get("provider_type"))
			}
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Test notifications sent successfully",
		})
	}, apis.RequireAdminOrRecordAuth())

	testEmailHandler := func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)

		if admin == nil && (authRecord == nil || authRecord.Collection().Name != "users") {
			return apis.NewUnauthorizedError("Only authenticated users can send test emails", nil)
		}

		var email string
		// Try to get email from query parameter first (GET request)
		email = c.QueryParam("email")
		if email == "" {
			// If not in query, try to get from JSON body (POST request)
			var data struct {
				Email string `json:"email"`
			}
			if err := c.Bind(&data); err == nil {
				email = data.Email
			}
		}

		if email == "" {
			return apis.NewBadRequestError("Email address is required", nil)
		}

		// Apply email settings before sending test email
		if err := ApplyEmailSettings(app); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to apply email settings", err)
		}

		if err := SendTestEmail(app, email); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to send test email", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Test email sent successfully",
		})
	}

	// Register both GET and POST handlers
	g.GET("/test-email", testEmailHandler, apis.RequireAdminOrRecordAuth())
	g.POST("/test-email", testEmailHandler, apis.RequireAdminOrRecordAuth())
}
