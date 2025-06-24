package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"bitor/models"
	"bitor/services/notification"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	pbModels "github.com/pocketbase/pocketbase/models"
)

// FindingManager handles finding deduplication and rollups
type FindingManager struct {
	app                 *pocketbase.PocketBase
	notificationService *notification.NotificationService
	logger              *log.Logger
}

// ScanDuplicateTracker tracks duplicates for each scan
type ScanDuplicateTracker struct {
	TotalNew  int
	TotalDups int
}

var scanTrackers = make(map[string]*ScanDuplicateTracker)

// NewFindingManager creates a new instance of FindingManager
func NewFindingManager(app *pocketbase.PocketBase, notificationService *notification.NotificationService) *FindingManager {
	return &FindingManager{
		app:                 app,
		notificationService: notificationService,
		logger:              log.New(log.Writer(), "[FindingManager] ", log.LstdFlags),
	}
}

// ProcessFinding processes a new finding, checking for duplicates and updating rollups
func (fm *FindingManager) ProcessFinding(finding *models.Finding) (bool, error) {
	// Generate hash for the finding
	hash := finding.GenerateHash()

	fm.logger.Printf("Processing finding - Name: %s, Hash: %s, Client: %s", finding.Name, hash, finding.ClientID)
	fm.logger.Printf("Finding details - Severity: %s, Host: %s, Type: %s, TemplateID: %s",
		finding.Severity, finding.Host, finding.Type, finding.TemplateID)

	// Check if finding exists in nuclei_findings
	existingResult, err := fm.app.Dao().FindFirstRecordByFilter(
		"nuclei_findings",
		"(hash = {:hash} && client = {:client} && hash != '' && hash != 'null' && hash != null)",
		dbx.Params{
			"hash":   hash,
			"client": finding.ClientID,
		},
	)

	// Only treat as error if it's not a "no rows" error
	if err != nil && err.Error() != "sql: no rows in result set" {
		return false, fmt.Errorf("failed to query nuclei_findings: %v", err)
	}

	// If finding exists, update it and return
	if existingResult != nil {
		fm.logger.Printf("Found duplicate finding with hash %s", hash)

		// Get existing scan_ids
		var scanIDs []string
		scanIDsStr := existingResult.GetString("scan_ids")
		if scanIDsStr != "" {
			if err := json.Unmarshal([]byte(scanIDsStr), &scanIDs); err != nil {
				fm.logger.Printf("Error unmarshaling scan_ids: %v", err)
				scanIDs = []string{}
			}
		}

		// Add new scan_id if it doesn't exist
		scanIDExists := false
		for _, id := range scanIDs {
			if id == finding.ScanID {
				scanIDExists = true
				break
			}
		}

		if !scanIDExists {
			scanIDs = append(scanIDs, finding.ScanID)
			scanIDsJSON, err := json.Marshal(scanIDs)
			if err != nil {
				fm.logger.Printf("Error marshaling scan_ids: %v", err)
			} else {
				existingResult.Set("scan_ids", string(scanIDsJSON))
				if err := fm.app.Dao().SaveRecord(existingResult); err != nil {
					fm.logger.Printf("Error updating finding record: %v", err)
				}
			}
		}

		if err := fm.updateRollup(finding, true); err != nil {
			fm.logger.Printf("Error updating rollup for duplicate finding: %v", err)
		}

		return true, nil
	}

	// Create new record
	resultsCollection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings")
	if err != nil {
		return false, fmt.Errorf("failed to find nuclei_findings collection: %v", err)
	}

	resultRecord := pbModels.NewRecord(resultsCollection)

	// Convert finding to map and set all fields
	data := finding.ToMap()
	for key, value := range data {
		resultRecord.Set(key, value)
	}

	// Set created_by field if available
	if finding.CreatedBy != "" {
		resultRecord.Set("created_by", finding.CreatedBy)
	}

	// Initialize scan_ids array with the current scan
	scanIDsJSON, err := json.Marshal([]string{finding.ScanID})
	if err == nil {
		resultRecord.Set("scan_ids", string(scanIDsJSON))
	}

	// Save the record
	if err := fm.app.Dao().SaveRecord(resultRecord); err != nil {
		fm.logger.Printf("[ERROR] Failed to save nuclei_findings record: %v", err)
		return false, fmt.Errorf("failed to save nuclei_findings record: %v", err)
	}

	fm.logger.Printf("[DEBUG] Saved nuclei_findings record with ID %s and hash %s", resultRecord.Id, hash)

	// Update rollup for new finding
	if err := fm.updateRollup(finding, false); err != nil {
		fm.logger.Printf("Error updating rollup for new finding: %v", err)
	}

	return false, nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// generateFindingHash creates a unique hash for a finding
func (fm *FindingManager) generateFindingHash(finding *models.Finding) (string, error) {
	return finding.GenerateHash(), nil
}

// updateRollup updates the finding rollup for a scan
func (fm *FindingManager) updateRollup(finding *models.Finding, isDuplicate bool) error {
	fm.logger.Printf("Updating rollup for finding - Name: %s, IsDuplicate: %v", finding.Name, isDuplicate)

	// Get or create tracker for this scan
	tracker, exists := scanTrackers[finding.ScanID]
	if !exists {
		tracker = &ScanDuplicateTracker{
			TotalNew:  0,
			TotalDups: 0,
		}
		scanTrackers[finding.ScanID] = tracker
	}

	// Get existing rollup or create new one
	rollup, err := fm.app.Dao().FindFirstRecordByFilter(
		"nuclei_findings_rollups",
		"scan_id = {:scan_id}",
		dbx.Params{
			"scan_id": finding.ScanID,
		},
	)

	// Log current state if rollup exists
	if err == nil && rollup != nil {
		fm.logger.Printf("Current rollup state for scan %s:", finding.ScanID)
		fm.logger.Printf("- New findings: %d", tracker.TotalNew)
		fm.logger.Printf("- Duplicate findings: %d", tracker.TotalDups)
		fm.logger.Printf("- Critical: %d, High: %d, Medium: %d, Low: %d, Info: %d",
			rollup.GetInt("critical_count"),
			rollup.GetInt("high_count"),
			rollup.GetInt("medium_count"),
			rollup.GetInt("low_count"),
			rollup.GetInt("info_count"))
	}

	// If no rollup exists, create a new one
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			fm.logger.Printf("No existing rollup found for scan %s - creating new rollup", finding.ScanID)
			collection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings_rollups")
			if err != nil {
				return fmt.Errorf("failed to get nuclei_findings_rollups collection: %v", err)
			}

			rollup = pbModels.NewRecord(collection)
			rollup.Set("scan_id", finding.ScanID)
			rollup.Set("critical_count", 0)
			rollup.Set("high_count", 0)
			rollup.Set("medium_count", 0)
			rollup.Set("low_count", 0)
			rollup.Set("info_count", 0)
			rollup.Set("new_findings_count", 0)
			rollup.Set("duplicate_findings_count", 0)
			rollup.Set("notification_sent", false)
		} else {
			return fmt.Errorf("failed to get rollup: %v", err)
		}
	}

	if rollup == nil {
		// Create new record with default values
		collection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings_rollups")
		if err != nil {
			return fmt.Errorf("failed to find nuclei_findings_rollups collection: %v", err)
		}

		rollup = pbModels.NewRecord(collection)
		rollup.Set("scan_id", finding.ScanID)
		rollup.Set("critical_count", 0)
		rollup.Set("high_count", 0)
		rollup.Set("medium_count", 0)
		rollup.Set("low_count", 0)
		rollup.Set("info_count", 0)
		rollup.Set("new_findings_count", 0)
		rollup.Set("duplicate_findings_count", 0)
		rollup.Set("notification_sent", false)
	}

	if isDuplicate {
		// Simply increment the duplicate count for every duplicate finding
		tracker.TotalDups++
		rollup.Set("duplicate_findings_count", tracker.TotalDups)
		fm.logger.Printf("Incrementing duplicate findings count to %d", tracker.TotalDups)
	} else {
		tracker.TotalNew++
		rollup.Set("new_findings_count", tracker.TotalNew)
		fm.logger.Printf("Incrementing new findings count to %d", tracker.TotalNew)

		severity := strings.ToLower(finding.Severity)
		currentSeverityCount := 0
		severityField := ""

		switch severity {
		case "critical":
			severityField = "critical_count"
			currentSeverityCount = rollup.GetInt("critical_count")
		case "high":
			severityField = "high_count"
			currentSeverityCount = rollup.GetInt("high_count")
		case "medium":
			severityField = "medium_count"
			currentSeverityCount = rollup.GetInt("medium_count")
		case "low":
			severityField = "low_count"
			currentSeverityCount = rollup.GetInt("low_count")
		case "info":
			severityField = "info_count"
			currentSeverityCount = rollup.GetInt("info_count")
		default:
			fm.logger.Printf("Warning: Unknown severity level: %s", severity)
		}

		if severityField != "" {
			rollup.Set(severityField, currentSeverityCount+1)
			fm.logger.Printf("Incrementing %s to %d", severityField, currentSeverityCount+1)
		}
	}

	// Save the rollup
	if err := fm.app.Dao().SaveRecord(rollup); err != nil {
		fm.logger.Printf("Error saving rollup: %v", err)
		return fmt.Errorf("failed to save rollup: %v", err)
	}

	// Log final state
	fm.logger.Printf("Final rollup state after update:")
	fm.logger.Printf("- New findings: %d", tracker.TotalNew)
	fm.logger.Printf("- Duplicate findings: %d", tracker.TotalDups)
	fm.logger.Printf("- Critical: %d, High: %d, Medium: %d, Low: %d, Info: %d",
		rollup.GetInt("critical_count"),
		rollup.GetInt("high_count"),
		rollup.GetInt("medium_count"),
		rollup.GetInt("low_count"),
		rollup.GetInt("info_count"))

	return nil
}

