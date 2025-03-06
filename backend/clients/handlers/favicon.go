package handlers

import (
	"io"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

// HandleFaviconProxy handles proxying favicon requests to avoid CORS issues
func HandleFaviconProxy(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
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
}
