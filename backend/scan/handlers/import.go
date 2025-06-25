package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"bitor/services"
	"bitor/services/notification"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	pbModels "github.com/pocketbase/pocketbase/models"

	bitorModels "bitor/models"
)

// chunkInfo stores information about uploaded chunks
type chunkInfo struct {
	TotalChunks int
	Chunks      map[int]bool
	FilePath    string
}

// In-memory store for chunk tracking
var (
	chunkTracker = make(map[string]*chunkInfo)
	chunkMutex   sync.Mutex
)

// Initialize services
var (
	findingManager      *services.FindingManager
	scanEventService    *services.ScanEventService
	notificationService *notification.NotificationService
)

// InitHandlers initializes the handlers with required services
func InitHandlers(app *pocketbase.PocketBase, ns *notification.NotificationService) {
	notificationService = ns
	findingManager = services.NewFindingManager(app, notificationService)
	scanEventService = services.NewScanEventService(app, findingManager)
}

func HandleImportNucleiScanResults(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get the current user or admin from context
		admin, _ := c.Get(apis.ContextAdminKey).(*pbModels.Admin)
		record, _ := c.Get(apis.ContextAuthRecordKey).(*pbModels.Record)

		// Get user ID (either from admin, regular user, or fallback for API key auth)
		var userID string
		if admin != nil {
			userID = admin.Id
		} else if record != nil {
			userID = record.Id
		} else {
			// If neither admin nor user is set, this request was authenticated via API key
			// by the middleware, so we'll use the scan's created_by field later
			userID = "api-key-auth"
		}

		// Retrieve form values
		clientID := c.FormValue("client_id")
		scanID := c.FormValue("scan_id")
		chunkIndexStr := c.FormValue("chunk_index")
		totalChunksStr := c.FormValue("total_chunks")

		if clientID == "" || scanID == "" {
			log.Printf("Error: Missing client_id or scan_id")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing client_id or scan_id",
			})
		}

		// Get the uploaded file
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid file",
			})
		}

		// Create temporary directory if it doesn't exist
		tempDir := filepath.Join(os.TempDir(), "bitor_uploads")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create temp directory",
			})
		}

		// Check if this is a chunked upload
		isChunkedUpload := chunkIndexStr != "" && totalChunksStr != ""

		if isChunkedUpload {
			// Parse chunk information
			chunkIndex, err := strconv.Atoi(chunkIndexStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid chunk_index",
				})
			}

			totalChunks, err := strconv.Atoi(totalChunksStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid total_chunks",
				})
			}

			chunkMutex.Lock()
			info, exists := chunkTracker[scanID]
			if !exists {
				info = &chunkInfo{
					TotalChunks: totalChunks,
					Chunks:      make(map[int]bool),
					FilePath:    filepath.Join(tempDir, fmt.Sprintf("%s.json", scanID)),
				}
				chunkTracker[scanID] = info
			}
			chunkMutex.Unlock()

			// Open chunk file
			src, err := file.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to open uploaded chunk",
				})
			}
			defer src.Close()

			// Open or create the destination file in append mode
			dest, err := os.OpenFile(info.FilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to create destination file",
				})
			}
			defer dest.Close()

			// Copy chunk data
			if _, err := io.Copy(dest, src); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to write chunk",
				})
			}

			// Mark chunk as received
			chunkMutex.Lock()
			info.Chunks[chunkIndex] = true
			receivedChunks := len(info.Chunks)
			isComplete := receivedChunks == info.TotalChunks
			chunkMutex.Unlock()

			// If all chunks received, process the file
			if isComplete {
				log.Printf("All chunks received for scan %s, starting processing", scanID)
				go processFile(app, info.FilePath, scanID, clientID, userID)
				return c.JSON(http.StatusOK, map[string]string{
					"status": "All chunks received and processing started",
				})
			}

			return c.JSON(http.StatusOK, map[string]string{
				"status": fmt.Sprintf("Chunk %d received (%d/%d complete)", chunkIndex, receivedChunks, totalChunks),
			})
		} else {
			// Handle single file upload
			filePath := filepath.Join(tempDir, fmt.Sprintf("%s.json", scanID))

			// Open the uploaded file
			src, err := file.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to open uploaded file",
				})
			}
			defer src.Close()

			// Create the destination file
			dest, err := os.Create(filePath)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to create destination file",
				})
			}
			defer dest.Close()

			// Copy file data
			if _, err := io.Copy(dest, src); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to write file",
				})
			}

			log.Printf("Processing file for scan %s", scanID)
			processFile(app, filePath, scanID, clientID, userID)

			return c.JSON(http.StatusOK, map[string]string{
				"status": "File processed successfully",
			})
		}
	}
}

