package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"

	"bitor/models"
	"bitor/scan/utils"
)

// HandleDestroyScan handles the destruction of a scan.
func HandleDestroyScan(app *pocketbase.PocketBase, ansibleBasePath string) echo.HandlerFunc {
	return func(c echo.Context) error {
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

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", scanReq.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Define the path to the destroy playbook
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

		// Get current time in RFC3339 format
		currentTime := time.Now().Format(time.RFC3339)

		// Update the scan record to set destroyed=true, end_time, and vm_stop_time
		record.Set("destroyed", true)
		record.Set("end_time", currentTime)
		record.Set("vm_stop_time", currentTime)

		if err := app.Dao().SaveRecord(record); err != nil {
			log.Printf("Failed to update scan record: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan record",
			})
		}

		// Delete the scan folder
		scanFolderPath := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID)
		if err := os.RemoveAll(scanFolderPath); err != nil {
			log.Printf("Failed to delete scan folder: %v", err)
			// Don't return error here as the main operation succeeded
		}

		// Return a success response
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Scan destruction completed successfully",
		})
	}
}
