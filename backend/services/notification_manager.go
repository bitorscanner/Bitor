package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"orbit/services/notification"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// NotificationData contains data for notifications
type NotificationData struct {
	ScanID       string
	ScanName     string
	ClientID     string
	ClientName   string
	Tool         string
	ToolVersion  string
	StartTime    time.Time
	EndTime      *time.Time
	Error        string
	Findings     map[string]int
	TotalTargets int
}

// NotificationManager handles sending notifications for scan events
type NotificationManager struct {
	app             *pocketbase.PocketBase
	notificationSvc *notification.NotificationService
}

// NewNotificationManager creates a new instance of NotificationManager
func NewNotificationManager(app *pocketbase.PocketBase, notificationSvc *notification.NotificationService) *NotificationManager {
	return &NotificationManager{
		app:             app,
		notificationSvc: notificationSvc,
	}
}

// formatMessage formats a template with the given data
func (n *NotificationManager) formatMessage(templateStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("notification").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return buf.String(), nil
}

// sendNotification sends a notification and records it in the database
func (n *NotificationManager) sendNotification(ctx context.Context, scanID string, event notification.NotificationEvent, message string) error {
	// Get the collection
	collection, err := n.app.Dao().FindCollectionByNameOrId("notification_tracking")
	if err != nil {
		return fmt.Errorf("failed to find notification_tracking collection: %v", err)
	}

	// Create notification record
	record := models.NewRecord(collection)
	record.Set("scan_id", scanID)
	record.Set("event_type", string(event))
	record.Set("message", message)
	record.Set("status", "sent")
	record.Set("sent_at", time.Now())

	if err := n.app.Dao().SaveRecord(record); err != nil {
		log.Printf("Failed to save notification record: %v", err)
		return fmt.Errorf("failed to save notification record: %v", err)
	}

	log.Printf("Recorded notification for scan %s, event %s", scanID, event)
	return nil
}

// NotifyScanStarted sends a notification when a scan starts
func (n *NotificationManager) NotifyScanStarted(ctx context.Context, scanID string, data map[string]interface{}) error {
	message, err := n.formatMessage(ScanStartedTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to format scan started message: %v", err)
	}

	log.Printf("Attempting to send scan started notification with message: %s", message)

	// Get notification rules from config
	rules := n.notificationSvc.GetRules()
	log.Printf("Found %d notification rules", len(rules))

	// Find rules that match this event
	var channels []string
	for _, rule := range rules {
		if rule.Type == "scan_started" && rule.Enabled {
			log.Printf("Found matching rule for scan_started event: %+v", rule)
			channels = append(channels, rule.Channels...)
		}
	}

	log.Printf("Sending notification to channels: %v", channels)
	if err := n.notificationSvc.NotifyWithChannels(ctx, "Scan Started", message, channels, scanID); err != nil {
		return fmt.Errorf("failed to send scan started notification: %v", err)
	}

	return n.sendNotification(ctx, scanID, notification.ScanStarted, message)
}

// NotifyScanFinished sends a notification when a scan finishes
func (n *NotificationManager) NotifyScanFinished(ctx context.Context, scanID string, data map[string]interface{}) error {
	message, err := n.formatMessage(ScanFinishedTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to format scan finished message: %v", err)
	}

	// Get notification rules from config
	rules := n.notificationSvc.GetRules()
	log.Printf("Found %d notification rules", len(rules))

	// Find rules that match this event
	var channels []string
	for _, rule := range rules {
		if rule.Type == "scan_finished" && rule.Enabled {
			log.Printf("Found matching rule for scan_finished event: %+v", rule)
			channels = append(channels, rule.Channels...)
		}
	}

	if len(channels) == 0 {
		log.Printf("No enabled rules found for scan_finished event, skipping notifications")
		return nil
	}

	log.Printf("Sending notification to channels: %v", channels)
	if err := n.notificationSvc.NotifyWithChannels(ctx, "Scan Finished", message, channels, scanID); err != nil {
		return fmt.Errorf("failed to send scan finished notification: %v", err)
	}

	return n.sendNotification(ctx, scanID, notification.ScanFinished, message)
}

