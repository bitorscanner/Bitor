package templates

import (
	"bitor/auth"       // Adjust the import path as necessary
	"bitor/middleware" // Import the middleware package for permission checks

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

	// Register read-only routes (require read permission)
	templatesGroup.GET("", ListTemplatesHandler, middleware.RequirePermission(app, "read", "templates"))
	templatesGroup.GET("/", ListTemplatesHandler, middleware.RequirePermission(app, "read", "templates"))
	templatesGroup.GET("/content", GetTemplateContentHandler, middleware.RequirePermission(app, "read", "templates"))
	templatesGroup.GET("/all", ListAllTemplatesHandler, middleware.RequirePermission(app, "read", "templates"))

	// Register write routes (require write permission)
	templatesGroup.POST("/content", SaveTemplateContentHandler, middleware.RequirePermission(app, "write", "templates"))
	templatesGroup.POST("/rename", RenameTemplateHandler, middleware.RequirePermission(app, "write", "templates"))

	// Register delete routes (require delete permission)
	templatesGroup.POST("/delete", DeleteTemplateHandler, middleware.RequirePermission(app, "delete", "templates"))
}
