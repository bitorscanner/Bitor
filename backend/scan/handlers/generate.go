package handlers

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"

	"bitor/models"
	"bitor/scan/utils"
)

// HandleGenerateScan handles the generation of scan code.

func HandleGenerateScan(app *pocketbase.PocketBase, ansibleBasePath string) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Println("Received request to /api/scan/generate")

		// Bind the request payload
		var scanReq models.ScanRequest
		if err := c.Bind(&scanReq); err != nil {
			log.Println("Error binding request:", err)
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

		// Update scan status to "Running"
		record, err := app.Dao().FindRecordById("nuclei_scans", scanReq.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		record.Set("status", "Generating")
		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan status",
			})
		}

		// Generate YAML file for Ansible
		yamlContent, err := utils.GenerateYAMLVars(app, scanReq.ScanID)
		if err != nil {
			log.Printf("Failed to generate YAML: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to generate YAML",
			})
		}

		// Define the paths using the scanID
		yamlFile := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "scan.yaml")
		targetsFile := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "targets.json")
		profileFile := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "nuclei_profile.yaml")
		logDir := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "logs")
		generateYamlFile := filepath.Join(ansibleBasePath, "generate.yml")
		inventoryPath := filepath.Join(ansibleBasePath, "scans", scanReq.ScanID, "inventory")
		// Create directories and files
		if err := utils.SetupScanFiles(yamlContent, yamlFile, targetsFile, profileFile, logDir, app, scanReq.ScanID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		// Execute Ansible playbook
		if err := utils.ExecuteAnsiblePlaybook(
			generateYamlFile,
			logDir,
			yamlFile,
			inventoryPath,
			ansibleBasePath,
			app,
			scanReq.ScanID,
		); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to run Ansible playbook",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan Code Generated",
		})
	}
}