// NotifyScanFailed sends a notification when a scan fails
func (n *NotificationManager) NotifyScanFailed(ctx context.Context, scanID string, data map[string]interface{}) error {
	message, err := n.formatMessage(ScanFailedTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to format scan failed message: %v", err)
	}

	if err := n.notificationSvc.NotifyWithChannels(ctx, "Scan Failed", message, []string{"email", "slack", "discord", "telegram"}, scanID); err != nil {
		return fmt.Errorf("failed to send scan failed notification: %v", err)
	}

	return n.sendNotification(ctx, scanID, notification.ScanFailed, message)
}

// NotifyScanStopped sends a notification when a scan is stopped
func (n *NotificationManager) NotifyScanStopped(ctx context.Context, scanID string, data map[string]interface{}) error {
	message, err := n.formatMessage(ScanStoppedTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to format scan stopped message: %v", err)
	}

	if err := n.notificationSvc.NotifyWithChannels(ctx, "Scan Stopped", message, []string{"email", "slack", "discord", "telegram"}, scanID); err != nil {
		return fmt.Errorf("failed to send scan stopped notification: %v", err)
	}

	return n.sendNotification(ctx, scanID, notification.ScanStopped, message)
}

// NotifyFinding sends a notification for a new finding
func (n *NotificationManager) NotifyFinding(ctx context.Context, scanID string, data map[string]interface{}) error {
	message, err := n.formatMessage(FindingTemplate, data)
	if err != nil {
		return fmt.Errorf("failed to format finding message: %v", err)
	}

	if err := n.notificationSvc.NotifyWithChannels(ctx, fmt.Sprintf("New %s Finding", data["severity"]), message, []string{"email", "slack", "discord", "telegram"}, scanID); err != nil {
		return fmt.Errorf("failed to send finding notification: %v", err)
	}

	return n.sendNotification(ctx, scanID, notification.Finding, message)
}

// HandleScanEvent processes scan-related notification events
func (n *NotificationManager) HandleScanEvent(ctx context.Context, event notification.NotificationEvent, data NotificationData) error {
	log.Printf("Handling scan event: %s for scan ID: %s", event, data.ScanID)

	// Get notification rules
	rules := n.notificationSvc.GetRules()
	log.Printf("Found %d notification rules", len(rules))

	// Find matching rules for this event type
	var channels []string
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if rule.Type == string(event) {
			log.Printf("Found matching rule: %+v", rule)
			channels = append(channels, rule.Channels...)
		}
	}

	// Remove duplicate channels
	channelsMap := make(map[string]bool)
	var uniqueChannels []string
	for _, ch := range channels {
		if !channelsMap[ch] {
			channelsMap[ch] = true
			uniqueChannels = append(uniqueChannels, ch)
		}
	}

	log.Printf("Using channels: %v", uniqueChannels)

	// Convert NotificationData to map for template
	templateData := map[string]interface{}{
		"scan_id":       data.ScanID,
		"scan_name":     data.ScanName,
		"client_name":   data.ClientName,
		"tool":          data.Tool,
		"tool_version":  data.ToolVersion,
		"time":          time.Now().Format(time.RFC3339),
		"start_time":    data.StartTime.Format(time.RFC3339),
		"total_targets": data.TotalTargets,
	}

	if data.EndTime != nil {
		templateData["end_time"] = data.EndTime.Format(time.RFC3339)
	}

	if data.Error != "" {
		templateData["error"] = data.Error
	}

	if data.Findings != nil {
		templateData["critical_findings"] = data.Findings["critical"]
		templateData["high_findings"] = data.Findings["high"]
		templateData["medium_findings"] = data.Findings["medium"]
		templateData["low_findings"] = data.Findings["low"]
		templateData["info_findings"] = data.Findings["info"]
	}

	var err error
	switch event {
	case notification.ScanStarted:
		err = n.NotifyScanStarted(ctx, data.ScanID, templateData)
	case notification.ScanFinished:
		err = n.NotifyScanFinished(ctx, data.ScanID, templateData)
	case notification.ScanFailed:
		err = n.NotifyScanFailed(ctx, data.ScanID, templateData)
	case notification.ScanStopped:
		err = n.NotifyScanStopped(ctx, data.ScanID, templateData)
	default:
		return fmt.Errorf("unknown event type: %s", event)
	}

	if err != nil {
		return fmt.Errorf("failed to handle event %s: %v", event, err)
	}

	return nil
}
