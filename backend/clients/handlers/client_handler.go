package handlers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"orbit/auth"
	"orbit/clients/services"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type ClientHandler struct {
	app            *pocketbase.PocketBase
	faviconFetcher *services.FaviconFetcher
}

func NewClientHandler(app *pocketbase.PocketBase) *ClientHandler {
	return &ClientHandler{
		app:            app,
		faviconFetcher: services.NewFaviconFetcher(),
	}
}

// RegisterRoutes registers all client-related routes
func (h *ClientHandler) RegisterRoutes(e *core.ServeEvent) {
	// Create a group for the client routes with authentication middleware
	clientGroup := e.Router.Group("/api/clients",
		apis.LoadAuthContext(h.app),     // Apply LoadAuthContext middleware first
		auth.RequireAuthOrAPIKey(h.app), // Use the custom middleware from the auth package
		apis.ActivityLogger(h.app),      // Optional: log activities
	)

	// Register favicon endpoints
	clientGroup.GET("/favicon", h.handleFaviconProxy)
	clientGroup.POST("/:id/fetch-favicon", h.handleFetchFavicon)
}

// handleFaviconProxy handles proxying favicon requests to avoid CORS issues
func (h *ClientHandler) handleFaviconProxy(c echo.Context) error {
	domain := c.QueryParam("domain")
	if domain == "" {
		return c.String(http.StatusBadRequest, "domain parameter is required")
	}

	// Construct Google Favicon URL
	googleFaviconUrl := "https://www.google.com/s2/favicons?domain=" + url.QueryEscape(domain) + "&sz=64"

	// Create HTTP client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("GET", googleFaviconUrl, nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error creating request")
	}

	// Add headers
	req.Header.Add("Accept", "image/png,image/*")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error fetching favicon")
	}
	defer resp.Body.Close()

	// Set response headers
	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours

	// Stream the response
	_, err = io.Copy(c.Response().Writer, resp.Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error streaming response")
	}

	return nil
}

// storeFaviconFile is a helper function to store the favicon as a file instead of as a data URI
func storeFaviconFile(app *pocketbase.PocketBase, record *models.Record, field string, data []byte, contentType string) error {
	// Check if data is empty
	if len(data) == 0 {
		return fmt.Errorf("no favicon data provided")
	}

	log.Printf("Storing favicon: data length = %d bytes, contentType = %s", len(data), contentType)

	// Generate a short hash-based file name
	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum[:8])
	// Determine file extension based on contentType
	ext := "png"
	if strings.Contains(contentType, "jpeg") {
		ext = "jpg"
	} else if strings.Contains(contentType, "svg") {
		ext = "svg"
	}
	fileName := fmt.Sprintf("%s.%s", hash, ext)

	// Save the file to the correct directory inside 'storage' using collection ID
	storageDir := filepath.Join(app.DataDir(), "storage", record.Collection().Id, record.Id)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory %s: %w", storageDir, err)
	}
	filePath := filepath.Join(storageDir, fileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	// Verify the file now exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file %s was not created: %w", filePath, err)
	}

	// Log the successfully saved file path for debugging
	log.Printf("Favicon file saved successfully at: %s", filePath)

	// Set the field to the saved file name
	record.Set(field, fileName)
	return nil
}

