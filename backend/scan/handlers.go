package scan

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"orbit/models"
	"orbit/services"
	"orbit/services/notification"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	pbModels "github.com/pocketbase/pocketbase/models"
)

var (
	pb                  *pocketbase.PocketBase
	notificationService *notification.NotificationService
	findingManager      *services.FindingManager
	notificationManager *services.NotificationManager
)

// InitHandlers initializes the scan handlers with required services
func InitHandlers(pocketBase *pocketbase.PocketBase, ansibleBasePath string, ns *notification.NotificationService) {
	pb = pocketBase
	notificationService = ns
	findingManager = services.NewFindingManager(pb, notificationService)
	notificationManager = services.NewNotificationManager(pb, notificationService)
}

// HandleScanStarted processes a scan start event
func HandleScanStarted(scanID string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := pb.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := pb.Dao().FindRecordById("clients", scan.GetString("client"))
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

// HandleFinding processes a finding and sends notifications if needed
func HandleFinding(app *pocketbase.PocketBase, finding *models.Finding, notificationService *notification.NotificationService) error {
	// Process the finding (deduplication and rollup)
	findingManager := services.NewFindingManager(app, notificationService)
	_, err := findingManager.ProcessFinding(finding)
	if err != nil {
		return err
	}

	return nil
}

// HandleScanFinished processes a scan completion event
func HandleScanFinished(scanID string) error {
	ctx := context.Background()

	// Get scan details
	scan, err := pb.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := pb.Dao().FindRecordById("clients", scan.GetString("client"))
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
	scan, err := pb.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := pb.Dao().FindRecordById("clients", scan.GetString("client"))
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
	scan, err := pb.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Get client details
	client, err := pb.Dao().FindRecordById("clients", scan.GetString("client"))
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
		collection, err := pb.Dao().FindCollectionByNameOrId("nuclei_scans")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to get nuclei_scans collection: %v", err),
			})
		}

		// Create a test scan record
		testScan := pbModels.NewRecord(collection)
		testScan.Set("name", "Test Scan")
		testScan.Set("tool", "test-tool")
		testScan.Set("tool_version", "1.0.0")
		testScan.Set("status", "running")
		testScan.Set("start_time", time.Now())
		testScan.Set("total_targets", 1)

		// Get the first available client
		clients, err := pb.Dao().FindRecordsByExpr("clients")
		if err != nil || len(clients) == 0 {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "No clients found in the database. Please create a client first.",
			})
		}
		testScan.Set("client", clients[0].Id)

		if err := pb.Dao().SaveRecord(testScan); err != nil {
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
		err = HandleFinding(pb, &models.Finding{
			Name:        req.Finding["name"].(string),
			Description: req.Finding["description"].(string),
			Severity:    req.Finding["severity"].(string),
			Host:        req.Finding["host"].(string),
			Type:        req.Finding["type"].(string),
			Tool:        req.Finding["tool"].(string),
			ScanID:      req.ScanID,
			ClientID:    req.Finding["client"].(string),
		}, notificationService)
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
