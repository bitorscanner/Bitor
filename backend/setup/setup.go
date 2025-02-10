package setup

import (
	"encoding/json"
	"fmt"
	"log"
	"orbit/services/notification"
	"orbit/utils/crypto"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	schema "github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

var Version = "development"
var ansibleBasePath string

// Initialize encryption key if not set
func init() {
	// Removed duplicate encryption key initialization since we're using utils/crypto
}

// Add hook for encrypting API keys before creation
func SetupEncryption(app *pocketbase.PocketBase) {
	app.OnModelBeforeCreate().Add(func(e *core.ModelEvent) error {
		collection, err := app.Dao().FindCollectionByNameOrId("api_keys")
		if err != nil || collection == nil || e.Model.TableName() != collection.Name {
			return nil
		}

		record := e.Model.(*models.Record)

		// Encrypt key field
		if key, ok := record.Get("key").(string); ok && key != "" {
			encryptedKey, err := crypto.Encrypt([]byte(key), os.Getenv("API_ENCRYPTION_KEY"))
			if err != nil {
				return fmt.Errorf("failed to encrypt key: %v", err)
			}
			record.Set("key", encryptedKey)
		}

		// Encrypt key_data field if it exists
		if keyData, ok := record.Get("key_data").(map[string]interface{}); ok {
			for k, v := range keyData {
				if str, ok := v.(string); ok {
					encryptedStr, err := crypto.Encrypt([]byte(str), os.Getenv("API_ENCRYPTION_KEY"))
					if err != nil {
						return fmt.Errorf("failed to encrypt key_data value: %v", err)
					}
					keyData[k] = encryptedStr
				}
			}
			record.Set("key_data", keyData)
		}

		return nil
	})
}

type FileData struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Size   int    `json:"size"`
	Base64 string `json:"base64"`
}

