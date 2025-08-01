package routes

import (
	"bitor/clients"
	"bitor/findings"
	"bitor/notifications"
	"bitor/providers"
	"bitor/providers/aws"
	"bitor/providers/digitalocean"
	"bitor/scan"
	"bitor/scan/profiles"
	scanTemplates "bitor/scan/templates"
	"bitor/scheduler"
	"bitor/services"
	"bitor/services/notification"
	"bitor/templates"
	"bitor/users"
	"bitor/version"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var scanScheduler *scheduler.ScanScheduler

// InitNotificationService initializes the notification service with settings from the database
func InitNotificationService(app *pocketbase.PocketBase) (*notification.NotificationService, error) {
	// Get notification settings from database
	records, err := app.Dao().FindRecordsByExpr("notification_settings")
	if err != nil {
		return nil, fmt.Errorf("failed to get notification settings: %v", err)
	}

	var config notification.NotificationConfig
	if len(records) > 0 {
		record := records[0]

		// Parse email config
		if emailJson := record.Get("email"); emailJson != nil {
			emailBytes, err := json.Marshal(emailJson)
			if err != nil {
				log.Printf("Failed to marshal email config: %v", err)
				return nil, fmt.Errorf("failed to marshal email config: %v", err)
			}
			if err := json.Unmarshal(emailBytes, &config.Email); err != nil {
				log.Printf("Failed to unmarshal email config: %v", err)
				return nil, fmt.Errorf("failed to parse email config: %v", err)
			}
		}

		// Parse slack config
		if slackJson := record.Get("slack"); slackJson != nil {
			slackBytes, err := json.Marshal(slackJson)
			if err != nil {
				log.Printf("Failed to marshal slack config: %v", err)
				return nil, fmt.Errorf("failed to marshal slack config: %v", err)
			}
			if err := json.Unmarshal(slackBytes, &config.Slack); err != nil {
				log.Printf("Failed to unmarshal slack config: %v", err)
				return nil, fmt.Errorf("failed to parse slack config: %v", err)
			}
		}

		// Parse discord config
		if discordJson := record.Get("discord"); discordJson != nil {
			discordBytes, err := json.Marshal(discordJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal discord config: %v", err)
			}
			if err := json.Unmarshal(discordBytes, &config.Discord); err != nil {
				return nil, fmt.Errorf("failed to parse discord config: %v", err)
			}
		}

		// Parse telegram config
		if telegramJson := record.Get("telegram"); telegramJson != nil {
			telegramBytes, err := json.Marshal(telegramJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal telegram config: %v", err)
			}
			if err := json.Unmarshal(telegramBytes, &config.Telegram); err != nil {
				return nil, fmt.Errorf("failed to parse telegram config: %v", err)
			}
		}

		// Parse Jira config
		if jiraJson := record.Get("jira"); jiraJson != nil {
			jiraBytes, err := json.Marshal(jiraJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal jira config: %v", err)
			}
			if err := json.Unmarshal(jiraBytes, &config.Jira); err != nil {
				return nil, fmt.Errorf("failed to parse jira config: %v", err)
			}
		}

		// Parse rules from the data field
		if dataField := record.Get("data"); dataField != nil {
			var rulesData struct {
				Rules []notification.NotificationRule `json:"rules"`
			}

			// Convert the data to JSON bytes
			var jsonBytes []byte
			switch v := dataField.(type) {
			case string:
				jsonBytes = []byte(v)
			case []byte:
				jsonBytes = v
			default:
				jsonBytes, err = json.Marshal(v)
				if err != nil {
					log.Printf("Error marshaling data field: %v", err)
					return nil, fmt.Errorf("failed to process rules data: %v", err)
				}
			}

			if err := json.Unmarshal(jsonBytes, &rulesData); err != nil {
				log.Printf("Error unmarshaling rules data: %v", err)
				return nil, fmt.Errorf("failed to parse rules data: %v", err)
			}

			config.Rules = rulesData.Rules
			log.Printf("Loaded %d notification rules", len(config.Rules))
		}
	}

	// Create notification service with both app and config
	notificationService, err := notification.NewNotificationService(app, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification service: %v", err)
	}

	// Watch for settings changes
	app.OnRecordAfterUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		if e.Record.Collection().Name == "notification_settings" {
			// Update notification service config
			var newConfig notification.NotificationConfig

			if emailJson := e.Record.Get("email"); emailJson != nil {
				emailBytes, _ := json.Marshal(emailJson)
				if err := json.Unmarshal(emailBytes, &newConfig.Email); err != nil {
					log.Printf("Failed to unmarshal email config: %v", err)
				}
			}
			if slackJson := e.Record.Get("slack"); slackJson != nil {
				slackBytes, _ := json.Marshal(slackJson)
				if err := json.Unmarshal(slackBytes, &newConfig.Slack); err != nil {
					log.Printf("Failed to unmarshal slack config: %v", err)
				}
			}
			if discordJson := e.Record.Get("discord"); discordJson != nil {
				discordBytes, _ := json.Marshal(discordJson)
				if err := json.Unmarshal(discordBytes, &newConfig.Discord); err != nil {
					log.Printf("Failed to unmarshal discord config: %v", err)
				}
			}
			if telegramJson := e.Record.Get("telegram"); telegramJson != nil {
				telegramBytes, _ := json.Marshal(telegramJson)
				if err := json.Unmarshal(telegramBytes, &newConfig.Telegram); err != nil {
					log.Printf("Failed to unmarshal telegram config: %v", err)
				}
			}
			if jiraJson := e.Record.Get("jira"); jiraJson != nil {
				jiraBytes, _ := json.Marshal(jiraJson)
				if err := json.Unmarshal(jiraBytes, &newConfig.Jira); err != nil {
					log.Printf("Failed to unmarshal jira config: %v", err)
				}
			}

			// Parse rules from the data field
			if dataField := e.Record.Get("data"); dataField != nil {
				var rulesData struct {
					Rules []notification.NotificationRule `json:"rules"`
				}

				// Convert the data to JSON bytes
				var jsonBytes []byte
				switch v := dataField.(type) {
				case string:
					jsonBytes = []byte(v)
				case []byte:
					jsonBytes = v
				default:
					jsonBytes, _ = json.Marshal(v)
				}

				if err := json.Unmarshal(jsonBytes, &rulesData); err != nil {
					log.Printf("Failed to unmarshal rules config: %v", err)
				} else {
					newConfig.Rules = rulesData.Rules
					log.Printf("Updated with %d notification rules", len(newConfig.Rules))
				}
			}

			if err := notificationService.UpdateConfig(&newConfig); err != nil {
				log.Printf("Failed to update notification config: %v", err)
			}
		}
		return nil
	})

	return notificationService, nil
}

