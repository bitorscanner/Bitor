// File: backend/scan/profiles/profiles.go
package profiles

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

const (
	profilesBaseURL = "https://raw.githubusercontent.com/projectdiscovery/nuclei-templates/main/profiles"
	profilesDir     = "nuclei-profiles"
)

var defaultProfiles = []string{
	"alibaba-cloud-config.yml",
	"aws-cloud-config.yml",
	"azure-cloud-config.yml",
	"cloud.yml",
	"compliance.yml",
	"cves.yml",
	"default-login.yml",
	"k8s-cluster-security.yml",
	"kev.yml",
	"misconfigurations.yml",
	"osint.yml",
	"pentest.yml",
	"privilege-escalation.yml",
	"recommended.yml",
	"subdomain-takeovers.yml",
	"windows-audit.yml",
	"wordpress.yml",
}

type ProfileManager struct {
	baseDir     string
	lastUpdate  time.Time
	updateMutex sync.Mutex
}

func NewProfileManager(baseDir string) *ProfileManager {
	return &ProfileManager{
		baseDir: baseDir,
	}
}

// EnsureProfilesExist checks if profiles exist and downloads them if necessary
func (pm *ProfileManager) EnsureProfilesExist() error {
	pm.updateMutex.Lock()
	defer pm.updateMutex.Unlock()

	// Create profiles directory if it doesn't exist
	profilePath := filepath.Join(pm.baseDir, profilesDir)
	if err := os.MkdirAll(profilePath, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %v", err)
	}

	// Check if we need to update (once per day)
	if !pm.shouldUpdate() {
		return nil
	}

	// Download all profiles
	for _, profile := range defaultProfiles {
		if err := pm.downloadProfile(profile); err != nil {
			return fmt.Errorf("failed to download profile %s: %v", profile, err)
		}
	}

	pm.lastUpdate = time.Now()
	return nil
}

func (pm *ProfileManager) shouldUpdate() bool {
	return time.Since(pm.lastUpdate) > 24*time.Hour
}

func (pm *ProfileManager) downloadProfile(profileName string) error {
	url := fmt.Sprintf("%s/%s", profilesBaseURL, profileName)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download profile %s: %s", profileName, resp.Status)
	}

	filePath := filepath.Join(pm.baseDir, profilesDir, profileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// GetProfilePath returns the full path to a profile
func (pm *ProfileManager) GetProfilePath(profileName string) (string, error) {
	// Sanitize profile name
	profileName = strings.TrimSpace(profileName)
	if !strings.HasSuffix(profileName, ".yml") {
		profileName += ".yml"
	}

	// Validate profile name
	valid := false
	for _, p := range defaultProfiles {
		if p == profileName {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("invalid profile name: %s", profileName)
	}

	return filepath.Join(pm.baseDir, profilesDir, profileName), nil
}

// ListProfiles returns a list of available profiles
func (pm *ProfileManager) ListProfiles() []string {
	return defaultProfiles
}

// GetProfileDescription returns a human-readable description of a profile
func (pm *ProfileManager) GetProfileDescription(profileName string) string {
	descriptions := map[string]string{
		"alibaba-cloud-config.yml": "Security checks for Alibaba Cloud configurations",
		"aws-cloud-config.yml":     "Security checks for AWS cloud configurations",
		"azure-cloud-config.yml":   "Security checks for Azure cloud configurations",
		"cloud.yml":                "General cloud security checks",
		"compliance.yml":           "Compliance-related security checks",
		"cves.yml":                 "Common Vulnerabilities and Exposures checks",
		"default-login.yml":        "Default credential checks",
		"k8s-cluster-security.yml": "Kubernetes cluster security checks",
		"kev.yml":                  "Known Exploited Vulnerabilities checks",
		"misconfigurations.yml":    "Common security misconfigurations",
		"osint.yml":                "Open Source Intelligence gathering",
		"pentest.yml":              "Penetration testing checks",
		"privilege-escalation.yml": "Privilege escalation vulnerabilities",
		"recommended.yml":          "Recommended security checks for general use",
		"subdomain-takeovers.yml":  "Subdomain takeover vulnerability checks",
		"windows-audit.yml":        "Windows security auditing checks",
		"wordpress.yml":            "WordPress security checks",
	}

	if desc, ok := descriptions[profileName]; ok {
		return desc
	}
	return "No description available"
}

// AddProfilesToDatabase adds the official Nuclei profiles to the database
func (pm *ProfileManager) AddProfilesToDatabase(app *pocketbase.PocketBase) error {
	// Ensure profiles exist locally
	if err := pm.EnsureProfilesExist(); err != nil {
		return fmt.Errorf("failed to ensure profiles exist: %v", err)
	}

	// Get the nuclei_profiles collection
	collection, err := app.Dao().FindCollectionByNameOrId("nuclei_profiles")
	if err != nil {
		return fmt.Errorf("failed to find nuclei_profiles collection: %v", err)
	}

	for _, profileName := range defaultProfiles {
		// Check if profile already exists
		existingProfile, _ := app.Dao().FindFirstRecordByData(collection.Id, "name", strings.TrimSuffix(profileName, ".yml"))
		if existingProfile != nil {
			continue // Skip if profile already exists
		}

		// Read the profile file
		profilePath := filepath.Join(pm.baseDir, profilesDir, profileName)
		yamlContent, err := os.ReadFile(profilePath)
		if err != nil {
			return fmt.Errorf("failed to read profile %s: %v", profileName, err)
		}

		// Create a new record
		record := models.NewRecord(collection)
		record.Set("name", strings.TrimSuffix(profileName, ".yml"))
		record.Set("raw_yaml", string(yamlContent))
		record.Set("description", pm.GetProfileDescription(profileName))

		// Save the record
		if err := app.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save profile %s: %v", profileName, err)
		}
	}

	return nil
}
