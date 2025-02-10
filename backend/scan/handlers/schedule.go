package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	pbModels "github.com/pocketbase/pocketbase/models"

	"orbit/models"
)

func HandleScheduleScan(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var scheduleReq models.ScheduleRequest
		if err := c.Bind(&scheduleReq); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request payload",
			})
		}

		// Validate the request data
		if scheduleReq.ScanID == "" || scheduleReq.Frequency == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and frequency are required",
			})
		}

		// Use the StartDate directly
		startDate := scheduleReq.StartDate
		if startDate.IsZero() {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Start date is required",
			})
		}

		// Save the schedule to the database
		collection, err := app.Dao().FindCollectionByNameOrId("scheduled_scans")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find collection",
			})
		}

		record := models.NewRecord(collection)
		record.Set("scan_id", scheduleReq.ScanID)
		record.Set("frequency", scheduleReq.Frequency)
		record.Set("cron_expression", scheduleReq.CronExpression)
		record.Set("start_date", startDate)
		record.Set("schedule_details", scheduleReq.ScheduleDetails)

		// Only set end_date if it's provided and valid
		if !scheduleReq.EndDate.IsZero() {
			record.Set("end_date", scheduleReq.EndDate)
		} else {
			record.Set("end_date", nil) // Set to null if no end date provided
		}

		record.Set("created", time.Now())

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error":   "Failed to save scheduled scan",
				"details": err.Error(),
			})
		}

		// Return the full record data for better client-side handling
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "Scan scheduled successfully",
			"id":     record.Id,
			"data": map[string]interface{}{
				"id":               record.Id,
				"scan_id":          record.GetString("scan_id"),
				"frequency":        record.GetString("frequency"),
				"cron_expression":  record.GetString("cron_expression"),
				"start_date":       record.GetDateTime("start_date"),
				"end_date":         record.GetDateTime("end_date"),
				"schedule_details": record.Get("schedule_details"),
				"created":          record.GetDateTime("created"),
			},
		})
	}
}

func HandleGetScheduledScans(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Fetch all scheduled scans
		records, err := app.Dao().FindRecordsByExpr("scheduled_scans", nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to retrieve scheduled scans",
			})
		}

		// Prepare the response data
		scheduledScans := make([]map[string]interface{}, 0)
		for _, record := range records {
			scheduledScans = append(scheduledScans, map[string]interface{}{
				"id":               record.Id,
				"scan_id":          record.GetString("scan_id"),
				"frequency":        record.GetString("frequency"),
				"cron_expression":  record.GetString("cron_expression"),
				"start_date":       record.GetDateTime("start_date"),
				"end_date":         record.GetDateTime("end_date"),
				"schedule_details": record.Get("schedule_details"),
			})
		}

		return c.JSON(http.StatusOK, scheduledScans)
	}
}

func HandleDeleteScheduledScan(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scheduled scan ID is required",
			})
		}

		// Find the record
		record, err := app.Dao().FindRecordById("scheduled_scans", id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scheduled scan not found",
			})
		}

		// Delete the record
		if err := app.Dao().DeleteRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to delete scheduled scan",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scheduled scan deleted successfully",
		})
	}
}

// recordToScheduledScan converts a PocketBase record to a ScheduledScan model.
// Currently unused but retained for future use in scan scheduling.
// It will be used for bulk scan scheduling operations.
// nolint:unused
func recordToScheduledScan(record *pbModels.Record) models.ScheduledScan {
	var scheduleDetails *models.ScheduleDetails
	if details := record.Get("schedule_details"); details != nil {
		// Try to convert the details to our struct
		if detailsMap, ok := details.(map[string]interface{}); ok {
			scheduleDetails = &models.ScheduleDetails{
				Type:         detailsMap["type"].(string),
				Frequency:    detailsMap["frequency"].(string),
				SelectedDays: convertToStringSlice(detailsMap["selectedDays"]),
				MonthlyType:  getString(detailsMap["monthlyType"]),
				MonthlyDate:  getInt(detailsMap["monthlyDate"]),
				MonthlyDay:   getString(detailsMap["monthlyDay"]),
				MonthlyWeek:  getString(detailsMap["monthlyWeek"]),
			}
		}
	}

	return models.ScheduledScan{
		ID:              record.Id,
		ScanID:          record.GetString("scan_id"),
		Frequency:       record.GetString("frequency"),
		CronExpression:  record.GetString("cron_expression"),
		StartDate:       record.GetTime("start_date"),
		EndDate:         record.GetTime("end_date"),
		ScheduleDetails: scheduleDetails,
		Created:         record.GetTime("created"),
	}
}

// convertToStringSlice is a helper function to convert interface{} to []string.
// Currently unused but retained for future use in type conversion.
// It will be used for handling dynamic schedule configurations.
// nolint:unused
func convertToStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

// getString is a helper function to safely convert interface{} to string.
// Currently unused but retained for future use in type conversion.
// It will be used for handling dynamic schedule configurations.
// nolint:unused
func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// getInt is a helper function to safely convert interface{} to int.
// Currently unused but retained for future use in type conversion.
// It will be used for handling dynamic schedule configurations.
// nolint:unused
func getInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