// RegisterRoutes registers all application routes
func RegisterRoutes(app *pocketbase.PocketBase, ansibleBasePath string, notificationService *notification.NotificationService, e *core.ServeEvent) error {
	log.Printf("RegisterRoutes called with ansible base path: %s", ansibleBasePath)
	log.Printf("Registering all routes...")

	// Initialize finding manager
	findingManager := services.NewFindingManager(app, notificationService)

	// Register findings routes
	RegisterFindingsRoutes(app, e, findingManager)

	// Create a base group for API routes
	apiGroup := e.Router.Group("/api")

	// Register all routes
	providers.RegisterRoutes(app, apiGroup)
	scan.RegisterRoutes(app, e, ansibleBasePath, notificationService)
	findings.RegisterRoutes(app, e, findingManager)
	templates.RegisterRoutes(app, e)
	scanTemplates.RegisterRoutes(app, apiGroup)
	version.RegisterRoutes(e)
	notifications.RegisterRoutes(app, apiGroup)
	profiles.RegisterRoutes(app, apiGroup)
	log.Printf("Registering users routes...")
	users.RegisterRoutes(app, e)
	log.Printf("Users routes registered")
	clients.RegisterRoutes(app, e)
	log.Printf("Client routes registered")

	// Register AWS provider routes
	aws.RegisterRoutes(e, apiGroup)

	// Register DigitalOcean provider routes
	digitalocean.RegisterRoutes(e, apiGroup)

	// Initialize collections
	if err := users.EnsureInvitationsCollection(app); err != nil {
		log.Fatal(err)
	}

	if err := notifications.EnsureNotificationsCollection(app); err != nil {
		log.Fatal(err)
	}

	// Apply email settings from the database
	if err := notifications.ApplyEmailSettings(app); err != nil {
		log.Printf("Failed to apply email settings: %v", err)
	}

	// Start the scan scheduler with the ansible base path
	scanScheduler = scheduler.NewScanScheduler(app, ansibleBasePath)
	log.Printf("Starting scan scheduler with ansible base path: %s", ansibleBasePath)
	scanScheduler.Start()
	log.Println("Scan Scheduler started.")

	// Start the cost calculation scheduler
	if _, err := scheduler.StartScheduler(app); err != nil {
		log.Printf("Error starting cost calculation scheduler: %v", err)
	} else {
		log.Println("Cost calculation scheduler started.")
	}

	return nil
}

// StopScheduler stops the scan scheduler
func StopScheduler() {
	if scanScheduler != nil {
		scanScheduler.Stop()
		log.Println("Scan Scheduler stopped.")
	}
}
