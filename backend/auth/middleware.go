package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

type ScanRequest struct {
	ScanID string `json:"scan_id"`
}

func RequireAuthOrAPIKey(app *pocketbase.PocketBase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the authorization header
			authHeader := c.Request().Header.Get("Authorization")

			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			if admin != nil {
				return next(c)
			}
			if record != nil {
				return next(c)
			}

			// Check for Bearer token
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")

				// Try to find admin by token
				admin, err := app.Dao().FindAdminByToken(token, app.Settings().RecordAuthToken.Secret)
				if err == nil && admin != nil {
					c.Set(apis.ContextAdminKey, admin)
					return next(c)
				}

				// Try to find user by token
				record, err := app.Dao().FindAuthRecordByToken(token, app.Settings().RecordAuthToken.Secret)
				if err == nil && record != nil {
					c.Set(apis.ContextAuthRecordKey, record)
					return next(c)
				}
			}

			// Check for API key authentication
			var scanID string

			// First try to get scan_id from form value (for multipart/form-data)
			scanID = c.FormValue("scan_id")
			if scanID == "" {
				// If not found in form, try to get from JSON body
				var req ScanRequest
				if err := c.Bind(&req); err == nil {
					scanID = req.ScanID
				}
			}

			if scanID != "" {
				scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
				if err == nil && scan != nil {
					// Compare the token to the record's api_key
					apiKey := scan.GetString("api_key")
					if apiKey != "" && apiKey == strings.TrimPrefix(authHeader, "Bearer ") {
						return next(c)
					}
				}
			}

			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"message": "Authentication required",
			})
		}
	}
}

// isValidAPIKey is a helper function to validate a scan's API key.
// Currently unused but kept for future use in API key validation.
// It compares the provided API key against the one stored in the scan record.
//
// Note: This function is intentionally unused in the current implementation
// but is retained for planned future authentication enhancements.
// nolint:unused
func isValidAPIKey(app *pocketbase.PocketBase, scanID, apiKey string) bool {
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		log.Printf("Error finding record: %v", err)
		return false
	}
	storedApiKey := record.GetString("api_key")
	log.Printf("Comparing API keys: provided=%s, stored=%s", apiKey, storedApiKey)
	return storedApiKey == apiKey
}
