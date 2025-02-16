package templates

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

// RegisterRoutes registers all template-related routes
func RegisterRoutes(app *pocketbase.PocketBase, apiGroup *echo.Group) {
	templatesGroup := apiGroup.Group("/templates")

	// List all templates
	templatesGroup.GET("/list", HandleListTemplates())
}
