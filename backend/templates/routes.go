package templates

import (
	"bitor/auth" // Adjust the import path as necessary

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Ensure you have imported Echo's middleware package

// RegisterRoutes registers the scan routes with the authentication middleware.
func RegisterRoutes(app *pocketbase.PocketBase, e *core.ServeEvent) {
	// Create a group for the templates routes with the authentication middleware
	templatesGroup := e.Router.Group("/api/templates",
		apis.LoadAuthContext(app),     // Apply LoadAuthContext middleware first
		auth.RequireAuthOrAPIKey(app), // Use the custom middleware from the auth package
		apis.ActivityLogger(app),      // Optional: log activities
	)

	// Register templates routes within the group
	templatesGroup.GET("", ListTemplatesHandler)  // Matches /api/templates
	templatesGroup.GET("/", ListTemplatesHandler) // Matches /api/templates/
	templatesGroup.GET("/content", GetTemplateContentHandler)
	templatesGroup.POST("/content", SaveTemplateContentHandler)
	templatesGroup.GET("/all", ListAllTemplatesHandler)   // Matches /api/templates/all - Used by template file browser
	templatesGroup.POST("/rename", RenameTemplateHandler) // Matches /api/templates/rename
	templatesGroup.POST("/delete", DeleteTemplateHandler) // Matches /api/templates/delete
}