// initNotificationService initializes the notification service.
// Currently unused but retained for future use in notification system setup.
// It will be used when implementing the notification service integration.
// nolint:unused
func initNotificationService(app *pocketbase.PocketBase) (*notification.NotificationService, error) {
	// Create empty config
	config := notification.NotificationConfig{}

	// Try to get notification settings from database
	records, err := app.Dao().FindRecordsByExpr("notification_settings")
	if err != nil {
		// If collection doesn't exist, just return empty config
		if err.Error() == "sql: no rows in result set" {
			notificationService, err := notification.NewNotificationService(&config)
			if err != nil {
				return nil, fmt.Errorf("failed to create notification service: %v", err)
			}
			return notificationService, nil
		}
		return nil, fmt.Errorf("failed to get notification settings: %v", err)
	}

	if len(records) > 0 {
		record := records[0]

		// Parse email config
		if emailJson := record.Get("email"); emailJson != nil {
			emailBytes, err := json.Marshal(emailJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal email config: %v", err)
			}
			if err := json.Unmarshal(emailBytes, &config.Email); err != nil {
				return nil, fmt.Errorf("failed to parse email config: %v", err)
			}
		}

		// Parse slack config
		if slackJson := record.Get("slack"); slackJson != nil {
			slackBytes, err := json.Marshal(slackJson)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal slack config: %v", err)
			}
			if err := json.Unmarshal(slackBytes, &config.Slack); err != nil {
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
				emailBytes, err := json.Marshal(emailJson)
				if err != nil {
					log.Printf("Failed to marshal email config: %v", err)
					return nil
				}
				if err := json.Unmarshal(emailBytes, &newConfig.Email); err != nil {
					log.Printf("Failed to parse email config: %v", err)
				}
			}
			if slackJson := e.Record.Get("slack"); slackJson != nil {
				slackBytes, err := json.Marshal(slackJson)
				if err != nil {
					log.Printf("Failed to marshal slack config: %v", err)
					return nil
				}
				if err := json.Unmarshal(slackBytes, &newConfig.Slack); err != nil {
					log.Printf("Failed to parse slack config: %v", err)
				}
			}
			if discordJson := e.Record.Get("discord"); discordJson != nil {
				discordBytes, err := json.Marshal(discordJson)
				if err != nil {
					log.Printf("Failed to marshal discord config: %v", err)
					return nil
				}
				if err := json.Unmarshal(discordBytes, &newConfig.Discord); err != nil {
					log.Printf("Failed to parse discord config: %v", err)
				}
			}
			if telegramJson := e.Record.Get("telegram"); telegramJson != nil {
				telegramBytes, err := json.Marshal(telegramJson)
				if err != nil {
					log.Printf("Failed to marshal telegram config: %v", err)
					return nil
				}
				if err := json.Unmarshal(telegramBytes, &newConfig.Telegram); err != nil {
					log.Printf("Failed to parse telegram config: %v", err)
				}
			}
			if rulesJson := e.Record.Get("rules"); rulesJson != nil {
				rulesBytes, err := json.Marshal(rulesJson)
				if err != nil {
					log.Printf("Failed to marshal rules: %v", err)
					return nil
				}
				if err := json.Unmarshal(rulesBytes, &newConfig.Rules); err != nil {
					log.Printf("Failed to parse rules: %v", err)
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

func initializePublicSettings(app *pocketbase.PocketBase) error {
	// Try to find settings_public collection
	collection, err := app.Dao().FindCollectionByNameOrId("settings_public")
	if err != nil {
		// Collection doesn't exist, create it
		collection = &models.Collection{
			Name:     "settings_public",
			Type:     models.CollectionTypeBase,
			ListRule: types.Pointer(""), // Empty means public access
			ViewRule: types.Pointer(""), // Empty means public access
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "setup_completed",
					Type:     schema.FieldTypeBool,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "initial_setup_done",
					Type:     schema.FieldTypeBool,
					Required: true,
				},
			),
		}

		if err := app.Dao().SaveCollection(collection); err != nil {
			return fmt.Errorf("failed to create settings_public collection: %v", err)
		}
	}

	// Try to find an existing settings record
	_, err = app.Dao().FindFirstRecordByData("settings_public", "id", "default")
	if err != nil {
		// Only create a new record if none exists
		setupRecord := models.NewRecord(collection)
		setupRecord.Set("id", "default")
		setupRecord.Set("setup_completed", false)
		setupRecord.Set("initial_setup_done", false)

		if err := app.Dao().SaveRecord(setupRecord); err != nil {
			return fmt.Errorf("failed to create initial public settings: %v", err)
		}

		// Set default app name only on first initialization
		settings := app.Settings()
		settings.Meta.AppName = "Orbit"
		if err := app.Dao().SaveSettings(settings); err != nil {
			return fmt.Errorf("failed to set default app name: %v", err)
		}

		// Mark initial setup as done
		setupRecord.Set("initial_setup_done", true)
		if err := app.Dao().SaveRecord(setupRecord); err != nil {
			return fmt.Errorf("failed to mark initial setup as done: %v", err)
		}
	}

	return nil
}

func InitializeApp(app *pocketbase.PocketBase) error {
	// Bootstrap the app (initializes the database)
	if err := app.Bootstrap(); err != nil {
		return err
	}

	// Setup encryption for API keys
	SetupEncryption(app)

	// Initialize public settings if they don't exist
	if err := initializePublicSettings(app); err != nil {
		return fmt.Errorf("failed to initialize public settings: %v", err)
	}

	// Add hook to prevent admin creation after setup is completed
	app.OnModelBeforeCreate().Add(func(e *core.ModelEvent) error {
		if e.Model.TableName() != "_admins" {
			return nil
		}

		// Check if setup is completed
		setupRecord, err := app.Dao().FindFirstRecordByData("settings_public", "id", "default")
		if err != nil {
			return fmt.Errorf("failed to check setup status: %v", err)
		}

		setupCompleted, _ := setupRecord.Get("setup_completed").(bool)
		if setupCompleted {
			// Count existing admins
			var count int
			err := app.Dao().DB().Select("count(*)").From("_admins").Row(&count)
			if err != nil {
				return fmt.Errorf("failed to check existing admins: %v", err)
			}

			if count > 0 {
				return fmt.Errorf("setup has already been completed, additional admin accounts can only be created by existing admins")
			}
		}

		return nil
	})

	// Add hook for deleting associated API keys when a provider is deleted
	app.OnModelBeforeDelete().Add(func(e *core.ModelEvent) error {
		collection, err := app.Dao().FindCollectionByNameOrId("providers")
		if err != nil || collection == nil || e.Model.TableName() != collection.Name {
			return nil
		}

		// Find and delete all API keys associated with this provider
		provider := e.Model.(*models.Record)
		apiKeys, err := app.Dao().FindRecordsByExpr("api_keys", dbx.HashExp{"provider": provider.Id})
		if err != nil {
			return fmt.Errorf("failed to find associated API keys: %v", err)
		}

		for _, key := range apiKeys {
			if err := app.Dao().DeleteRecord(key); err != nil {
				return fmt.Errorf("failed to delete associated API key: %v", err)
			}
		}

		return nil
	})

	return nil
}

func Setup(app *pocketbase.PocketBase) error {
	// Get the ansible base path from environment or use default
	if envPath := os.Getenv("ANSIBLE_BASE_PATH"); envPath != "" {
		// Convert to absolute path if it's not already
		if !filepath.IsAbs(envPath) {
			if absPath, err := filepath.Abs(envPath); err == nil {
				ansibleBasePath = absPath
			} else {
				ansibleBasePath = envPath
			}
		} else {
			ansibleBasePath = envPath
		}
	} else {
		// Use default path
		workspaceRoot, err := filepath.Abs(filepath.Join("..", ""))
		if err != nil {
			log.Printf("Error getting workspace root: %v", err)
			// Always use ansible in current directory
			ansibleBasePath = "ansible"
		} else {
			// Use absolute path
			ansibleBasePath = filepath.Join(workspaceRoot, "ansible")
		}
	}

	log.Printf("Using ansible base path in setup: %s", ansibleBasePath)

	// Ensure ansible base path exists
	if _, err := os.Stat(ansibleBasePath); os.IsNotExist(err) {
		log.Printf("Creating ansible base path: %s", ansibleBasePath)
		if err := os.MkdirAll(ansibleBasePath, 0755); err != nil {
			log.Printf("Failed to create ansible base path: %v", err)
			return err
		}
	}

	return nil
}

func EnsureGroupsCollection(app *pocketbase.PocketBase) error {
	// Create default groups
	defaultGroups := []map[string]interface{}{
		{
			"name":        "admin",
			"description": "Full system access",
			"permissions": map[string]interface{}{
				"read":                 []string{"*"},
				"write":                []string{"*"},
				"delete":               []string{"*"},
				"manage_users":         true,
				"manage_groups":        true,
				"manage_providers":     true,
				"manage_api_keys":      true,
				"manage_notifications": true,
				"settings":             true,
			},
		},
		{
			"name":        "manager",
			"description": "Can manage users and content",
			"permissions": map[string]interface{}{
				"read":                 []string{"clients", "findings", "scan_profiles", "users", "nuclei_scans", "client_groups", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"write":                []string{"clients", "findings", "scan_profiles", "nuclei_scans", "client_groups", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"delete":               []string{"clients", "findings", "nuclei_scans", "client_groups", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"manage_users":         true,
				"manage_groups":        false,
				"manage_providers":     false,
				"manage_api_keys":      false,
				"manage_notifications": true,
				"settings":             false,
			},
		},
		{
			"name":        "editor",
			"description": "Can edit content",
			"permissions": map[string]interface{}{
				"read":                 []string{"clients", "findings", "scan_profiles", "nuclei_scans", "client_groups", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"write":                []string{"findings", "scan_profiles", "nuclei_scans", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"delete":               []string{},
				"manage_users":         false,
				"manage_groups":        false,
				"manage_providers":     false,
				"manage_api_keys":      false,
				"manage_notifications": false,
				"settings":             false,
			},
		},
		{
			"name":        "viewer",
			"description": "Read-only access",
			"permissions": map[string]interface{}{
				"read":                 []string{"clients", "findings", "scan_profiles", "nuclei_scans", "client_groups", "nuclei_targets", "nuclei_interact", "nuclei_profiles"},
				"write":                []string{},
				"delete":               []string{},
				"manage_users":         false,
				"manage_groups":        false,
				"manage_providers":     false,
				"manage_api_keys":      false,
				"manage_notifications": false,
				"settings":             false,
			},
		},
	}

	// Create each default group if it doesn't exist
	for _, group := range defaultGroups {
		// Just check if the group exists
		_, err := app.Dao().FindFirstRecordByData("groups", "name", group["name"])
		if err != nil {
			collection, err := app.Dao().FindCollectionByNameOrId("groups")
			if err != nil {
				return err
			}

			newRecord := models.NewRecord(collection)
			newRecord.Set("name", group["name"])
			newRecord.Set("description", group["description"])
			newRecord.Set("permissions", group["permissions"])

			if err := app.Dao().SaveRecord(newRecord); err != nil {
				log.Printf("Error creating group %s: %v", group["name"], err)
				return err
			}
		}
	}

	return nil
}
