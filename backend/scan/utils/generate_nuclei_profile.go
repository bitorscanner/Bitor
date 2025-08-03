package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	yaml "gopkg.in/yaml.v2"
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

	// Extract profile data - check both 'profile' (JSON) and 'raw_yaml' (YAML) fields
	var profileData []byte
	var isYamlData bool

	switch profile := expandedProfile.(type) {
	case *models.Record:
		log.Printf("Profile is a single record with ID: %s", profile.Id)

		// First try the 'profile' field (for custom profiles - JSON format)
		rawProfile := profile.Get("profile")
		if rawProfile != nil {
			if jsonData, ok := rawProfile.(types.JsonRaw); ok && len(jsonData) > 0 {
				profileData = []byte(jsonData)
				isYamlData = false
				log.Printf("Using profile field data (JSON)")
			}
		}

		// If profile field is empty, try 'raw_yaml' field (for default profiles - YAML format)
		if len(profileData) == 0 {
			rawYaml := profile.Get("raw_yaml")
			if rawYaml != nil {
				if yamlData, ok := rawYaml.(types.JsonRaw); ok && len(yamlData) > 0 {
					// The raw_yaml field contains JSON-encoded YAML content, so we need to decode it
					var yamlContent string
					if err := json.Unmarshal(yamlData, &yamlContent); err != nil {
						log.Printf("Failed to decode JSON-encoded YAML: %v", err)
						// Fallback: treat as raw bytes
						profileData = []byte(yamlData)
					} else {
						profileData = []byte(yamlContent)
						log.Printf("Successfully decoded JSON-encoded YAML content")
					}
					isYamlData = true
					log.Printf("Using raw_yaml field data (YAML)")
				} else if yamlStr, ok := rawYaml.(string); ok && len(yamlStr) > 0 {
					profileData = []byte(yamlStr)
					isYamlData = true
					log.Printf("Using raw_yaml string data (YAML)")
				}
			}
		}

		// Final validation
		if len(profileData) == 0 {
			log.Printf("Both profile and raw_yaml fields are empty")
			return fmt.Errorf("both profile and raw_yaml fields are empty")
		}
	case []*models.Record:
		log.Printf("Profile is a record array with length %d", len(profile))
		if len(profile) > 0 {
			// First try the 'profile' field (for custom profiles - JSON format)
			rawProfile := profile[0].Get("profile")
			if rawProfile != nil {
				if jsonData, ok := rawProfile.(types.JsonRaw); ok && len(jsonData) > 0 {
					profileData = []byte(jsonData)
					isYamlData = false
					log.Printf("Using profile field data from array (JSON)")
				}
			}

			// If profile field is empty, try 'raw_yaml' field (for default profiles - YAML format)
			if len(profileData) == 0 {
				rawYaml := profile[0].Get("raw_yaml")
				if rawYaml != nil {
					if yamlData, ok := rawYaml.(types.JsonRaw); ok && len(yamlData) > 0 {
						// The raw_yaml field contains JSON-encoded YAML content, so we need to decode it
						var yamlContent string
						if err := json.Unmarshal(yamlData, &yamlContent); err != nil {
							log.Printf("Failed to decode JSON-encoded YAML from array: %v", err)
							// Fallback: treat as raw bytes
							profileData = []byte(yamlData)
						} else {
							profileData = []byte(yamlContent)
							log.Printf("Successfully decoded JSON-encoded YAML content from array")
						}
						isYamlData = true
						log.Printf("Using raw_yaml field data from array (YAML)")
					} else if yamlStr, ok := rawYaml.(string); ok && len(yamlStr) > 0 {
						profileData = []byte(yamlStr)
						isYamlData = true
						log.Printf("Using raw_yaml string data from array (YAML)")
					}
				}
			}

			// Final validation
			if len(profileData) == 0 {
				log.Printf("Both profile and raw_yaml fields are empty in array")
				return fmt.Errorf("both profile and raw_yaml fields are empty")
			}
		} else {
			log.Printf("Profile relation is empty")
			return fmt.Errorf("profile relation is empty")
		}
	default:
		log.Printf("Unexpected profile type: %T", expandedProfile)
		return fmt.Errorf("unexpected type for profile relation")
	}
	log.Printf("Extracted profile data length: %d", len(profileData))
	log.Printf("Profile data is YAML format: %v", isYamlData)

	// Process the data based on its format
	var finalYamlData []byte
	if isYamlData {
		// Data is already in YAML format, use directly
		log.Printf("Using raw YAML data directly")
		finalYamlData = profileData
	} else {
		// Data is in JSON format, convert to YAML
		log.Printf("Converting JSON to YAML...")
		yamlData, err := ConvertJSONToYAML(profileData)
		if err != nil {
			log.Printf("Failed to convert JSON to YAML: %v", err)
			return fmt.Errorf("failed to convert JSON to YAML: %w", err)
		}
		finalYamlData = yamlData
		log.Printf("Successfully converted JSON to YAML")
	}

	// Save profile YAML to the specified file path
	log.Printf("Writing YAML to file: %s", filePath)
	if err := os.WriteFile(filePath, finalYamlData, 0644); err != nil {
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
