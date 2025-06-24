package findings

import (
	// Adjust the import path as necessary
	"bitor/services"

	"net/http"

	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// Ensure you have imported Echo's middleware package

// BatchMigrationResponse represents the response from the batch migration endpoint
type BatchMigrationResponse struct {
	ProcessedCount int    `json:"processedCount"`
	TotalCount     int    `json:"totalCount"`
	Error          string `json:"error,omitempty"`
}

// RegisterRoutes registers the scan routes with the authentication middleware.
func RegisterRoutes(app *pocketbase.PocketBase, e *core.ServeEvent, findingManager *services.FindingManager) {
	// Create a findings routes instance
	routes := NewFindingsRoutes(app, findingManager)

	// Create a middleware that allows either admin or record auth
	authMiddleware := apis.RequireAdminOrRecordAuth()

	// Register routes with combined auth
	findingsGroup := e.Router.Group("/api/findings", authMiddleware)
	findingsGroup.GET("/grouped", HandleGroupedFindings(app))
	findingsGroup.GET("", HandleFindings(app))
	findingsGroup.POST("/bulk-update", HandleBulkUpdateFindings(app))
	findingsGroup.GET("/by-client", HandleVulnerabilitiesByClient(app))
	findingsGroup.GET("/recent", HandleRecentFindings(app))

	// Admin-only routes
	adminGroup := e.Router.Group("/api/findings", apis.RequireAdminAuth())
	adminGroup.DELETE("/client/:id", routes.deleteClientFindings)
	adminGroup.DELETE("/orphaned", routes.deleteOrphanedFindings)
	adminGroup.POST("/migrate-batch", routes.migrateFindingsBatch)
	adminGroup.GET("/migration-status", routes.getMigrationStatus)
}

type FindingsRoutes struct {
	app            *pocketbase.PocketBase
	findingManager *services.FindingManager
}

func NewFindingsRoutes(app *pocketbase.PocketBase, findingManager *services.FindingManager) *FindingsRoutes {
	return &FindingsRoutes{
		app:            app,
		findingManager: findingManager,
	}
}

func (r *FindingsRoutes) deleteClientFindings(c echo.Context) error {
	clientID := c.PathParam("id")
	if clientID == "" {
		return apis.NewBadRequestError("client ID is required", nil)
	}

	if err := r.findingManager.DeleteClientFindings(clientID); err != nil {
		return apis.NewBadRequestError("failed to delete findings", err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "findings deleted successfully",
	})
}

func (r *FindingsRoutes) deleteOrphanedFindings(c echo.Context) error {
	if err := r.findingManager.DeleteOrphanedFindings(); err != nil {
		return apis.NewBadRequestError("failed to delete orphaned findings", err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "orphaned findings deleted successfully",
	})
}

func (r *FindingsRoutes) requireSuperAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get the admin status from the auth store
		admin := c.Get("admin")
		if admin == nil || !admin.(bool) {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "super admin access required",
			})
		}
		return next(c)
	}
}

