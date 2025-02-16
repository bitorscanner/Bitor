package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"orbit/services/notification"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// FindingManager handles finding deduplication and rollups
type FindingManager struct {
	app                 *pocketbase.PocketBase
	notificationService *notification.NotificationService
}

// NewFindingManager creates a new instance of FindingManager
func NewFindingManager(app *pocketbase.PocketBase, notificationService *notification.NotificationService) *FindingManager {
	return &FindingManager{
		app:                 app,
		notificationService: notificationService,
	}
}

// Finding represents a security finding
type Finding struct {
	Title       string
	Description string
	Severity    string
	Target      string
	Type        string
	Tool        string
	ScanID      string
	ClientID    string
	Raw         interface{} // Raw finding data for hash calculation
}

// ProcessFinding processes a new finding, checking for duplicates and updating rollups
func (fm *FindingManager) ProcessFinding(finding Finding) error {
	// Generate hash for the finding
	hash, err := fm.generateFindingHash(finding)
	if err != nil {
		return fmt.Errorf("failed to generate finding hash: %v", err)
	}

	// Check if this is a duplicate
	isDuplicate, err := fm.checkDuplicate(hash, finding)
	if err != nil {
		return fmt.Errorf("failed to check duplicate: %v", err)
	}

	// Update finding rollup
	if err := fm.updateRollup(finding, isDuplicate); err != nil {
		return fmt.Errorf("failed to update rollup: %v", err)
	}

	return nil
}

