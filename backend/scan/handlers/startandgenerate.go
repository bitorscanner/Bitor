package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"

	"bitor/models"
	"bitor/scan/utils"
	"bitor/services/notification"
)

// updateScanStatus safely updates the scan status by fetching a fresh copy first
func updateScanStatus(app *pocketbase.PocketBase, scanID string, status string) error {
	// Always fetch a fresh copy of the record to preserve all fields
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to fetch record: %v", err)
	}

	// Get existing logs to preserve them
	var existingLogs interface{}
	if logsValue := record.Get("ansible_logs"); logsValue != nil {
		existingLogs = logsValue
	}

	// Update the status
	record.Set("status", status)

	// If status is Failed or Destroyed, set end time and destroyed flag
	if status == "Failed" || status == "Destroyed" {
		currentTime := time.Now().Format(time.RFC3339)
		record.Set("end_time", currentTime)
		record.Set("vm_stop_time", currentTime)
		if status == "Destroyed" {
			record.Set("destroyed", true)
		}
	}

	// Preserve existing logs if they exist
	if existingLogs != nil {
		record.Set("ansible_logs", existingLogs)
	}

	if err := app.Dao().SaveRecord(record); err != nil {
		return fmt.Errorf("failed to update status: %v", err)
	}

	return nil
}

// validatePlaybook runs ansible-playbook --syntax-check to catch parsing errors
func validatePlaybook(playbookPath string, scanID string, app *pocketbase.PocketBase) error {
	// Create a buffer to capture error output
	var errBuf bytes.Buffer

	// Run ansible-playbook --syntax-check
	cmd := exec.Command("ansible-playbook", "--syntax-check", playbookPath)
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		// Get the error output
		errOutput := errBuf.String()

		// Update scan record with the error
		record, findErr := app.Dao().FindRecordById("nuclei_scans", scanID)
		if findErr == nil {
			// Get existing logs
			var existingLogs []interface{}
			if logsValue := record.Get("ansible_logs"); logsValue != nil {
				// Try to unmarshal the logs regardless of type
				var logs []interface{}
				rawData, err := json.Marshal(logsValue)
				if err == nil {
					if err := json.Unmarshal(rawData, &logs); err == nil {
						existingLogs = logs
						log.Printf("Successfully loaded %d existing logs", len(existingLogs))
					}
				}
			}
			if existingLogs == nil {
				existingLogs = make([]interface{}, 0)
			}

			// Add error to ansible_logs
			logEntry := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"content":   errOutput,
				"type":      "stderr",
			}

			// Append the new log entry
			existingLogs = append(existingLogs, logEntry)

			// Update the record with combined logs
			record.Set("ansible_logs", existingLogs)
			if err := app.Dao().SaveRecord(record); err != nil {
				log.Printf("Failed to save validation error logs: %v", err)
			} else {
				log.Printf("Successfully saved %d logs including validation error", len(existingLogs))
			}
		}

		return fmt.Errorf("playbook validation failed: %v\n%s", err, errOutput)
	}

	return nil
}