func (r *FindingsRoutes) migrateFindingsBatch(c echo.Context) error {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in migrateFindingsBatch: %v", r)
			return
		}
	}()

	// Parse request parameters
	offset := c.QueryParam("offset")
	if offset == "" {
		offset = "0"
	}
	limit := c.QueryParam("limit")
	if limit == "" {
		limit = "25"
	}
	migrationOnly := c.QueryParam("migrationOnly") == "true"

	log.Printf("Starting batch migration with offset=%s, limit=%s, migrationOnly=%v", offset, limit, migrationOnly)

	// Get current migration status
	statusCollection, err := r.app.Dao().FindCollectionByNameOrId("migration_status")
	if err != nil {
		log.Printf("Error finding migration_status collection: %v", err)
		return apis.NewBadRequestError("failed to find migration_status collection", err)
	}

	// Try to find existing status record
	records, err := r.app.Dao().FindRecordsByFilter(
		statusCollection.Id,
		"id != ''",
		"",
		0,
		1,
		nil,
	)
	if err != nil {
		log.Printf("Error finding migration status record: %v", err)
		return apis.NewBadRequestError("failed to find migration status record", err)
	}

	var record *models.Record
	var totalCount int

	// Get total count first
	totalCount, err = r.findingManager.GetTotalFindingsCount()
	if err != nil {
		log.Printf("Error getting total findings count: %v", err)
		return apis.NewBadRequestError("failed to get total findings count", err)
	}
	log.Printf("Total findings to process: %d", totalCount)

	if len(records) > 0 {
		record = records[0]
	} else {
		// Create new status record
		record = models.NewRecord(statusCollection)
		record.Set("is_processing", true)
		record.Set("total_count", totalCount)
		record.Set("processed_count", 0)
		record.Set("progress", 0)
		record.Set("error", "")
		record.Set("current_status", fmt.Sprintf("Starting migration of %d findings...", totalCount))
		log.Printf("Created new status record with total_count: %d", totalCount)
	}

	// Save initial record
	if err := r.app.Dao().SaveRecord(record); err != nil {
		log.Printf("Error saving initial migration status record: %v", err)
		return apis.NewBadRequestError("failed to save migration status record", err)
	}

	// Process batch
	processedCount, err := r.findingManager.ProcessFindingsBatch(offset, limit, migrationOnly)
	if err != nil {
		log.Printf("Error processing batch: %v", err)
		record.Set("error", err.Error())
		record.Set("current_status", fmt.Sprintf("Error processing batch: %v", err))
		if saveErr := r.app.Dao().SaveRecord(record); saveErr != nil {
			log.Printf("Failed to update migration status with error: %v", saveErr)
		}
		return apis.NewBadRequestError("failed to process findings batch", err)
	}

	// Update progress
	currentProcessed := int(record.GetFloat("processed_count"))
	newTotal := currentProcessed + processedCount
	progress := float64(newTotal) / float64(totalCount) * 100

	record.Set("processed_count", newTotal)
	record.Set("progress", progress)
	record.Set("current_status", fmt.Sprintf("Processed %d of %d findings (%.1f%%)", newTotal, totalCount, progress))

	// Check if migration is complete
	if newTotal >= totalCount {
		record.Set("is_processing", false)
		record.Set("current_status", fmt.Sprintf("Migration completed. Processed %d findings.", newTotal))
	} else {
		record.Set("is_processing", true)
	}

	// Save the updated record
	if err := r.app.Dao().SaveRecord(record); err != nil {
		log.Printf("Failed to update migration status: %v", err)
		return apis.NewBadRequestError("failed to update migration status", err)
	}

	return c.JSON(http.StatusOK, BatchMigrationResponse{
		ProcessedCount: processedCount,
		TotalCount:     totalCount,
	})
}

// Update the getMigrationStatus function to return actual progress
func (r *FindingsRoutes) getMigrationStatus(c echo.Context) error {
	statusCollection, err := r.app.Dao().FindCollectionByNameOrId("migration_status")
	if err != nil {
		return apis.NewBadRequestError("failed to find migration_status collection", err)
	}

	// Use a more efficient query to get the latest status record
	records, err := r.app.Dao().FindRecordsByFilter(
		statusCollection.Id,
		"id != ''",
		"",
		0,
		1,
		nil,
	)
	if err != nil {
		return apis.NewBadRequestError("failed to find migration status record", err)
	}

	if len(records) == 0 {
		// Return default status if no record exists
		return c.JSON(http.StatusOK, map[string]interface{}{
			"processedCount": 0,
			"totalCount":     0,
			"isProcessing":   false,
			"progress":       0,
			"error":          "",
			"currentStatus":  "",
		})
	}

	record := records[0]
	processedCount := int(record.GetFloat("processed_count"))
	totalCount := int(record.GetFloat("total_count"))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"processedCount": processedCount,
		"totalCount":     totalCount,
		"isProcessing":   record.GetBool("is_processing"),
		"progress":       int(record.GetFloat("progress")),
		"error":          record.GetString("error"),
		"currentStatus":  record.GetString("current_status"),
	})
}
