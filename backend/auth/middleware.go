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
			log.Printf("[AUTH] Processing request: %s %s", c.Request().Method, c.Request().URL.Path)
			log.Printf("[AUTH] Authorization header present: %t", authHeader != "")

			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			if admin != nil {
				log.Printf("[AUTH] Admin already authenticated: %s", admin.Id)
				return next(c)
			}
			if record != nil {
				log.Printf("[AUTH] User already authenticated: %s", record.Id)
				return next(c)
			}

			// Check for Bearer token
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				log.Printf("[AUTH] Bearer token found, length: %d", len(token))

				// Try to find admin by token
				admin, err := app.Dao().FindAdminByToken(token, app.Settings().RecordAuthToken.Secret)
				if err == nil && admin != nil {
					log.Printf("[AUTH] Admin authenticated via token: %s", admin.Id)
					c.Set(apis.ContextAdminKey, admin)
					return next(c)
				} else if err != nil {
					log.Printf("[AUTH] Admin token lookup failed: %v", err)
				}

				// Try to find user by token
				record, err := app.Dao().FindAuthRecordByToken(token, app.Settings().RecordAuthToken.Secret)
				if err == nil && record != nil {
					log.Printf("[AUTH] User authenticated via token: %s", record.Id)
					c.Set(apis.ContextAuthRecordKey, record)
					return next(c)
				} else if err != nil {
					log.Printf("[AUTH] User token lookup failed: %v", err)
				}
			}

			// Check for API key authentication
			var scanID string

			// First try to get scan_id from form value (for multipart/form-data)
			scanID = c.FormValue("scan_id")
			log.Printf("[AUTH] Form scan_id: %s", scanID)
			if scanID == "" {
				// If not found in form, try to get from JSON body
				var req ScanRequest
				if err := c.Bind(&req); err == nil {
					scanID = req.ScanID
					log.Printf("[AUTH] JSON scan_id: %s", scanID)
				} else {
					log.Printf("[AUTH] Failed to bind JSON body: %v", err)
				}
			}

			if scanID != "" {
				log.Printf("[AUTH] Attempting API key authentication for scan: %s", scanID)
				scan, err := app.Dao().FindRecordById("nuclei_scans", scanID)
				if err == nil && scan != nil {
					// Compare the token to the record's api_key
					apiKey := scan.GetString("api_key")
					providedToken := strings.TrimPrefix(authHeader, "Bearer ")
					log.Printf("[AUTH] Stored API key length: %d, Provided token length: %d", len(apiKey), len(providedToken))
					if apiKey != "" && apiKey == providedToken {
						log.Printf("[AUTH] API key authentication successful for scan: %s", scanID)
						return next(c)
					} else {
						log.Printf("[AUTH] API key mismatch for scan: %s", scanID)
					}
				} else {
					log.Printf("[AUTH] Failed to find scan record: %v", err)
				}
			} else {
				log.Printf("[AUTH] No scan_id found in request")
			}

			log.Printf("[AUTH] Authentication failed - no valid credentials found")
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
