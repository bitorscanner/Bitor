package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	"gopkg.in/yaml.v2"
)

func GenerateNucleiProfileYAML(app *pocketbase.PocketBase, scanID string, filePath string) error {
	log.Printf("Starting GenerateNucleiProfileYAML for scan %s", scanID)
	log.Printf("Output file path: %s", filePath)

	// Fetch the record from the "nuclei_scans" collection
	log.Printf("Fetching record from nuclei_scans collection...")
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		log.Printf("Failed to find record: %v", err)
		return fmt.Errorf("failed to find record: %w", err)
	}
	log.Printf("Found record with ID: %s", record.Id)

	// Expand the necessary relations
	relations := []string{"nuclei_profile"}
	log.Printf("Expanding relations: %v", relations)
	expandedData, err := ExpandRelations(app, record, relations)
	if err != nil {
		log.Printf("Failed to expand relations: %v", err)
		return err
	}
	log.Printf("Successfully expanded relations")

	// Access the expanded "profile" record
	expandedProfile, ok := expandedData["nuclei_profile"]
	if !ok {
		log.Printf("Profile relation not found")
		return fmt.Errorf("profile relation not found")
	}
	log.Printf("Found expanded profile")

	// Extract profile JSON blob
	var profileJSON types.JsonRaw
	switch profile := expandedProfile.(type) {
	case *models.Record:
		log.Printf("Profile is a single record with ID: %s", profile.Id)
		rawProfile := profile.Get("profile")
		if rawProfile == nil {
			log.Printf("Profile data is nil")
			return fmt.Errorf("profile data is nil")
		}
		profileJSON, ok = rawProfile.(types.JsonRaw)
		if !ok {
			log.Printf("Profile data is not JSON: %T", rawProfile)
			return fmt.Errorf("profile data is not JSON")
		}
		if len(profileJSON) == 0 {
			log.Printf("Profile JSON is empty")
			return fmt.Errorf("profile JSON is empty")
		}
	case []*models.Record:
		log.Printf("Profile is a record array with length %d", len(profile))
		if len(profile) > 0 {
			rawProfile := profile[0].Get("profile")
			if rawProfile == nil {
				log.Printf("Profile data is nil")
				return fmt.Errorf("profile data is nil")
			}
			profileJSON, ok = rawProfile.(types.JsonRaw)
			if !ok {
				log.Printf("Profile data is not JSON: %T", rawProfile)
				return fmt.Errorf("profile data is not JSON")
			}
			if len(profileJSON) == 0 {
				log.Printf("Profile JSON is empty")
				return fmt.Errorf("profile JSON is empty")
			}
		} else {
			log.Printf("Profile relation is empty")
			return fmt.Errorf("profile relation is empty")
		}
	default:
		log.Printf("Unexpected profile type: %T", expandedProfile)
		return fmt.Errorf("unexpected type for profile relation")
	}
	log.Printf("Raw profile data type: %T", profileJSON)
	log.Printf("Extracted profile JSON length: %d", len(profileJSON))
	log.Printf("Extracted profile JSON: %s", string(profileJSON))

	// Convert JSON to YAML
	log.Printf("Converting JSON to YAML...")
	yamlData, err := ConvertJSONToYAML(profileJSON)
	if err != nil {
		log.Printf("Failed to convert JSON to YAML: %v", err)
		return fmt.Errorf("failed to convert JSON to YAML: %w", err)
	}
	log.Printf("Successfully converted to YAML")

	// Save profile YAML to the specified file path
	log.Printf("Writing YAML to file: %s", filePath)
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		log.Printf("Failed to write nuclei_profile.yaml: %v", err)
		return fmt.Errorf("failed to write nuclei_profile.yaml: %w", err)
	}
	log.Printf("Successfully wrote YAML file")

	log.Printf("Successfully completed GenerateNucleiProfileYAML for scan %s", scanID)
	return nil
}

func ConvertJSONToYAML(jsonData []byte) ([]byte, error) {
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return nil, err
	}
	return yaml.Marshal(jsonObj)
}
