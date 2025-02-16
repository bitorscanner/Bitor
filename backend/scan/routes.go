package scan

import (
	"log"
	"orbit/auth"
	"orbit/scan/handlers"
	"orbit/services/notification"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers the scan routes with the authentication middleware.
func RegisterRoutes(app *pocketbase.PocketBase, e *core.ServeEvent, ansibleBasePath string, notificationService *notification.NotificationService) {
	log.Printf("Registering scan routes with ansible base path: %s", ansibleBasePath)

	// Initialize handlers with required services
	InitHandlers(app, ansibleBasePath, notificationService)

	// Create a group for the scan routes with the authentication middleware
	scanGroup := e.Router.Group("/api/scan",
		apis.LoadAuthContext(app),     // Apply LoadAuthContext middleware first
		auth.RequireAuthOrAPIKey(app), // Use the custom middleware from the auth package
		apis.ActivityLogger(app),      // Optional: log activities
	)

	// Register test notification endpoint
	scanGroup.POST("/test-notification", HandleTestNotification)

	// Register scan routes
	scanGroup.POST("/start", handlers.HandleStartAndGenerateScan(app, ansibleBasePath, notificationService))
	scanGroup.POST("/stop", handlers.HandleStopScan(app, ansibleBasePath, notificationService))
	scanGroup.POST("/generate", handlers.HandleGenerateScan(app, ansibleBasePath))
	scanGroup.POST("/destroy", handlers.HandleDestroyScan(app, ansibleBasePath))
	scanGroup.POST("/update-status", handlers.HandleUpdateScanStatus(app))
	scanGroup.POST("/update-logs", handlers.HandleUpdateScanLogs(app))
	scanGroup.POST("/update-archives", handlers.HandleUpdateNucleiScanArchives(app))
	scanGroup.POST("/update-ip", handlers.HandleUpdateScanIP(app))
	scanGroup.POST("/update-cost", handlers.HandleUpdateScanCost(app))
	scanGroup.POST("/update-vm-times", handlers.HandleUpdateVMTimes(app))
	scanGroup.POST("/update-nuclei-times", handlers.HandleUpdateNucleiTimes(app))
	scanGroup.POST("/update-skipped-hosts", handlers.HandleUpdateSkippedHosts(app))
	scanGroup.GET("/current-cost", handlers.HandleGetCurrentCost(app))
	scanGroup.POST("/import-scan-results", handlers.HandleImportNucleiScanResults(app))
	scanGroup.POST("/signed-url", handlers.HandleSignedURL(app))
	scanGroup.POST("/schedule", handlers.HandleScheduleScan(app))
	scanGroup.GET("/scheduled", handlers.HandleGetScheduledScans(app))
	scanGroup.DELETE("/scheduled/:id", handlers.HandleDeleteScheduledScan(app))
}
