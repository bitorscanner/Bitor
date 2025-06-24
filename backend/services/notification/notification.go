package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"bitor/providers/jira"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/slack"
	"github.com/nikoksr/notify/service/telegram"
	"github.com/pocketbase/pocketbase"
)

// NotificationEvent represents a notification event type
type NotificationEvent string

const (
	ScanStarted  NotificationEvent = "scan_started"
	ScanFinished NotificationEvent = "scan_finished"
	ScanFailed   NotificationEvent = "scan_failed"
	ScanStopped  NotificationEvent = "scan_stopped"
	Finding      NotificationEvent = "finding"
)

// NotificationService handles sending notifications through various channels
type NotificationService struct {
	notifier *notify.Notify
	config   *NotificationConfig
	jira     *JiraService
	app      *pocketbase.PocketBase
	mu       sync.RWMutex
}

type NotificationConfig struct {
	Email    *EmailConfig       `json:"email"`
	Slack    *SlackConfig       `json:"slack"`
	Discord  *DiscordConfig     `json:"discord"`
	Telegram *TelegramConfig    `json:"telegram"`
	Jira     *JiraConfig        `json:"jira"`
	Rules    []NotificationRule `json:"rules"`
}

type EmailConfig struct {
	Enabled  bool     `json:"enabled"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

type SlackConfig struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
}

type DiscordConfig struct {
	Enabled   bool   `json:"enabled"`
	WebhookID string `json:"webhook_id"`
	Token     string `json:"token"`
}

type TelegramConfig struct {
	Enabled bool    `json:"enabled"`
	Token   string  `json:"token"`
	ChatIDs []int64 `json:"chat_ids"`
}

type JiraConfig struct {
	Enabled    bool   `json:"enabled"`
	URL        string `json:"url"`
	Username   string `json:"username"`
	APIToken   string `json:"api_token"`
	ProjectKey string `json:"project_key"`
	IssueType  string `json:"issue_type"`
	Template   string `json:"template"`
}

type NotificationRule struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Severity []string `json:"severity"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}

// NewNotificationService creates a new instance of NotificationService
func NewNotificationService(app *pocketbase.PocketBase, config *NotificationConfig) (*NotificationService, error) {
	n := &NotificationService{
		notifier: notify.New(),
		config:   config,
		app:      app,
	}

	if err := n.configureServices(); err != nil {
		return nil, fmt.Errorf("failed to configure notification services: %v", err)
	}

	// Configure Jira service if enabled
	if config.Jira != nil && config.Jira.Enabled {
		n.jira = NewJiraService(config.Jira)
	}

	return n, nil
}

// configureServices sets up the notification services based on configuration
func (n *NotificationService) configureServices() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Create a new notifier instance
	n.notifier = notify.New()

	// Configure Email
	if n.config.Email != nil && n.config.Email.Enabled {
		mailService := mail.New(n.config.Email.Username, n.config.Email.Password)
		mailService.AddReceivers(n.config.Email.To...)
		n.notifier.UseServices(mailService)
	}

	// Configure Slack
	if n.config.Slack != nil && n.config.Slack.Enabled {
		slackService := slack.New(n.config.Slack.Token)
		slackService.AddReceivers(n.config.Slack.Channel)
		n.notifier.UseServices(slackService)
	}

	// Configure Discord
	if n.config.Discord != nil && n.config.Discord.Enabled {
		webhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s",
			n.config.Discord.WebhookID,
			n.config.Discord.Token)
		discordService := discord.New()
		discordService.AddReceivers(webhookURL)
		n.notifier.UseServices(discordService)
	}

	// Configure Telegram
	if n.config.Telegram != nil && n.config.Telegram.Enabled {
		telegramService, err := telegram.New(n.config.Telegram.Token)
		if err != nil {
			return fmt.Errorf("failed to create telegram service: %v", err)
		}
		for _, chatID := range n.config.Telegram.ChatIDs {
			telegramService.AddReceivers(chatID)
		}
		n.notifier.UseServices(telegramService)
	}

	return nil
}

// UpdateConfig updates the notification service configuration
func (n *NotificationService) UpdateConfig(config *NotificationConfig) error {
	n.mu.Lock()
	n.config = config
	n.mu.Unlock()

	return n.configureServices()
}

// ShouldNotify checks if a notification should be sent based on type and severity
func (n *NotificationService) ShouldNotify(notificationType string, severity string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, rule := range n.config.Rules {
		if !rule.Enabled {
			continue
		}

		if rule.Type == notificationType {
			// For scan notifications that don't have severity
			if severity == "" {
				return true
			}

			// Check if severity matches
			for _, s := range rule.Severity {
				if s == severity {
					return true
				}
			}
		}
	}

	return false
}

