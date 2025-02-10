package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"

	"orbit/models"
)

func HandleImportNucleiScanResults(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Retrieve form values
		clientID := c.FormValue("client_id")
		scanID := c.FormValue("scan_id")

		if clientID == "" || scanID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing client_id or scan_id",
			})
		}

		// Retrieve the file from the request
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid file",
			})
		}

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to open file",
			})
		}
		defer src.Close()

		// Read the JSON file
		jsonData, err := io.ReadAll(src)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to read file",
			})
		}

		var findings []models.NucleiFinding
		if err := json.Unmarshal(jsonData, &findings); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to parse JSON",
			})
		}

		// Send success response immediately
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if _, err := c.Response().Write([]byte(`{"status": "processing"}`)); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
		c.Response().Flush()

		// Process findings in a goroutine
		go func() {
			for _, finding := range findings {
				// Serialize the Info struct to JSON
				infoJSON, err := json.Marshal(finding.Info)
				if err != nil {
					log.Printf("Failed to marshal Info field for finding %s: %v", finding.Info.Name, err)
					continue
				}

				// Determine the severity order
				severity := strings.ToLower(finding.Info.Severity)
				severityOrder, exists := SeverityOrderMap[severity]
				if !exists {
					severityOrder = 99 // Default for undefined severities
				}

				// Prepare the data for PocketBase, including severity_order
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
				}

				// Build the duplicate check expression using fields that uniquely identify the finding
				duplicateCheckFields := dbx.HashExp{
					"template_id": finding.TemplateID,
					"host":        finding.Host,
					"port":        finding.Port,
					"url":         finding.URL,
					"matched_at":  finding.MatchedAt,
					"client":      clientID,
				}

				existingRecords, err := app.Dao().FindRecordsByExpr("nuclei_results", duplicateCheckFields)
				if err != nil {
					log.Printf("Error finding existing records: %v", err)
					continue
				}

				if len(existingRecords) > 0 {
					// If a record exists, update only the last_seen timestamp if the new timestamp is more recent
					existingRecord := existingRecords[0]
					currentLastSeen := existingRecord.GetDateTime("last_seen")

					// Only update last_seen if it's empty or the new timestamp is more recent
					if currentLastSeen.IsZero() {
						existingRecord.Set("last_seen", finding.Timestamp)
						if err := app.Dao().SaveRecord(existingRecord); err != nil {
							log.Printf("Failed to update record last_seen: %v", err)
							continue
						}
						log.Printf("Updated last_seen for record: %s", finding.Info.Name)
					} else {
						// Compare timestamps using Unix timestamps for comparison
						newTime := existingRecord.GetDateTime("timestamp")
						if newTime.Time().Unix() > currentLastSeen.Time().Unix() {
							existingRecord.Set("last_seen", finding.Timestamp)
							if err := app.Dao().SaveRecord(existingRecord); err != nil {
								log.Printf("Failed to update record last_seen: %v", err)
								continue
							}
							log.Printf("Updated last_seen for record: %s", finding.Info.Name)
						} else {
							log.Printf("Skipped updating last_seen as new timestamp is older for: %s", finding.Info.Name)
						}
					}
				} else {
					// If no record exists, create a new one with timestamp as both first seen and last seen
					collection, err := app.Dao().FindCollectionByNameOrId("nuclei_results")
					if err != nil {
						log.Printf("Failed to find collection: %v", err)
						continue
					}

					record := models.NewRecord(collection)

					// Set all fields from the data map
					for key, value := range data {
						record.Set(key, value)
					}

					// Set the initial timestamp as both first seen (timestamp field) and last seen
					record.Set("timestamp", finding.Timestamp) // This is our first_seen
					record.Set("last_seen", finding.Timestamp) // Initialize last_seen to the same value

					// Handle []string fields by converting them to JSON strings
					// Info.Author
					if len(finding.Info.Author) > 0 {
						authorBytes, err := json.Marshal(finding.Info.Author)
						if err != nil {
							log.Printf("Failed to marshal author field: %v", err)
							continue
						}
						record.Set("author", string(authorBytes))
					} else {
						record.Set("author", "")
					}

					// Info.Tags
					if len(finding.Info.Tags) > 0 {
						tagsBytes, err := json.Marshal(finding.Info.Tags)
						if err != nil {
							log.Printf("Failed to marshal tags field: %v", err)
							continue
						}
						record.Set("tags", string(tagsBytes))
					} else {
						record.Set("tags", "")
					}

					// Info.Reference
					if len(finding.Info.Reference) > 0 {
						refBytes, err := json.Marshal(finding.Info.Reference)
						if err != nil {
							log.Printf("Failed to marshal reference field: %v", err)
							continue
						}
						record.Set("reference", string(refBytes))
					} else {
						record.Set("reference", "")
					}

					// ExtractedResults
					if len(finding.ExtractedResults) > 0 {
						extractedBytes, err := json.Marshal(finding.ExtractedResults)
						if err != nil {
							log.Printf("Failed to marshal extracted_results field: %v", err)
							continue
						}
						record.Set("extracted_results", string(extractedBytes))
					} else {
						record.Set("extracted_results", "")
					}

					// Info.Classification.CweID
					if len(finding.Info.Classification.CweID) > 0 {
						cweBytes, err := json.Marshal(finding.Info.Classification.CweID)
						if err != nil {
							log.Printf("Failed to marshal cwe_id field: %v", err)
							continue
						}
						record.Set("cwe_id", string(cweBytes))
					} else {
						record.Set("cwe_id", "")
					}

					// Info.Classification.CveID (interface{})
					if finding.Info.Classification.CveID != nil {
						cveIDBytes, err := json.Marshal(finding.Info.Classification.CveID)
						if err != nil {
							log.Printf("Failed to marshal cve_id field: %v", err)
							continue
						}
						record.Set("cve_id", string(cveIDBytes))
					} else {
						record.Set("cve_id", "")
					}

					if err := app.Dao().SaveRecord(record); err != nil {
						log.Printf("Failed to save record: %v", err)
						continue
					}
					log.Printf("Created new record for finding: %s", finding.Info.Name)
				}
			}
			log.Printf("Finished processing %d findings for scan %s", len(findings), scanID)
		}()

		return nil
	}
}
