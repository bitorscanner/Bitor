package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"

	orbitModels "orbit/models"
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

func HandleImportNucleiScanResults(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Retrieve form values
		clientID := c.FormValue("client_id")
		scanID := c.FormValue("scan_id")
		chunkIndexStr := c.FormValue("chunk_index")
		totalChunksStr := c.FormValue("total_chunks")

		if clientID == "" || scanID == "" {
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
		tempDir := filepath.Join(os.TempDir(), "orbit_uploads")
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
			isComplete := len(info.Chunks) == info.TotalChunks
			chunkMutex.Unlock()

			// If all chunks received, process the file
			if isComplete {
				go processFile(app, info.FilePath, scanID, clientID)
				return c.JSON(http.StatusOK, map[string]string{
					"status": "All chunks received and processing started",
				})
			}

			return c.JSON(http.StatusOK, map[string]string{
				"status": "Chunk received",
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

			// Process the file
			go processFile(app, filePath, scanID, clientID)

			return c.JSON(http.StatusOK, map[string]string{
				"status": "File received and processing started",
			})
		}
	}
}

// processFile handles the processing of the complete file
func processFile(app *pocketbase.PocketBase, filePath string, scanID string, clientID string) {
	defer os.Remove(filePath)

	// Read the complete file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to read complete file: %v", err)
		return
	}

	var findings []orbitModels.NucleiFinding
	if err := json.Unmarshal(jsonData, &findings); err != nil {
		log.Printf("Failed to parse JSON: %v", err)
		return
	}

	// Create a worker pool for parallel processing
	numWorkers := 4
	findingsChan := make(chan orbitModels.NucleiFinding)
	resultsChan := make(chan map[string]interface{}, len(findings))
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for finding := range findingsChan {
				// Process finding and prepare data
				data := prepareFindingData(finding, clientID, scanID)
				resultsChan <- data
			}
		}()
	}

	// Send findings to workers
	go func() {
		for _, finding := range findings {
			findingsChan <- finding
		}
		close(findingsChan)
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Process results in batches of 100
	batchSize := 100
	currentBatch := make([]map[string]interface{}, 0, batchSize)
	duplicateCheckMap := make(map[string]bool)

	collection, err := app.Dao().FindCollectionByNameOrId("nuclei_results")
	if err != nil {
		log.Printf("Failed to find collection: %v", err)
		return
	}

	for data := range resultsChan {
		// Create unique key for duplicate checking
		uniqueKey := fmt.Sprintf("%s-%s-%s-%s-%s",
			data["template_id"],
			data["host"],
			data["port"],
			data["url"],
			data["matched_at"])

		if !duplicateCheckMap[uniqueKey] {
			duplicateCheckMap[uniqueKey] = true
			currentBatch = append(currentBatch, data)

			// When batch is full, process it
			if len(currentBatch) >= batchSize {
				if err := processBatch(app, collection, currentBatch); err != nil {
					log.Printf("Error processing batch: %v", err)
				}
				currentBatch = make([]map[string]interface{}, 0, batchSize)
			}
		}
	}

	// Process remaining items
	if len(currentBatch) > 0 {
		if err := processBatch(app, collection, currentBatch); err != nil {
			log.Printf("Error processing final batch: %v", err)
		}
	}

	log.Printf("Finished processing %d findings for scan %s", len(findings), scanID)
}

// prepareFindingData prepares the data for a single finding
func prepareFindingData(finding orbitModels.NucleiFinding, clientID string, scanID string) map[string]interface{} {
	// Serialize the Info struct to JSON
	infoJSON, err := json.Marshal(finding.Info)
	if err != nil {
		log.Printf("Failed to marshal Info field for finding %s: %v", finding.Info.Name, err)
		return nil
	}

	// Determine the severity order
	severity := strings.ToLower(finding.Info.Severity)
	severityOrder, exists := SeverityOrderMap[severity]
	if !exists {
		severityOrder = 99 // Default for undefined severities
	}

	// Prepare the data
	data := map[string]interface{}{
		"template_id":    finding.TemplateID,
		"template_path":  finding.TemplatePath,
		"name":           finding.Info.Name,
		"description":    finding.Info.Description,
		"type":           finding.Type,
		"host":           finding.Host,
		"port":           finding.Port,
		"scheme":         finding.Scheme,
		"url":            finding.URL,
		"matched_at":     finding.MatchedAt,
		"request":        finding.Request,
		"response":       finding.Response,
		"ip":             finding.IP,
		"curl_command":   finding.CurlCommand,
		"matcher_status": finding.MatcherStatus,
		"severity":       finding.Info.Severity,
		"severity_order": severityOrder,
		"client":         clientID,
		"scan_id":        scanID,
		"timestamp":      finding.Timestamp,
		"info":           string(infoJSON),
		"last_seen":      finding.Timestamp,
	}

	// Handle array fields
	if len(finding.Info.Author) > 0 {
		if authorBytes, err := json.Marshal(finding.Info.Author); err == nil {
			data["author"] = string(authorBytes)
		}
	}
	if len(finding.Info.Tags) > 0 {
		if tagsBytes, err := json.Marshal(finding.Info.Tags); err == nil {
			data["tags"] = string(tagsBytes)
		}
	}
	if len(finding.Info.Reference) > 0 {
		if refBytes, err := json.Marshal(finding.Info.Reference); err == nil {
			data["reference"] = string(refBytes)
		}
	}
	if len(finding.ExtractedResults) > 0 {
		if extractedBytes, err := json.Marshal(finding.ExtractedResults); err == nil {
			data["extracted_results"] = string(extractedBytes)
		}
	}
	if len(finding.Info.Classification.CweID) > 0 {
		if cweBytes, err := json.Marshal(finding.Info.Classification.CweID); err == nil {
			data["cwe_id"] = string(cweBytes)
		}
	}
	if finding.Info.Classification.CveID != nil {
		if cveIDBytes, err := json.Marshal(finding.Info.Classification.CveID); err == nil {
			data["cve_id"] = string(cveIDBytes)
		}
	}

	return data
}

// processBatch handles batch insertion of findings
func processBatch(app *pocketbase.PocketBase, collection *models.Collection, batch []map[string]interface{}) error {
	// Process each record in the batch
	for _, data := range batch {
		record := models.NewRecord(collection)
		for key, value := range data {
			record.Set(key, value)
		}
		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save record: %v", err)
		}
	}

	return nil
}
