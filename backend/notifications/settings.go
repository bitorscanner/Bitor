package notifications

import (
	"log"
	"net/mail"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func EnsureNotificationsCollection(app *pocketbase.PocketBase) error {
	// Check if collection already exists
	collection, err := app.Dao().FindCollectionByNameOrId("notification_settings")
	if err == nil {
		// Collection exists, update the rules
		updateRule := "@request.auth.manage_notifications = true"
		collection.ListRule = &updateRule
		collection.ViewRule = &updateRule
		collection.CreateRule = &updateRule
		collection.UpdateRule = &updateRule
		collection.DeleteRule = &updateRule
		return app.Dao().SaveCollection(collection)
	}

	// Collection doesn't exist, create it
	updateRule := "@request.auth.manage_notifications = true"
	notificationSettings := &models.Collection{
		Name:       "notification_settings",
		Type:       models.CollectionTypeBase,
		System:     false,
		ListRule:   &updateRule,
		ViewRule:   &updateRule,
		CreateRule: &updateRule,
		UpdateRule: &updateRule,
		DeleteRule: &updateRule,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "email_enabled",
				Type:     schema.FieldTypeBool,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "smtp_host",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "smtp_port",
				Type:     schema.FieldTypeNumber,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "smtp_username",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "smtp_password",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "smtp_encryption",
				Type:     schema.FieldTypeSelect,
				Required: false,
				Options: &schema.SelectOptions{
					MaxSelect: 1,
					Values:    []string{"none", "ssl", "tls"},
				},
			},
			&schema.SchemaField{
				Name:     "from_address",
				Type:     schema.FieldTypeEmail,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "from_name",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_enabled",
				Type:     schema.FieldTypeBool,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "jira_url",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_username",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_api_token",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_project_key",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_issue_type",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "jira_template",
				Type:     schema.FieldTypeText,
				Required: false,
			},
		),
	}

	return app.Dao().SaveCollection(notificationSettings)
}

func ApplyEmailSettings(app *pocketbase.PocketBase) error {
	// Get all settings records
	records, err := app.Dao().FindRecordsByExpr("notification_settings")
	if err != nil {
		log.Printf("Error querying notification settings: %v", err)
		return err
	}

	// Log how many records we found
	log.Printf("Found %d notification settings records", len(records))

	// Use the first record if any exist
	var settings *models.Record
	if len(records) > 0 {
		settings = records[0]
		log.Printf("Using settings record with ID: %s", settings.Id)
	} else {
		log.Printf("No notification settings records found")
		return apis.NewBadRequestError("No notification settings found", nil)
	}

	// Only apply settings if email is enabled
	if settings.GetBool("email_enabled") {
		// Configure SMTP settings
		app.Settings().Smtp.Enabled = true
		app.Settings().Smtp.Host = settings.GetString("smtp_host")
		app.Settings().Smtp.Port = int(settings.GetInt("smtp_port"))
		app.Settings().Smtp.Username = settings.GetString("smtp_username")
		app.Settings().Smtp.Password = settings.GetString("smtp_password")

		// Set encryption based on the selected option
		encryption := settings.GetString("smtp_encryption")
		app.Settings().Smtp.Tls = encryption == "tls"

		// Set sender information
		app.Settings().Meta.SenderName = settings.GetString("from_name")
		app.Settings().Meta.SenderAddress = settings.GetString("from_address")

		log.Printf("SMTP settings applied successfully: enabled=%v, host=%s, port=%d, username=%s",
			app.Settings().Smtp.Enabled,
			app.Settings().Smtp.Host,
			app.Settings().Smtp.Port,
			app.Settings().Smtp.Username)
	} else {
		app.Settings().Smtp.Enabled = false
		log.Println("SMTP is disabled in settings")
	}

	return nil
}

// ValidateSmtpSettings checks if SMTP is properly configured
func ValidateSmtpSettings(app *pocketbase.PocketBase) error {
	// Log current SMTP state
	log.Printf("Validating SMTP settings: enabled=%v, host=%s, port=%d, sender=%s",
		app.Settings().Smtp.Enabled,
		app.Settings().Smtp.Host,
		app.Settings().Smtp.Port,
		app.Settings().Meta.SenderAddress)

	if !app.Settings().Smtp.Enabled {
		return apis.NewBadRequestError("SMTP is not enabled", nil)
	}

	if app.Settings().Smtp.Host == "" {
		return apis.NewBadRequestError("SMTP host is not configured", nil)
	}

	if app.Settings().Smtp.Port == 0 {
		return apis.NewBadRequestError("SMTP port is not configured", nil)
	}

	if app.Settings().Meta.SenderAddress == "" {
		return apis.NewBadRequestError("Sender email address is not configured", nil)
	}

	return nil
}

// Function to send a test email
func SendTestEmail(app *pocketbase.PocketBase, to string) error {
	if !app.Settings().Smtp.Enabled {
		return apis.NewBadRequestError("SMTP is not enabled", nil)
	}

	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      []mail.Address{{Address: to}},
		Subject: "Bitor Email Test",
		HTML:    "This is a test email from your bitor installation.",
	}

	log.Printf("Sending test email to %s from %s", to, app.Settings().Meta.SenderAddress)
	return app.NewMailClient().Send(message)
}