func HandleStartAndGenerateScan(app *pocketbase.PocketBase, ansibleBasePath string, notificationService *notification.NotificationService) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Initialize the database path
		utils.InitDBPath(app)

		// Get the absolute path of the working directory
		workDir, err := os.Getwd()
		if err != nil {
			log.Printf("Failed to get working directory: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to initialize environment",
			})
		}

		// Ensure ansible base path is absolute
		if !filepath.IsAbs(ansibleBasePath) {
			ansibleBasePath = filepath.Join(workDir, ansibleBasePath)
		}

		var scanReq models.ScanRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&scanReq); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		// Ensure ansible base path is provided
		if ansibleBasePath == "" {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Ansible base path is not configured",
			})
		}

		// Log the absolute paths being used
		log.Printf("Working Directory: %s", workDir)
		log.Printf("Ansible Base Path: %s", ansibleBasePath)

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", scanReq.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Check if the scan is already in progress
		currentStatus := record.GetString("status")
		if currentStatus == "Generating" || currentStatus == "Deploying" {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Scan is already in progress",
			})
		}

		// Get the scan profile to copy the VM size
		scanProfileID := record.Get("scan_profile").(string)
		scanProfile, err := app.Dao().FindRecordById("scan_profiles", scanProfileID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find scan profile",
			})
		}

		// Copy the VM size from the scan profile to the scan record
		vmSize := scanProfile.Get("vm_size")
		if vmSize != nil {
			record.Set("vm_size", vmSize)
		}

		// Update scan status to "Generating"
		record.Set("status", "Generating")
		record.Set("start_time", time.Now().Format(time.RFC3339))

		// Generate and set API key
		apiKey := utils.GenerateAPIKey()
		record.Set("api_key", apiKey)

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan status",
			})
		}

		// Send notification for scan start
		if err := notificationService.NotifyScanStarted(context.Background(), scanReq.ScanID, record.GetString("name")); err != nil {
			log.Printf("Failed to send scan start notification: %v", err)
		}

		// Generate YAML file for Ansible
		yamlContent, err := utils.GenerateYAMLVars(app, scanReq.ScanID)
		if err != nil {
			log.Printf("Failed to generate YAML: %v", err)

			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}

			// Send notification for scan failure
			if notifyErr := notificationService.NotifyScanFailed(context.Background(), scanReq.ScanID, record.GetString("name"), err.Error()); notifyErr != nil {
				log.Printf("Failed to send scan failure notification: %v", notifyErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to generate YAML",
			})
		}

		// Define the paths using the scanID
		scanDir := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID)
		yamlFile := filepath.Join(scanDir, "scan.yaml")
		targetsFile := filepath.Join(scanDir, "targets.json")
		profileFile := filepath.Join(scanDir, "nuclei_profile.yaml")
		logDir := filepath.Join(scanDir, "logs")
		generateYamlFile := filepath.Join(ansibleBasePath, "generate.yml")
		deployPlaybookPath := filepath.Join(scanDir, "deploy.yml")
		inventoryPath := filepath.Join(scanDir, "inventory")

		// Log the paths for debugging
		log.Printf("Ansible Base Path: %s", ansibleBasePath)
		log.Printf("Scan Directory: %s", scanDir)
		log.Printf("Generate YAML Path: %s", generateYamlFile)
		log.Printf("Deploy Playbook Path: %s", deployPlaybookPath)
		log.Printf("Inventory Path: %s", inventoryPath)

		// Create scan directory if it doesn't exist
		if err := os.MkdirAll(scanDir, 0755); err != nil {
			log.Printf("Failed to create scan directory: %v", err)

			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}

			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to create scan directory: %v", err),
			})
		}

		// Create directories and files
		if err := utils.SetupScanFiles(yamlContent, yamlFile, targetsFile, profileFile, logDir, app, scanReq.ScanID); err != nil {
			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}

			// Send notification for scan failure
			if notifyErr := notificationService.NotifyScanFailed(context.Background(), scanReq.ScanID, record.GetString("name"), err.Error()); notifyErr != nil {
				log.Printf("Failed to send scan failure notification: %v", notifyErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		// Validate the generate playbook before running it
		if err := validatePlaybook(generateYamlFile, scanReq.ScanID, app); err != nil {
			log.Printf("Failed to validate generate playbook: %v", err)
			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to validate generate playbook: %v", err),
			})
		}

		// First run generate.yml to create all necessary files
		log.Printf("Running generate.yml to create necessary files")
		if err := utils.ExecuteAnsiblePlaybook(
			generateYamlFile,
			logDir,
			yamlFile,
			inventoryPath,
			ansibleBasePath,
			app,
			scanReq.ScanID,
		); err != nil {
			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}

			// Send notification for scan failure
			if notifyErr := notificationService.NotifyScanFailed(context.Background(), scanReq.ScanID, record.GetString("name"), err.Error()); notifyErr != nil {
				log.Printf("Failed to send scan failure notification: %v", notifyErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to run generate playbook",
			})
		}

		// Now validate the deploy playbook that was just generated
		if err := validatePlaybook(deployPlaybookPath, scanReq.ScanID, app); err != nil {
			log.Printf("Failed to validate deploy playbook: %v", err)
			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to validate deploy playbook: %v", err),
			})
		}

		// Update scan status to "Deploying" using fresh record
		if err := updateScanStatus(app, scanReq.ScanID, "Deploying"); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan status",
			})
		}

		// Then run deploy.yml to start the scan
		log.Printf("Running deploy.yml to start the scan")
		if err := utils.ExecuteAnsiblePlaybook(
			deployPlaybookPath,
			logDir,
			yamlFile,
			inventoryPath,
			ansibleBasePath,
			app,
			scanReq.ScanID,
		); err != nil {
			// Update scan status to Failed using fresh record
			if updateErr := updateScanStatus(app, scanReq.ScanID, "Failed"); updateErr != nil {
				log.Printf("Failed to update scan status to Failed: %v", updateErr)
			}

			// Send notification for scan failure
			if notifyErr := notificationService.NotifyScanFailed(context.Background(), scanReq.ScanID, record.GetString("name"), err.Error()); notifyErr != nil {
				log.Printf("Failed to send scan failure notification: %v", notifyErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to run deploy playbook",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan started",
		})
	}
}

func HandleScanComplete(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			ScanID string `json:"scan_id"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		if err := updateScanStatus(app, req.ScanID, "Destroyed"); err != nil {
			log.Printf("Failed to update scan status: %v", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan removed from monitoring",
		})
	}
}