// handleFetchFavicon handles fetching and storing favicons for a specific client
func (h *ClientHandler) handleFetchFavicon(c echo.Context) error {
	clientID := c.PathParam("id")
	if clientID == "" {
		return apis.NewBadRequestError("missing client id", nil)
	}

	// Get the client record
	client, err := h.app.Dao().FindRecordById("clients", clientID)
	if err != nil {
		return apis.NewNotFoundError("client not found", err)
	}

	// Get the homepage URL
	homepage := client.GetString("homepage")
	if homepage == "" {
		return apis.NewBadRequestError("client has no homepage URL", nil)
	}

	// Clean up the homepage URL
	homepage = strings.TrimSpace(homepage)
	if !strings.HasPrefix(homepage, "http") {
		// Try with https first
		homepage = "https://" + homepage
	}

	log.Printf("Attempting to fetch favicon for homepage: %s", homepage)

	// Try to fetch the favicon
	faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage)
	if err != nil {
		// If the first attempt failed and we used https, try http
		if strings.HasPrefix(homepage, "https://") {
			httpHomepage := "http://" + strings.TrimPrefix(homepage, "https://")
			log.Printf("HTTPS attempt failed, trying HTTP: %s", httpHomepage)
			faviconData, contentType, err = h.faviconFetcher.FetchFavicon(httpHomepage)
		}

		// If still failed, try without www if it was present
		if err != nil && strings.HasPrefix(strings.ToLower(homepage), "https://www.") {
			noWwwHomepage := "https://" + strings.TrimPrefix(strings.ToLower(homepage), "https://www.")
			log.Printf("Trying without www: %s", noWwwHomepage)
			faviconData, contentType, err = h.faviconFetcher.FetchFavicon(noWwwHomepage)
		}

		// If all attempts failed, try Google's favicon service
		if err != nil {
			domain := homepage
			if strings.HasPrefix(domain, "http://") {
				domain = strings.TrimPrefix(domain, "http://")
			} else if strings.HasPrefix(domain, "https://") {
				domain = strings.TrimPrefix(domain, "https://")
			}
			if strings.HasPrefix(domain, "www.") {
				domain = strings.TrimPrefix(domain, "www.")
			}
			domain = strings.Split(domain, "/")[0]

			googleFaviconURL := fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", domain)
			log.Printf("Trying Google favicon service: %s", googleFaviconURL)
			faviconData, contentType, err = h.faviconFetcher.FetchFavicon(googleFaviconURL)
		}

		// If all attempts failed, return the error
		if err != nil {
			log.Printf("All favicon fetch attempts failed for %s: %v", homepage, err)
			return apis.NewBadRequestError(fmt.Sprintf("failed to fetch favicon: %v", err), nil)
		}
	}

	log.Printf("Successfully fetched favicon for %s with content type: %s", homepage, contentType)

	// Instead of storing a data URI, store the actual file
	err = storeFaviconFile(h.app, client, "favicon", faviconData, contentType)
	if err != nil {
		log.Printf("Failed to save client favicon file: %v", err)
		return apis.NewBadRequestError("failed to save client record", err)
	}

	if err := h.app.Dao().SaveRecord(client); err != nil {
		log.Printf("Failed to save client record with favicon: %v", err)
		return apis.NewBadRequestError("failed to save client record", err)
	}

	log.Printf("Successfully saved favicon for client %s", clientID)

	// Return success response with the favicon file name
	return c.JSON(http.StatusOK, map[string]string{
		"favicon": client.GetString("favicon"),
	})
}

// OnModelBeforeCreate hook for automatically fetching favicons on client creation
func (h *ClientHandler) OnModelBeforeCreate(e *core.ModelEvent) error {
	record, ok := e.Model.(*models.Record)
	if !ok || record.Collection().Name != "clients" {
		return nil
	}

	// Try to fetch favicon if homepage is provided
	homepage := record.GetString("homepage")
	if homepage != "" {
		if faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage); err == nil {
			err := storeFaviconFile(h.app, record, "favicon", faviconData, contentType)
			if err != nil {
				log.Printf("Failed to store favicon file for %s: %v", homepage, err)
			}
		} else {
			log.Printf("Failed to fetch favicon for %s: %v", homepage, err)
		}
	}

	return nil
}

// OnModelBeforeUpdate hook for automatically updating favicons when homepage changes
func (h *ClientHandler) OnModelBeforeUpdate(e *core.ModelEvent) error {
	record, ok := e.Model.(*models.Record)
	if !ok || record.Collection().Name != "clients" {
		return nil
	}

	// Check if homepage field was changed
	homepage := record.GetString("homepage")
	if homepage != "" {
		if faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage); err == nil {
			err := storeFaviconFile(h.app, record, "favicon", faviconData, contentType)
			if err != nil {
				log.Printf("Failed to store favicon file for %s: %v", homepage, err)
			}
		} else {
			log.Printf("Failed to fetch favicon for %s: %v", homepage, err)
		}
	}

	return nil
}