// generateFindingHash creates a unique hash for a finding
func (fm *FindingManager) generateFindingHash(finding Finding) (string, error) {
	// Create a map of the important fields for hashing
	hashData := map[string]interface{}{
		"title":       finding.Title,
		"description": finding.Description,
		"type":        finding.Type,
		"target":      finding.Target,
		"raw":         finding.Raw,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(hashData)
	if err != nil {
		return "", err
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// checkDuplicate checks if a finding is a duplicate and updates the hash record
func (fm *FindingManager) checkDuplicate(hash string, finding Finding) (bool, error) {
	// Check for existing hash
	record, err := fm.app.Dao().FindFirstRecordByFilter(
		"finding_hashes",
		"hash = {:hash}",
		dbx.Params{
			"hash": hash,
		},
	)

	if err != nil {
		return false, err
	}

	if record != nil {
		// Update last seen time
		record.Set("last_seen", time.Now())
		record.Set("scan_id", finding.ScanID)
		if err := fm.app.Dao().SaveRecord(record); err != nil {
			return false, err
		}
		return true, nil
	}

	// Create new hash record
	collection, err := fm.app.Dao().FindCollectionByNameOrId("finding_hashes")
	if err != nil {
		return false, err
	}

	newRecord := models.NewRecord(collection)
	newRecord.Set("hash", hash)
	newRecord.Set("target", finding.Target)
	newRecord.Set("finding_type", finding.Type)
	newRecord.Set("severity", finding.Severity)
	newRecord.Set("last_seen", time.Now())
	newRecord.Set("scan_id", finding.ScanID)
	newRecord.Set("client_id", finding.ClientID)

	if err := fm.app.Dao().SaveRecord(newRecord); err != nil {
		return false, err
	}

	return false, nil
}

// updateRollup updates the finding rollup for a scan
func (fm *FindingManager) updateRollup(finding Finding, isDuplicate bool) error {
	// Get or create rollup record
	record, err := fm.app.Dao().FindFirstRecordByFilter(
		"finding_rollups",
		"scan_id = {:scan_id}",
		dbx.Params{
			"scan_id": finding.ScanID,
		},
	)

	if err != nil {
		return err
	}

	if record == nil {
		// Create new rollup record
		collection, err := fm.app.Dao().FindCollectionByNameOrId("finding_rollups")
		if err != nil {
			return err
		}

		record = models.NewRecord(collection)
		record.Set("scan_id", finding.ScanID)
		record.Set("critical_count", 0)
		record.Set("high_count", 0)
		record.Set("medium_count", 0)
		record.Set("low_count", 0)
		record.Set("info_count", 0)
		record.Set("new_findings_count", 0)
		record.Set("duplicate_findings_count", 0)
		record.Set("notification_sent", false)
	}

	// Update counts
	if isDuplicate {
		record.Set("duplicate_findings_count", record.GetInt("duplicate_findings_count")+1)
	} else {
		record.Set("new_findings_count", record.GetInt("new_findings_count")+1)
		// Update severity count
		severityField := fmt.Sprintf("%s_count", finding.Severity)
		record.Set(severityField, record.GetInt(severityField)+1)
	}

	return fm.app.Dao().SaveRecord(record)
}

// GetRollupSummary gets the finding summary for a scan
func (fm *FindingManager) GetRollupSummary(scanID string) (map[string]int, error) {
	record, err := fm.app.Dao().FindFirstRecordByFilter(
		"finding_rollups",
		"scan_id = {:scan_id}",
		dbx.Params{
			"scan_id": scanID,
		},
	)

	if err != nil {
		// If no record found, return default values instead of error
		if err.Error() == "sql: no rows in result set" {
			return map[string]int{
				"critical":  0,
				"high":      0,
				"medium":    0,
				"low":       0,
				"info":      0,
				"new":       0,
				"duplicate": 0,
			}, nil
		}
		return nil, err
	}

	if record == nil {
		return map[string]int{
			"critical":  0,
			"high":      0,
			"medium":    0,
			"low":       0,
			"info":      0,
			"new":       0,
			"duplicate": 0,
		}, nil
	}

	return map[string]int{
		"critical":  record.GetInt("critical_count"),
		"high":      record.GetInt("high_count"),
		"medium":    record.GetInt("medium_count"),
		"low":       record.GetInt("low_count"),
		"info":      record.GetInt("info_count"),
		"new":       record.GetInt("new_findings_count"),
		"duplicate": record.GetInt("duplicate_findings_count"),
	}, nil
}

// MarkRollupNotificationSent marks that a notification has been sent for this rollup
func (fm *FindingManager) MarkRollupNotificationSent(scanID string) error {
	record, err := fm.app.Dao().FindFirstRecordByFilter(
		"finding_rollups",
		"scan_id = {:scan_id}",
		dbx.Params{
			"scan_id": scanID,
		},
	)

	if err != nil {
		// If no record found, create a new one with default values
		if err.Error() == "sql: no rows in result set" {
			collection, err := fm.app.Dao().FindCollectionByNameOrId("finding_rollups")
			if err != nil {
				return fmt.Errorf("failed to find finding_rollups collection: %v", err)
			}

			record = models.NewRecord(collection)
			record.Set("scan_id", scanID)
			record.Set("critical_count", 0)
			record.Set("high_count", 0)
			record.Set("medium_count", 0)
			record.Set("low_count", 0)
			record.Set("info_count", 0)
			record.Set("new_findings_count", 0)
			record.Set("duplicate_findings_count", 0)
			record.Set("notification_sent", true)
			record.Set("last_notification_time", time.Now())

			return fm.app.Dao().SaveRecord(record)
		}
		return err
	}

	if record == nil {
		// Create new record with default values
		collection, err := fm.app.Dao().FindCollectionByNameOrId("finding_rollups")
		if err != nil {
			return fmt.Errorf("failed to find finding_rollups collection: %v", err)
		}

		record = models.NewRecord(collection)
		record.Set("scan_id", scanID)
		record.Set("critical_count", 0)
		record.Set("high_count", 0)
		record.Set("medium_count", 0)
		record.Set("low_count", 0)
		record.Set("info_count", 0)
		record.Set("new_findings_count", 0)
		record.Set("duplicate_findings_count", 0)
		record.Set("notification_sent", true)
		record.Set("last_notification_time", time.Now())

		return fm.app.Dao().SaveRecord(record)
	}

	record.Set("notification_sent", true)
	record.Set("last_notification_time", time.Now())

	return fm.app.Dao().SaveRecord(record)
}

// HandleFinding processes a finding and sends notifications if needed
func (fm *FindingManager) HandleFinding(ctx context.Context, finding Finding) error {
	// Process the finding (deduplication and rollup)
	if err := fm.ProcessFinding(finding); err != nil {
		return fmt.Errorf("failed to process finding: %v", err)
	}

	// Get scan details
	scan, err := fm.app.Dao().FindRecordById("scans", finding.ScanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Check if we should notify based on severity
	if fm.notificationService.ShouldNotify(finding.Severity+"_finding", finding.Severity) {
		// Send notification
		if err := fm.notificationService.NotifyFinding(
			ctx,
			finding.ScanID,
			scan.GetString("name"),
			finding.Severity,
			finding.Title,
		); err != nil {
			return fmt.Errorf("failed to send finding notification: %v", err)
		}
	}

	return nil
}