// processFile handles the processing of the complete file
func processFile(app *pocketbase.PocketBase, filePath string, scanID string, clientID string, userID string) {
	var logger *log.Logger

	// Create logs directory if it doesn't exist
	logsDir := filepath.Join("logs", "scans")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("[ERROR] Failed to create logs directory: %v", err)
		logger = log.Default()
	} else {
		// Create a log file for this scan
		logFileName := filepath.Join(logsDir, fmt.Sprintf("scan_%s_import.log", scanID))
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[ERROR] Failed to create log file %s: %v", logFileName, err)
			logger = log.Default()
		} else {
			defer logFile.Close()
			// Create a multi-writer to write to both file and stdout
			multiWriter := io.MultiWriter(os.Stdout, logFile)
			logger = log.New(multiWriter, "", log.LstdFlags)
			logger.Printf("[INFO] Starting import process for scan %s", scanID)
			logger.Printf("[INFO] Log file created at: %s", logFileName)
		}
	}

	defer os.Remove(filePath)

	// Get the scan record to check its status
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		logger.Printf("[ERROR] Failed to find scan record: %v", err)
		return
	}

	// Read the complete file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		logger.Printf("[ERROR] Failed to read complete file: %v", err)
		return
	}

	// Log file size and path
	logger.Printf("[DEBUG] Read JSON file size: %d bytes", len(jsonData))
	logger.Printf("[DEBUG] JSON file path: %s", filePath)

	// First try to parse as array
	var findings []bitorModels.NucleiFinding
	if err := json.Unmarshal(jsonData, &findings); err != nil {
		logger.Printf("[ERROR] Failed to parse JSON as array: %v", err)

		// Try to parse as single finding
		var singleFinding bitorModels.NucleiFinding
		if err := json.Unmarshal(jsonData, &singleFinding); err != nil {
			logger.Printf("[ERROR] Failed to parse JSON as single finding: %v", err)
			return
		}
		findings = []bitorModels.NucleiFinding{singleFinding}
	}

	// Log the actual number of findings
	logger.Printf("[DEBUG] ===== Findings Summary =====")
	logger.Printf("[DEBUG] Raw JSON size: %d bytes", len(jsonData))
	logger.Printf("[DEBUG] Total findings after parse: %d", len(findings))
	logger.Printf("[DEBUG] First 10 findings details:")
	for i := 0; i < len(findings) && i < 10; i++ {
		findingJSON, _ := json.Marshal(findings[i])
		logger.Printf("[DEBUG] Finding %d:", i)
		logger.Printf("[DEBUG]   - Name: %s", findings[i].Info.Name)
		logger.Printf("[DEBUG]   - Template ID: %s", findings[i].TemplateID)
		logger.Printf("[DEBUG]   - Host: %s", findings[i].Host)
		logger.Printf("[DEBUG]   - Size: %d bytes", len(findingJSON))
	}
	logger.Printf("[DEBUG] =========================")

	// Only update status if it's not a manual scan
	currentStatus := record.GetString("status")
	if currentStatus != "Manual" {
		record.Set("status", "Finished")
		if err := app.Dao().SaveRecord(record); err != nil {
			logger.Printf("[ERROR] Failed to update scan status: %v", err)
			return
		}
	}

	// Get the created_by from the scan record
	scanCreatedBy := record.GetString("created_by")
	if scanCreatedBy == "" || userID == "api-key-auth" {
		if scanCreatedBy != "" {
			userID = scanCreatedBy // Use the scan's created_by field for API key auth
		} else {
			userID = "system" // Fallback if nothing is available
		}
	}
	logger.Printf("[DEBUG] Using created_by: %s for findings", userID)

	// Process findings in parallel
	processFindings(app, findings, clientID, scanID, logger, userID)

	// Trigger scan finished event to create nuclei_findings_rollup
	if err := scanEventService.HandleScanFinished(scanID); err != nil {
		logger.Printf("[ERROR] Error triggering scan finished event: %v", err)
	}

	// Clean up the scan tracker after processing is complete
	findingManager.FinalizeScan(scanID)
	logger.Printf("[INFO] Import process completed for scan %s", scanID)
}

// Helper function to get severity order
func getSeverityOrder(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	case "info":
		return 5
	default:
		return 0 // Unknown severity
	}
}

