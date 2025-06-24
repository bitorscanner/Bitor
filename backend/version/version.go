package version

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Version is set during build
var Version string = "development"

// IsDocker indicates if the application is running in a Docker container
var IsDocker bool

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func init() {
	// Check if running in Docker
	_, err := os.Stat("/.dockerenv")
	IsDocker = err == nil
}

// RegisterRoutes registers all version-related routes
func RegisterRoutes(e *core.ServeEvent) {
	e.Router.GET("/api/version/check", handleVersionCheck)
	e.Router.POST("/api/version/update", handleUpdate, apis.RequireAdminAuth())
}

func handleVersionCheck(c echo.Context) error {
	currentVersion := Version
	if currentVersion == "development" {
		currentVersion = "v0.0.0"
	}

	// Get latest release from GitHub
	latestRelease, err := getLatestRelease()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to check latest version", err)
	}

	// Parse versions
	current, err := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Invalid current version", err)
	}

	latest, err := semver.NewVersion(strings.TrimPrefix(latestRelease.TagName, "v"))
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Invalid latest version", err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_version":  currentVersion,
		"latest_version":   latestRelease.TagName,
		"update_available": latest.GreaterThan(current),
		"is_docker":        IsDocker,
		"release_notes":    latestRelease.Body,
		"download_url":     getDownloadURL(latestRelease),
	})
}

func handleUpdate(c echo.Context) error {
	if IsDocker {
		return apis.NewApiError(http.StatusBadRequest, "Updates must be performed by pulling a new Docker image", nil)
	}

	// Get latest release
	latestRelease, err := getLatestRelease()
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to get latest version", err)
	}

	// Download new version
	downloadURL := getDownloadURL(latestRelease)
	if downloadURL == "" {
		return apis.NewApiError(http.StatusInternalServerError, "No suitable download found for your platform", nil)
	}

	// Start update process
	go func() {
		if err := performUpdate(downloadURL); err != nil {
			log.Printf("Update failed: %v", err)
		}
	}()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Update process started",
	})
}

func getLatestRelease() (*GithubRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/bitorscanner/bitor/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release GithubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func getDownloadURL(release *GithubRelease) string {
	// Determine platform-specific binary name
	platform := fmt.Sprintf("%s_%s", os.Getenv("GOOS"), os.Getenv("GOARCH"))

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, platform) {
			return asset.URL
		}
	}

	return ""
}

func performUpdate(downloadURL string) error {
	// Download new binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "bitor-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy download to temporary file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write update: %v", err)
	}

	// Make temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make update executable: %v", err)
	}

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Replace current executable with new version
	if err := os.Rename(tmpFile.Name(), executable); err != nil {
		return fmt.Errorf("failed to replace executable: %v", err)
	}

	// Restart application
	// Note: This will be handled by the process manager (systemd, etc.)
	os.Exit(0)
	return nil
}
