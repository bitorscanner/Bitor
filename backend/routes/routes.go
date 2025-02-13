package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"orbit/findings"
	"orbit/notifications"
	"orbit/providers"
	"orbit/providers/aws"
	"orbit/providers/digitalocean"
	"orbit/scan"
	"orbit/scan/profiles"
	"orbit/scheduler"
	"orbit/services/notification"
	"orbit/templates"
	"orbit/users"
	"orbit/version"

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

		// Parse rules
		if rulesJson := record.Get("rules"); rulesJson != nil {
			rulesBytes, err := json.Marshal(rulesJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal rules: %v", err)
			}
			if err := json.Unmarshal(rulesBytes, &config.Rules); err != nil {
				return nil, fmt.Errorf("failed to parse notification rules: %v", err)
			}
		}
	}

	// Create notification service
	notificationService, err := notification.NewNotificationService(&config)
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
			if rulesJson := e.Record.Get("rules"); rulesJson != nil {
				rulesBytes, _ := json.Marshal(rulesJson)
				if err := json.Unmarshal(rulesBytes, &newConfig.Rules); err != nil {
					log.Printf("Failed to unmarshal rules config: %v", err)
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

	// Create a base group for API routes
	apiGroup := e.Router.Group("/api")

	providers.RegisterRoutes(app, apiGroup)
	scan.RegisterRoutes(app, e, ansibleBasePath, notificationService)
	findings.RegisterRoutes(app, e)
	templates.RegisterRoutes(app, e)
	version.RegisterRoutes(e)
	notifications.RegisterRoutes(app, apiGroup)
	profiles.RegisterRoutes(app, apiGroup)
	log.Printf("Registering users routes...")
	users.RegisterRoutes(app, e)
	log.Printf("Users routes registered")

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