// Process findings in parallel
func processFindings(app *pocketbase.PocketBase, findings []bitorModels.NucleiFinding, clientID string, scanID string, logger *log.Logger, userID string) {
	logger.Printf("[DEBUG] Starting processFindings for scan %s with %d total findings", scanID, len(findings))
	logger.Printf("[DEBUG] Using userID %s for created_by field", userID)

	// Map to track unique findings
	findingsMap := make(map[string]struct {
		finding      *bitorModels.Finding
		originalData string
	})

	// Map to track duplicate counts by template
	duplicatesByTemplate := make(map[string]int)

	// First pass: identify unique findings and check for hash collisions
	for _, finding := range findings {
		newFinding, err := bitorModels.NewFindingFromNuclei(finding, clientID, scanID, userID)
		if err != nil {
			logger.Printf("[ERROR] Error creating finding: %v", err)
			continue
		}

		hash := newFinding.GenerateHash()
		if entry, exists := findingsMap[hash]; exists {
			// Track duplicates by template
			duplicatesByTemplate[newFinding.TemplateID]++

			// Special debug logging for RDAP/WHOIS findings
			if newFinding.TemplateID == "rdap-whois" {
				logger.Printf("[DEBUG] RDAP/WHOIS Duplicate Found:")
				logger.Printf("[DEBUG] Original Finding:")
				logger.Printf("[DEBUG]   Host: %s", entry.finding.Host)
				logger.Printf("[DEBUG]   URL: %s", entry.finding.URL)
				logger.Printf("[DEBUG]   MatchedAt: %s", entry.finding.MatchedAt)
				logger.Printf("[DEBUG]   ExtractedResults: %v", entry.finding.ExtractedResults)
				logger.Printf("[DEBUG]   Hash: %s", hash)
				logger.Printf("[DEBUG] Duplicate Finding:")
				logger.Printf("[DEBUG]   Host: %s", newFinding.Host)
				logger.Printf("[DEBUG]   URL: %s", newFinding.URL)
				logger.Printf("[DEBUG]   MatchedAt: %s", newFinding.MatchedAt)
				logger.Printf("[DEBUG]   ExtractedResults: %v", newFinding.ExtractedResults)
				logger.Printf("[DEBUG]   Hash: %s", hash)
			}

			// Log duplicate details every 100th duplicate or if it's a new template
			if duplicatesByTemplate[newFinding.TemplateID] == 1 || duplicatesByTemplate[newFinding.TemplateID]%100 == 0 {
				logger.Printf("[DEBUG] Found duplicate #%d for template %s",
					duplicatesByTemplate[newFinding.TemplateID],
					newFinding.TemplateID)
				logger.Printf("[DEBUG]   Original: Name=%s, Host=%s, MatchedAt=%s",
					entry.finding.Name,
					entry.finding.Host,
					entry.finding.MatchedAt)
				logger.Printf("[DEBUG]   Duplicate: Name=%s, Host=%s, MatchedAt=%s",
					newFinding.Name,
					newFinding.Host,
					newFinding.MatchedAt)
			}
		} else {
			findingsMap[hash] = struct {
				finding      *bitorModels.Finding
				originalData string
			}{
				finding:      newFinding,
				originalData: fmt.Sprintf("%+v", finding),
			}
		}
	}

	logger.Printf("[DEBUG] Found %d unique findings", len(findingsMap))
	logger.Printf("[DEBUG] Duplicates by template:")
	for templateID, count := range duplicatesByTemplate {
		logger.Printf("[DEBUG]   - %s: %d duplicates", templateID, count)
	}

	// Process each unique finding and track counts
	totalNew := 0
	duplicatesInDB := 0
	totalProcessed := len(findingsMap)

	// Process each unique finding once
	for hash, entry := range findingsMap {
		logger.Printf("[DEBUG] Processing hash %s", hash[:8])

		// Process finding through manager
		isDuplicate, err := findingManager.ProcessFinding(entry.finding)
		if err != nil {
			logger.Printf("[ERROR] Error processing finding: %v", err)
			continue
		}
		logger.Printf("[DEBUG] Hash %s isDuplicate: %v", hash[:8], isDuplicate)

		if isDuplicate {
			duplicatesInDB++
			logger.Printf("[DEBUG] Hash %s is duplicate in DB, total duplicatesInDB: %d", hash[:8], duplicatesInDB)
		} else {
			totalNew++
			logger.Printf("[DEBUG] Hash %s is new, total new: %d", hash[:8], totalNew)
		}
	}

	// Get scan name for the notification
	scanRecord, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		logger.Printf("[ERROR] Error getting scan name: %v", err)
		return
	}
	scanName := scanRecord.GetString("name")

	// Verify our counts add up
	if totalNew+duplicatesInDB != totalProcessed {
		logger.Printf("[ERROR] Count mismatch! Total processed: %d, but got new: %d + duplicates: %d = %d",
			totalProcessed, totalNew, duplicatesInDB, totalNew+duplicatesInDB)
	}

	// Log final counts
	logger.Printf("[DEBUG] FINAL COUNTS for scan %s:", scanID)
	logger.Printf("[DEBUG] - Total findings: %d", len(findings))
	logger.Printf("[DEBUG] - Unique findings: %d", len(findingsMap))
	logger.Printf("[DEBUG] - New findings: %d", totalNew)
	logger.Printf("[DEBUG] - Duplicates in database: %d", duplicatesInDB)

	// Create user message about import completion
	message := fmt.Sprintf("Scan '%s' completed: %d total findings (%d new, %d duplicates in database)",
		scanName, len(findings), totalNew, duplicatesInDB)

	if err := createUserMessage(app, clientID, scanID, message, "info"); err != nil {
		logger.Printf("[ERROR] Error creating user message: %v", err)
	}
}