// CleanupScanTracker removes the tracker for a completed scan
func (fm *FindingManager) CleanupScanTracker(scanID string) {
	delete(scanTrackers, scanID)
	fm.logger.Printf("Cleaned up tracker for scan %s", scanID)
}

// GetRollupSummary gets the finding summary for a scan
func (fm *FindingManager) GetRollupSummary(scanID string) (map[string]int, error) {
	record, err := fm.app.Dao().FindFirstRecordByFilter(
		"nuclei_findings_rollups",
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
		"nuclei_findings_rollups",
		"scan_id = {:scan_id}",
		dbx.Params{
			"scan_id": scanID,
		},
	)

	if err != nil {
		// If no record found, create a new one with default values
		if err.Error() == "sql: no rows in result set" {
			collection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings_rollups")
			if err != nil {
				return fmt.Errorf("failed to find nuclei_findings_rollups collection: %v", err)
			}

			record = pbModels.NewRecord(collection)
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
		collection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings_rollups")
		if err != nil {
			return fmt.Errorf("failed to find nuclei_findings_rollups collection: %v", err)
		}

		record = pbModels.NewRecord(collection)
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
func (fm *FindingManager) HandleFinding(ctx context.Context, finding *models.Finding) error {
	// Process the finding (deduplication and rollup)
	isDuplicate, err := fm.ProcessFinding(finding)
	if err != nil {
		return fmt.Errorf("failed to process finding: %v", err)
	}

	// If it's a duplicate, we don't need to send a notification
	if isDuplicate {
		return nil
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
			finding.Name,
		); err != nil {
			return fmt.Errorf("failed to send finding notification: %v", err)
		}
	}

	return nil
}

// FinalizeScan cleans up the scan tracker and performs any final processing
func (fm *FindingManager) FinalizeScan(scanID string) {
	fm.CleanupScanTracker(scanID)
}

// DeleteClientFindings deletes all findings for a specific client
func (fm *FindingManager) DeleteClientFindings(clientID string) error {
	// Delete from nuclei_findings
	records, err := fm.app.Dao().FindRecordsByFilter(
		"nuclei_findings",
		"client = {:client}",
		"id",
		0,
		-1,
		dbx.Params{
			"client": clientID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get findings for client: %v", err)
	}

	// Delete each finding
	for _, record := range records {
		if err := fm.app.Dao().DeleteRecord(record); err != nil {
			return fmt.Errorf("failed to delete finding %s: %v", record.Id, err)
		}
	}

	return nil
}

// DeleteOrphanedFindings deletes all findings that have no valid client
func (fm *FindingManager) DeleteOrphanedFindings() error {
	// Delete from nuclei_findings
	records, err := fm.app.Dao().FindRecordsByFilter(
		"nuclei_findings",
		"client = 'N/A' || client = ''",
		"id",
		0,
		-1,
		dbx.Params{},
	)
	if err != nil {
		return fmt.Errorf("failed to get orphaned findings: %v", err)
	}

	// Delete each finding
	for _, record := range records {
		if err := fm.app.Dao().DeleteRecord(record); err != nil {
			return fmt.Errorf("failed to delete finding %s: %v", record.Id, err)
		}
	}

	return nil
}

// Close is no longer needed since we're not managing any resources that need closing
func (fm *FindingManager) Close() {
	// No-op as we're not managing any resources that need closing
}

// GetTotalFindingsCount returns the total number of findings that need to be processed
func (fm *FindingManager) GetTotalFindingsCount() (int, error) {
	// Find the nuclei_findings collection
	resultsCollection, err := fm.app.Dao().FindCollectionByNameOrId("nuclei_findings")
	if err != nil {
		fm.logger.Printf("Error finding nuclei_findings collection: %v", err)
		return 0, fmt.Errorf("failed to find nuclei_findings collection: %v", err)
	}

	// Count findings that need processing (no hash yet)
	records, err := fm.app.Dao().FindRecordsByFilter(
		resultsCollection.Id,
		"(hash = '' || hash = null || hash = 'null')",
		"",
		0,
		-1,
		nil,
	)
	if err != nil {
		fm.logger.Printf("Error counting findings: %v", err)
		return 0, fmt.Errorf("failed to count findings: %v", err)
	}

	count := len(records)
	fm.logger.Printf("Found %d findings that need processing", count)
	return count, nil
}

// ProcessFindingsBatch processes a batch of findings for migration
func (fm *FindingManager) ProcessFindingsBatch(offset string, limit string, migrationOnly bool) (int, error) {
	// Convert string parameters to integers
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		return 0, fmt.Errorf("invalid offset: %v", err)
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return 0, fmt.Errorf("invalid limit: %v", err)
	}

	processedCount := 0
	// Start a transaction
	err = fm.app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
		// Get records that need processing
		filter := "(hash = '' OR hash = null OR hash = 'null')"
		if !migrationOnly {
			filter = ""
		}

		records, err := txDao.FindRecordsByFilter(
			"nuclei_findings",
			filter,
			"created",
			offsetInt,
			limitInt,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to get records: %v", err)
		}

		if len(records) == 0 {
			fm.logger.Printf("[INFO] No records to process")
			return nil
		}

		fm.logger.Printf("[INFO] Processing %d records", len(records))

		for _, record := range records {
			// Skip if record already has a valid hash
			currentHash := record.GetString("hash")
			if currentHash != "" && currentHash != "null" {
				fm.logger.Printf("[DEBUG] Skipping record %s as it already has hash %s", record.Id, currentHash)
				continue
			}

			// Create Finding struct from record
			finding := &models.Finding{
				ClientID:    record.GetString("client"),
				ScanID:      record.GetString("scan_id"),
				Name:        record.GetString("name"),
				Description: record.GetString("description"),
				Severity:    record.GetString("severity"),
				Host:        record.GetString("host"),
				Type:        record.GetString("type"),
				Tool:        record.GetString("tool"),
				TemplateID:  record.GetString("template_id"),
				IP:          record.GetString("ip"),
				Port:        record.GetString("port"),
				Protocol:    record.GetString("protocol"),
				Timestamp:   record.GetString("timestamp"),
				MatchedAt:   record.GetString("matched_at"),
				CurlCommand: record.GetString("curl_command"),
				Request:     record.GetString("request"),
				Response:    record.GetString("response"),
				URL:         record.GetString("url"),
			}

			// Safely handle Info field
			if info := record.Get("info"); info != nil {
				if infoMap, ok := info.(map[string]interface{}); ok {
					finding.Info = infoMap
				} else {
					fm.logger.Printf("[WARN] Info field is not a map for record %s", record.Id)
					finding.Info = make(map[string]interface{})
				}
			} else {
				finding.Info = make(map[string]interface{})
			}

			// Safely handle ExtractedResults field
			if results := record.Get("extracted_results"); results != nil {
				if resultsSlice, ok := results.([]string); ok {
					finding.ExtractedResults = resultsSlice
				} else if resultsInterface, ok := results.([]interface{}); ok {
					// Convert []interface{} to []string
					strResults := make([]string, len(resultsInterface))
					for i, v := range resultsInterface {
						if str, ok := v.(string); ok {
							strResults[i] = str
						}
					}
					finding.ExtractedResults = strResults
				} else {
					fm.logger.Printf("[WARN] ExtractedResults field is not a string slice for record %s", record.Id)
					finding.ExtractedResults = []string{}
				}
			} else {
				finding.ExtractedResults = []string{}
			}

			// Generate hash for the finding
			hash := finding.GenerateHash()

			// Check for existing finding with same hash
			existingRecord, err := txDao.FindFirstRecordByFilter(
				"nuclei_findings",
				"(hash = {:hash} && client = {:client} && id != {:id})",
				dbx.Params{
					"hash":   hash,
					"client": finding.ClientID,
					"id":     record.Id,
				},
			)

			if err != nil && err.Error() != "sql: no rows in result set" {
				fm.logger.Printf("[ERROR] Failed to check for existing finding: %v", err)
				continue
			}

			if existingRecord != nil {
				// Get existing scan_ids
				var existingScanIDs []string
				scanIDsStr := existingRecord.GetString("scan_ids")
				if scanIDsStr != "" {
					if err := json.Unmarshal([]byte(scanIDsStr), &existingScanIDs); err != nil {
						fm.logger.Printf("[ERROR] Failed to unmarshal existing scan_ids: %v", err)
						existingScanIDs = []string{}
					}
				}

				// Add current scan_id if not already present
				scanIDExists := false
				for _, id := range existingScanIDs {
					if id == finding.ScanID {
						scanIDExists = true
						break
					}
				}

				if !scanIDExists {
					existingScanIDs = append(existingScanIDs, finding.ScanID)
					scanIDsJSON, err := json.Marshal(existingScanIDs)
					if err != nil {
						fm.logger.Printf("[ERROR] Failed to marshal scan_ids: %v", err)
						continue
					}
					existingRecord.Set("scan_ids", string(scanIDsJSON))
					if err := txDao.SaveRecord(existingRecord); err != nil {
						fm.logger.Printf("[ERROR] Failed to update existing record: %v", err)
						continue
					}
				}

				// Delete the duplicate record
				if err := txDao.DeleteRecord(record); err != nil {
					fm.logger.Printf("[ERROR] Failed to delete duplicate record: %v", err)
				}
			} else {
				// Update the record with the new hash and initialize scan_ids
				record.Set("hash", hash)
				scanIDsJSON, err := json.Marshal([]string{finding.ScanID})
				if err != nil {
					fm.logger.Printf("[ERROR] Failed to marshal scan_ids: %v", err)
					continue
				}
				record.Set("scan_ids", string(scanIDsJSON))

				if err := txDao.SaveRecord(record); err != nil {
					fm.logger.Printf("[ERROR] Failed to save record: %v", err)
					continue
				}
			}

			processedCount++
			if processedCount%100 == 0 {
				fm.logger.Printf("[INFO] Processed %d records", processedCount)
			}
		}

		return nil
	})

	if err != nil {
		return processedCount, fmt.Errorf("failed to process findings batch: %v", err)
	}

	fm.logger.Printf("[INFO] Successfully processed %d records", processedCount)
	return processedCount, nil
}

