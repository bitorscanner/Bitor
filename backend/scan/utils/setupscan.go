package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
)

// Helper function to set up scan files.
func SetupScanFiles(yamlContent, yamlFile, targetsFile, profileFile, logDir string, app *pocketbase.PocketBase, scanID string) error {
	log.Printf("Starting SetupScanFiles for scan %s", scanID)
	log.Printf("YAML file: %s", yamlFile)
	log.Printf("Targets file: %s", targetsFile)
	log.Printf("Profile file: %s", profileFile)
	log.Printf("Log directory: %s", logDir)

	// Ensure the log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	log.Printf("Created log directory: %s", logDir)

	// Ensure the directory for yamlFile exists
	yamlDir := filepath.Dir(yamlFile)
	if err := os.MkdirAll(yamlDir, 0755); err != nil {
		log.Printf("Failed to create directory for YAML file: %v", err)
		return fmt.Errorf("failed to create directory for YAML file: %v", err)
	}
	log.Printf("Created YAML directory: %s", yamlDir)

	// Ensure the directory for targetsFile exists
	targetsDir := filepath.Dir(targetsFile)
	if err := os.MkdirAll(targetsDir, 0755); err != nil {
		log.Printf("Failed to create directory for targets file: %v", err)
		return fmt.Errorf("failed to create directory for targets file: %v", err)
	}
	log.Printf("Created targets directory: %s", targetsDir)

	// Ensure the directory for profileFile exists
	profileDir := filepath.Dir(profileFile)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		log.Printf("Failed to create directory for profile file: %v", err)
		return fmt.Errorf("failed to create directory for profile file: %v", err)
	}
	log.Printf("Created profile directory: %s", profileDir)

	// Ensure the inventory directory exists
	inventoryDir := filepath.Join(filepath.Dir(yamlFile), "inventory")
	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		log.Printf("Failed to create inventory directory: %v", err)
		return fmt.Errorf("failed to create inventory directory: %v", err)
	}
	log.Printf("Created inventory directory: %s", inventoryDir)

	// Save YAML to a file
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		log.Printf("Failed to write YAML file: %v", err)
		return fmt.Errorf("failed to write YAML file: %v", err)
	}
	log.Printf("Wrote YAML file: %s", yamlFile)

	// Save targets JSON to a file
	if err := GenerateTargetsJSON(app, scanID, targetsFile); err != nil {
		log.Printf("Failed to generate targets JSON: %v", err)
		return fmt.Errorf("failed to generate targets JSON: %v", err)
	}
	log.Printf("Generated targets JSON: %s", targetsFile)

	// Save nuclei profile YAML to a file
	log.Printf("Attempting to generate nuclei profile YAML...")
	if err := GenerateNucleiProfileYAML(app, scanID, profileFile); err != nil {
		log.Printf("Failed to generate nuclei profile YAML: %v", err)
		return fmt.Errorf("failed to generate nuclei profile YAML: %v", err)
	}
	log.Printf("Generated nuclei profile YAML: %s", profileFile)

	log.Printf("Successfully completed SetupScanFiles for scan %s", scanID)
	return nil
}