// saveBatch saves a batch of findings to the database and returns the number of successful saves
func saveBatch(app *pocketbase.PocketBase, collection *pbModels.Collection, batch []map[string]interface{}) int {
	savedCount := 0
	for _, data := range batch {
		record := pbModels.NewRecord(collection)
		for key, value := range data {
			if value != nil && value != "" {
				record.Set(key, value)
			}
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			log.Printf("Error saving record: %v", err)
			continue
		}
		savedCount++
		log.Printf("Added new finding: %s (hash: %s)", data["name"], data["hash"])
	}
	return savedCount
}

func createUserMessage(app *pocketbase.PocketBase, creator string, scanId string, message string, messageType string) error {
	// Ensure messageType is one of the allowed values
	switch messageType {
	case "info", "success", "warning", "error":
		// These are valid types
	default:
		messageType = "info" // Default to info for unknown types
	}

	log.Printf("Creating user message for creator %s and scan %s", creator, scanId)

	// Get the scan record to get the creator
	scanRecord, err := app.Dao().FindRecordById("nuclei_scans", scanId)
	if err != nil {
		return fmt.Errorf("failed to find scan record: %v", err)
	}

	// Get the creator from the scan record
	creator = scanRecord.GetString("created_by")
	if creator == "" {
		log.Printf("No creator found for scan %s, creating message without user association", scanId)
		// Create message without user association so super admins can see it
		collection, err := app.Dao().FindCollectionByNameOrId("user_messages")
		if err != nil {
			return fmt.Errorf("failed to find user_messages collection: %v", err)
		}

		record := pbModels.NewRecord(collection)
		record.Set("message", fmt.Sprintf("%s: %s", message, scanId))
		record.Set("type", messageType)
		record.Set("read", false)

		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save message: %v", err)
		}
		return nil
	}

	// Try to find the creator in the admin collection first
	_, err = app.Dao().FindRecordById("_pb_users_auth_", creator)
	if err == nil {
		// Creator is an admin
		log.Printf("Found admin record for creator %s", creator)
		collection, err := app.Dao().FindCollectionByNameOrId("user_messages")
		if err != nil {
			return fmt.Errorf("failed to find user_messages collection: %v", err)
		}

		record := pbModels.NewRecord(collection)
		record.Set("message", fmt.Sprintf("%s: %s", message, scanId))
		record.Set("type", messageType)
		record.Set("read", false)
		record.Set("admin_id", creator)

		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save admin message: %v", err)
		}
		return nil
	}

	// Try to find the creator in the users collection
	_, err = app.Dao().FindRecordById("27do0wbcuyfmbmx", creator)
	if err == nil {
		// Creator is a regular user
		log.Printf("Found user record for creator %s", creator)
		collection, err := app.Dao().FindCollectionByNameOrId("user_messages")
		if err != nil {
			return fmt.Errorf("failed to find user_messages collection: %v", err)
		}

		record := pbModels.NewRecord(collection)
		record.Set("message", fmt.Sprintf("%s: %s", message, scanId))
		record.Set("type", messageType)
		record.Set("read", false)
		record.Set("user", creator)

		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save user message: %v", err)
		}
		return nil
	}

	// If we get here, create a message without user association for super admins
	log.Printf("Creating message without user association for scan %s", scanId)
	collection, err := app.Dao().FindCollectionByNameOrId("user_messages")
	if err != nil {
		return fmt.Errorf("failed to find user_messages collection: %v", err)
	}

	record := pbModels.NewRecord(collection)
	record.Set("message", fmt.Sprintf("%s: %s", message, scanId))
	record.Set("type", messageType)
	record.Set("read", false)

	if err := app.Dao().SaveRecord(record); err != nil {
		return fmt.Errorf("failed to save message: %v", err)
	}
	return nil
}