// Notify sends a notification through configured channels
func (n *NotificationService) Notify(ctx context.Context, subject, message string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Send through standard notifier
	if err := n.notifier.Send(ctx, subject, message); err != nil {
		return fmt.Errorf("failed to send standard notifications: %v", err)
	}

	// Send through Jira if configured
	if n.jira != nil && n.config.Jira != nil && n.config.Jira.Enabled {
		if err := n.jira.CreateIssue(ctx, subject, message, ""); err != nil {
			return fmt.Errorf("failed to create Jira issue: %v", err)
		}
	}

	return nil
}

// NotifyFinding sends a notification about a new finding
func (n *NotificationService) NotifyFinding(ctx context.Context, scanID, scanName, severity, title string) error {
	subject := fmt.Sprintf("New %s Finding in %s", severity, scanName)
	message := fmt.Sprintf("A new %s severity finding has been detected:\nTitle: %s\nScan: %s (ID: %s)",
		severity,
		title,
		scanName,
		scanID,
	)

	return n.Notify(ctx, subject, message)
}

// NotifyScanStarted sends a notification about a scan starting
func (n *NotificationService) NotifyScanStarted(ctx context.Context, scanID, scanName string) error {
	subject := fmt.Sprintf("Scan Started: %s", scanName)
	message := fmt.Sprintf("The scan %s (ID: %s) has started at %s",
		scanName,
		scanID,
		time.Now().Format(time.RFC3339),
	)

	return n.Notify(ctx, subject, message)
}

// NotifyScanFinished sends a notification about a scan completing
func (n *NotificationService) NotifyScanFinished(ctx context.Context, scanID, scanName string) error {
	subject := fmt.Sprintf("Scan Finished: %s", scanName)
	message := fmt.Sprintf("The scan %s (ID: %s) has finished at %s",
		scanName,
		scanID,
		time.Now().Format(time.RFC3339),
	)

	return n.Notify(ctx, subject, message)
}

// NotifyScanFailed sends a notification about a scan failure
func (n *NotificationService) NotifyScanFailed(ctx context.Context, scanID, scanName, error string) error {
	subject := fmt.Sprintf("Scan Failed: %s", scanName)
	message := fmt.Sprintf("The scan %s (ID: %s) has failed at %s\nError: %s",
		scanName,
		scanID,
		time.Now().Format(time.RFC3339),
		error,
	)

	return n.Notify(ctx, subject, message)
}

// NotifyScanStopped sends a notification about a scan being stopped
func (n *NotificationService) NotifyScanStopped(ctx context.Context, scanID, scanName string) error {
	subject := fmt.Sprintf("Scan Stopped: %s", scanName)
	message := fmt.Sprintf("The scan %s (ID: %s) was manually stopped at %s",
		scanName,
		scanID,
		time.Now().Format(time.RFC3339),
	)

	return n.Notify(ctx, subject, message)
}

// GetRules returns the current notification rules
func (n *NotificationService) GetRules() []NotificationRule {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.config == nil {
		log.Printf("Warning: notification config is nil")
		return []NotificationRule{}
	}

	log.Printf("Returning %d notification rules", len(n.config.Rules))
	return n.config.Rules
}

