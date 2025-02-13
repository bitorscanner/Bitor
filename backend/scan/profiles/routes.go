package profiles

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

// RegisterRoutes registers all profile-related routes
func RegisterRoutes(app *pocketbase.PocketBase, apiGroup *echo.Group) {
	profilesGroup := apiGroup.Group("/profiles")

	// Add official profiles to database
	profilesGroup.POST("/add-official", HandleAddOfficialProfiles(app))
}