// Helper function to remove duplicates from a string slice
func removeDuplicates(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Helper function to get map keys
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetUserFindings returns findings for a specific user
func (fm *FindingManager) GetUserFindings(userID string, filter string, sort string, page int, perPage int) ([]*pbModels.Record, error) {
	if filter != "" {
		filter = fmt.Sprintf("(%s) && created_by = {:userID}", filter)
	} else {
		filter = "created_by = {:userID}"
	}

	records, err := fm.app.Dao().FindRecordsByFilter(
		"nuclei_findings",
		filter,
		sort,
		(page-1)*perPage,
		perPage,
		dbx.Params{
			"userID": userID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user findings: %v", err)
	}

	return records, nil
}

// GetUserFindingsCount returns the total count of findings for a specific user
func (fm *FindingManager) GetUserFindingsCount(userID string, filter string) (int, error) {
	if filter != "" {
		filter = fmt.Sprintf("(%s) && created_by = {:userID}", filter)
	} else {
		filter = "created_by = {:userID}"
	}

	count, err := fm.app.Dao().FindRecordsByFilter(
		"nuclei_findings",
		filter,
		"",
		0,
		-1,
		dbx.Params{
			"userID": userID,
		},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to count user findings: %v", err)
	}

	return len(count), nil
}

// GetUserFindingsSummary returns a summary of findings for a specific user
func (fm *FindingManager) GetUserFindingsSummary(userID string) (map[string]int, error) {
	records, err := fm.app.Dao().FindRecordsByFilter(
		"nuclei_findings",
		"created_by = {:userID}",
		"",
		0,
		-1,
		dbx.Params{
			"userID": userID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user findings: %v", err)
	}

	summary := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
		"total":    len(records),
	}

	for _, record := range records {
		severity := strings.ToLower(record.GetString("severity"))
		if count, ok := summary[severity]; ok {
			summary[severity] = count + 1
		}
	}

	return summary, nil
}