// NotifyWithChannels sends a notification through specified channels
func (n *NotificationService) NotifyWithChannels(ctx context.Context, subject, message string, channels []string, scanID string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	log.Printf("NotifyWithChannels called with subject: %s, channels: %v, scanID: %s", subject, channels, scanID)

	for _, providerID := range channels {
		log.Printf("Processing provider ID: %s", providerID)

		// Get provider details
		provider, err := n.app.Dao().FindRecordById("providers", providerID)
		if err != nil {
			log.Printf("Error finding provider %s: %v", providerID, err)
			continue
		}

		// Check if provider is enabled
		if !provider.GetBool("enabled") {
			log.Printf("Provider %s is disabled", providerID)
			continue
		}

		// Get provider type
		providerType := provider.GetString("provider_type")
		log.Printf("Provider type: %s", providerType)

		// Get provider settings
		settings := provider.Get("settings")
		if settings == nil {
			log.Printf("Provider %s has no settings", providerID)
			continue
		}

		// Convert settings to map
		var settingsMap map[string]interface{}
		settingsStr := fmt.Sprintf("%v", settings) // Convert JsonRaw to string
		if err := json.Unmarshal([]byte(settingsStr), &settingsMap); err != nil {
			log.Printf("Failed to parse settings JSON for provider %s: %v", providerID, err)
			continue
		}

		log.Printf("Parsed settings: %+v", settingsMap)

		// Check if provider is configured for notifications
		usesInterface := provider.Get("use") // Try 'use' first
		if usesInterface == nil {
			// If 'use' is not found, try 'uses'
			usesInterface = provider.Get("uses")
		}
		if usesInterface == nil {
			log.Printf("Provider %s has no use/uses field", providerID)
			continue
		}

		// Check if the provider is configured for notifications
		var isConfiguredForNotifications bool
		switch uses := usesInterface.(type) {
		case string:
			isConfiguredForNotifications = uses == "notification"
		case []string:
			for _, use := range uses {
				if use == "notification" {
					isConfiguredForNotifications = true
					break
				}
			}
		case []interface{}:
			for _, use := range uses {
				if useStr, ok := use.(string); ok && useStr == "notification" {
					isConfiguredForNotifications = true
					break
				}
			}
		default:
			log.Printf("Provider %s use/uses field is of unexpected type: %T", providerID, usesInterface)
			continue
		}

		if !isConfiguredForNotifications {
			log.Printf("Provider %s is not configured for notifications", providerID)
			continue
		}

		// Handle different provider types
		switch providerType {
		case "jira":
			// Get Jira credentials
			username, apiKey, err := n.getJiraCredentials(providerID)
			if err != nil {
				log.Printf("Error getting Jira credentials: %v", err)
				continue
			}

			// Get other required Jira settings
			jiraURL, ok := settingsMap["jira_url"].(string)
			if !ok || jiraURL == "" {
				log.Printf("Invalid or missing Jira URL")
				continue
			}

			projectKey, ok := settingsMap["project_key"].(string)
			if !ok || projectKey == "" {
				log.Printf("Invalid or missing Jira project key")
				continue
			}

			// Get issue type from settings
			issueType, ok := settingsMap["issue_type"].(string)
			if !ok || issueType == "" {
				log.Printf("Invalid or missing issue type, using default Task")
				issueType = "Task"
			}
			log.Printf("Using issue type: %s", issueType)

			// Get client mappings
			var clientMappings []map[string]interface{}
			if mappings, ok := settingsMap["client_mappings"].([]interface{}); ok {
				for _, m := range mappings {
					if mapping, ok := m.(map[string]interface{}); ok {
						clientMappings = append(clientMappings, mapping)
					}
				}
			}
			log.Printf("Found %d client mappings", len(clientMappings))

			// Get scan record to find client ID
			scan, err := n.app.Dao().FindRecordById("nuclei_scans", scanID)
			if err != nil {
				log.Printf("Error finding scan record: %v", err)
				continue
			}

			clientID := scan.GetString("client")
			var organizationID string

			// Find matching client mapping
			for _, mapping := range clientMappings {
				mappingClientID := fmt.Sprintf("%v", mapping["client_id"])
				if mappingClientID == clientID {
					organizationID = fmt.Sprintf("%v", mapping["organization_id"])
					log.Printf("Found matching client mapping: client_id=%s, organization_id=%s", mappingClientID, organizationID)
					break
				}
			}

			if organizationID == "" {
				log.Printf("No organization mapping found for client ID: %s", clientID)
			}

			// Configure Jira service with the provider settings
			jiraConfig := &JiraConfig{
				Enabled:    true,
				URL:        jiraURL,
				Username:   username,
				APIToken:   apiKey,
				ProjectKey: projectKey,
				IssueType:  issueType,
			}

			// Create a new Jira service with these settings
			jiraService := NewJiraService(jiraConfig)

			// Create Jira issue with organization field if found
			if err := jiraService.CreateIssue(ctx, subject, message, organizationID); err != nil {
				log.Printf("Error creating Jira issue: %v", err)
				continue
			}
			log.Printf("Successfully created Jira issue")

		case "email":
			if n.config.Email != nil && n.config.Email.Enabled {
				// Add email notification logic
				log.Printf("Email notifications not implemented yet")
			}
		case "slack":
			if n.config.Slack != nil && n.config.Slack.Enabled {
				// Add Slack notification logic
				log.Printf("Slack notifications not implemented yet")
			}
		case "discord":
			if n.config.Discord != nil && n.config.Discord.Enabled {
				// Add Discord notification logic
				log.Printf("Discord notifications not implemented yet")
			}
		case "telegram":
			if n.config.Telegram != nil && n.config.Telegram.Enabled {
				// Add Telegram notification logic
				log.Printf("Telegram notifications not implemented yet")
			}
		default:
			log.Printf("Unsupported provider type: %s", providerType)
		}
	}

	return nil
}

// getJiraCredentials retrieves the Jira credentials from the provider settings
func (n *NotificationService) getJiraCredentials(providerID string) (string, string, error) {
	provider, err := n.app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return "", "", fmt.Errorf("failed to find provider: %v", err)
	}

	// Use our local jira.GetCredentials helper
	username, apiKey, err := jira.GetCredentials(n.app, provider)
	if err != nil {
		return "", "", fmt.Errorf("failed to get Jira credentials: %v", err)
	}

	return username, apiKey, nil
}
