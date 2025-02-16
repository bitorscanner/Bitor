package scan

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"orbit/services"
	"orbit/services/notification"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

var (
	app                 *pocketbase.PocketBase
	notificationService *notification.NotificationService
	findingManager      *services.FindingManager
	notificationManager *services.NotificationManager
)

// InitHandlers initializes the scan handlers with required services
func InitHandlers(pb *pocketbase.PocketBase, ansibleBasePath string, ns *notification.NotificationService) {
	app = pb
	notificationService = ns
	findingManager = services.NewFindingManager(app, notificationService)
	notificationManager = services.NewNotificationManager(app, notificationService)
}

// HandleScanStarted processes a scan start event
func HandleScanStarted(scanID string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := app.Dao().FindRecordById("clients", scan.GetString("client"))
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	// Prepare notification data
	data := services.NotificationData{
		ScanID:       scanID,
		ScanName:     scan.GetString("name"),
		ClientID:     client.Id,
		ClientName:   client.GetString("name"),
		Tool:         scan.GetString("tool"),
		ToolVersion:  scan.GetString("tool_version"),
		StartTime:    time.Now(),
		TotalTargets: scan.GetInt("total_targets"),
	}

	// Send notification
	if err := notificationManager.HandleScanEvent(ctx, notification.ScanStarted, data); err != nil {
		log.Printf("Failed to send scan started notification: %v", err)
	}

	return nil
}

// HandleFinding processes a new finding
func HandleFinding(scanID string, finding map[string]interface{}) error {
	ctx := context.Background()

	// Get scan details
	scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Create Finding object
	f := services.Finding{
		Title:       finding["title"].(string),
		Description: finding["description"].(string),
		Severity:    finding["severity"].(string),
		Target:      finding["target"].(string),
		Type:        finding["type"].(string),
		Tool:        scan.GetString("tool"),
		ScanID:      scanID,
		ClientID:    scan.GetString("client"),
		Raw:         finding,
	}

	// Process finding
	if err := findingManager.HandleFinding(ctx, f); err != nil {
		return fmt.Errorf("failed to handle finding: %v", err)
	}

	return nil
}

// HandleScanFinished processes a scan completion event
func HandleScanFinished(scanID string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := app.Dao().FindRecordById("clients", scan.GetString("client"))
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	// Get finding summary
	summary, err := findingManager.GetRollupSummary(scanID)
	if err != nil {
		return fmt.Errorf("failed to get finding summary: %v", err)
	}

	// Prepare notification data
	endTime := time.Now()
	data := services.NotificationData{
		ScanID:       scanID,
		ScanName:     scan.GetString("name"),
		ClientID:     client.Id,
		ClientName:   client.GetString("name"),
		Tool:         scan.GetString("tool"),
		ToolVersion:  scan.GetString("tool_version"),
		StartTime:    scan.GetDateTime("start_time").Time(),
		EndTime:      &endTime,
		Findings:     summary,
		TotalTargets: scan.GetInt("total_targets"),
	}

	// Send notification
	if err := notificationManager.HandleScanEvent(ctx, notification.ScanFinished, data); err != nil {
		log.Printf("Failed to send scan finished notification: %v", err)
	}

	// Mark rollup notification as sent
	if err := findingManager.MarkRollupNotificationSent(scanID); err != nil {
		log.Printf("Failed to mark rollup notification as sent: %v", err)
	}

	return nil
}

// HandleScanFailed processes a scan failure event
func HandleScanFailed(scanID string, errorMsg string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := app.Dao().FindRecordById("clients", scan.GetString("client"))
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	// Prepare notification data
	data := services.NotificationData{
		ScanID:     scanID,
		ScanName:   scan.GetString("name"),
		ClientID:   client.Id,
		ClientName: client.GetString("name"),
		Tool:       scan.GetString("tool"),
		Error:      errorMsg,
		StartTime:  scan.GetDateTime("start_time").Time(),
	}

	// Send notification
	if err := notificationManager.HandleScanEvent(ctx, notification.ScanFailed, data); err != nil {
		log.Printf("Failed to send scan failed notification: %v", err)
	}

	return nil
}

// HandleScanStopped processes a scan stop event
func HandleScanStopped(scanID string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := app.Dao().FindRecordById("clients", scan.GetString("client"))
	if err != nil {
		return fmt.Errorf("failed to get client: %v", err)
	}

	// Prepare notification data
	data := services.NotificationData{
		ScanID:     scanID,
		ScanName:   scan.GetString("name"),
		ClientID:   client.Id,
		ClientName: client.GetString("name"),
		Tool:       scan.GetString("tool"),
		StartTime:  scan.GetDateTime("start_time").Time(),
	}

	// Send notification
	if err := notificationManager.HandleScanEvent(ctx, notification.ScanStopped, data); err != nil {
		log.Printf("Failed to send scan stopped notification: %v", err)
	}

	return nil
}

// TestNotificationEvent represents the request body for testing notifications
type TestNotificationEvent struct {
	ScanID  string                 `json:"scan_id"`
	Event   string                 `json:"event"`
	Finding map[string]interface{} `json:"finding,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// HandleTestNotification allows testing different notification events
func HandleTestNotification(c echo.Context) error {
	var req TestNotificationEvent
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// If no scan ID is provided, create a temporary test scan
	if req.ScanID == "" {
		collection, err := app.Dao().FindCollectionByNameOrId("nuclei_scans")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to get nuclei_scans collection: %v", err),
			})
		}

		// Create a test scan record
		testScan := models.NewRecord(collection)
		testScan.Set("name", "Test Scan")
		testScan.Set("tool", "test-tool")
		testScan.Set("tool_version", "1.0.0")
		testScan.Set("status", "running")
		testScan.Set("start_time", time.Now())
		testScan.Set("total_targets", 1)

		// Get the first available client
		clients, err := app.Dao().FindRecordsByExpr("clients")
		if err != nil || len(clients) == 0 {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "No clients found in the database. Please create a client first.",
			})
		}
		testScan.Set("client", clients[0].Id)

		if err := app.Dao().SaveRecord(testScan); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to create test scan: %v", err),
			})
		}
		req.ScanID = testScan.Id
	}

	var err error
	switch req.Event {
	case "scan_started":
		err = HandleScanStarted(req.ScanID)
	case "scan_finished":
		err = HandleScanFinished(req.ScanID)
	case "scan_failed":
		err = HandleScanFailed(req.ScanID, req.Error)
	case "scan_stopped":
		err = HandleScanStopped(req.ScanID)
	case "finding":
		if req.Finding == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Finding data is required for finding event",
			})
		}
		err = HandleFinding(req.ScanID, req.Finding)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid event type",
		})
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to handle event: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Successfully processed %s event", req.Event),
		"scan_id": req.ScanID,
	})
}
