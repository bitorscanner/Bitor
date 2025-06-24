package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"

	"bitor/models"
	"bitor/scan/utils"
	"bitor/services/notification"
)

// HandleStopScan stops the scan process.
func HandleStopScan(app *pocketbase.PocketBase, ansibleBasePath string, notificationService *notification.NotificationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Initialize the database path
		utils.InitDBPath(app)

		// Bind the request payload
		var scanReq models.ScanRequest
		if err := c.Bind(&scanReq); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID is provided
		if scanReq.ScanID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID is required",
			})
		}

		// Update scan status to "Stopping"
		record, err := app.Dao().FindRecordById("nuclei_scans", scanReq.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Get the start time and calculate the final cost
		startTimeStr := record.GetString("start_time")
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			// Try alternative format
			startTime, err = time.Parse("2006-01-02 15:04:05.000Z", startTimeStr)
			if err != nil {
				log.Printf("Failed to parse start time: %v", err)
				startTime = time.Now() // Fallback to now if parsing fails
			}
		}

		endTime := time.Now()
		record.Set("end_time", endTime.Format(time.RFC3339))

		// Calculate and set the final cost if not already set
		if record.GetFloat("cost") == 0 {
			// Get the VM size from the record
			vmSize := record.GetString("vm_size")
			if vmSize != "" {
				// Get the provider ID
				providerId := record.GetString("vm_provider")
				if providerId != "" {
					// Get the provider record to get the region
					provider, err := app.Dao().FindRecordById("providers", providerId)
					if err != nil {
						log.Printf("Failed to find provider: %v", err)
					} else {
						// Get region from provider settings
						var settings struct {
							Region string `json:"region"`
						}
						settingsData := provider.Get("settings")
						if settingsData == nil {
							log.Printf("Settings not found in provider")
						} else {
							settingsBytes, err := json.Marshal(settingsData)
							if err != nil {
								log.Printf("Failed to marshal settings: %v", err)
							} else if err := json.Unmarshal(settingsBytes, &settings); err != nil {
								log.Printf("Failed to unmarshal settings: %v", err)
							} else if settings.Region == "" {
								log.Printf("Region not found in provider settings")
							} else {
								// Calculate duration in hours
								duration := endTime.Sub(startTime).Hours()

								// Get the hourly price for the VM size
								hourlyPrice, err := utils.GetVMPrice(app, providerId, settings.Region, vmSize)
								if err == nil && hourlyPrice > 0 {
									finalCost := duration * hourlyPrice
									record.Set("cost", finalCost)
								} else {
									log.Printf("Failed to get VM price: %v", err)
								}
							}
						}
					}
				}
			}
		}

		// Run the destroy playbook
		playbookPath := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "destroy.yml")
		logDir := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "logs")
		yamlFile := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "scan.yaml")
		inventoryPath := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "inventory")

		// Execute Ansible playbook
		if err := utils.ExecuteAnsiblePlaybook(
			playbookPath,
			logDir,
			yamlFile,
			inventoryPath,
			ansibleBasePath,
			app,
			scanReq.ScanID,
		); err != nil {
			log.Printf("Failed to run Ansible destroy playbook: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to run Ansible destroy playbook",
			})
		}

		// Update the scan record to set destroyed=true
		if _, err := app.Dao().DB().
			NewQuery("UPDATE nuclei_scans SET destroyed = true WHERE id = {:id}").
			Bind(map[string]any{"id": scanReq.ScanID}).
			Execute(); err != nil {
			log.Printf("Failed to update destroyed status: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update destroyed status",
			})
		}

		// Save the record with end time and cost
		if err := app.Dao().SaveRecord(record); err != nil {
			log.Printf("Failed to update end time and cost: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update end time and cost",
			})
		}

		record.Set("status", "Stopped")
		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan status",
			})
		}

		// Send notification for scan stopped
		if err := notificationService.NotifyScanStopped(context.Background(), scanReq.ScanID, record.GetString("name")); err != nil {
			log.Printf("Failed to send scan stopped notification: %v", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan stopped",
		})
	}
}
