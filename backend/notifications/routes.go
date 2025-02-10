package notifications

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func RegisterRoutes(app *pocketbase.PocketBase, g *echo.Group) {
	// Apply email settings handler
	g.POST("/api/apply-email-settings", func(c echo.Context) error {
		if err := ApplyEmailSettings(app); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to apply email settings", err)
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Email settings applied successfully",
		})
	}, apis.RequireAdminOrRecordAuth())

	testEmailHandler := func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)

		if admin == nil && (authRecord == nil || authRecord.Collection().Name != "users") {
			return apis.NewUnauthorizedError("Only authenticated users can send test emails", nil)
		}

		var email string
		// Try to get email from query parameter first (GET request)
		email = c.QueryParam("email")
		if email == "" {
			// If not in query, try to get from JSON body (POST request)
			var data struct {
				Email string `json:"email"`
			}
			if err := c.Bind(&data); err == nil {
				email = data.Email
			}
		}

		if email == "" {
			return apis.NewBadRequestError("Email address is required", nil)
		}

		// Apply email settings before sending test email
		if err := ApplyEmailSettings(app); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to apply email settings", err)
		}

		if err := SendTestEmail(app, email); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to send test email", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Test email sent successfully",
		})
	}

	// Register both GET and POST handlers
	g.GET("/api/test-email", testEmailHandler, apis.RequireAdminOrRecordAuth())
	g.POST("/api/test-email", testEmailHandler, apis.RequireAdminOrRecordAuth())
}
