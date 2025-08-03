package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/digitalocean/godo"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

func HandleUpdateScanStatus(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var statusUpdate struct {
			ScanID  string `json:"scan_id"`
			Status  string `json:"status"`
			Message string `json:"message,omitempty"`
		}
		if err := c.Bind(&statusUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and status are provided
		if statusUpdate.ScanID == "" || statusUpdate.Status == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and status are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", statusUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Update the scan status
		record.Set("status", statusUpdate.Status)

		// Update progress message if provided
		if statusUpdate.Message != "" {
			record.Set("progress_message", statusUpdate.Message)
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan status",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan status updated",
		})
	}
}

func HandleUpdateScanProgress(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var progressUpdate struct {
			ScanID   string `json:"scan_id"`
			Message  string `json:"message"`
			Progress int    `json:"progress,omitempty"` // 0-100 percentage
		}
		if err := c.Bind(&progressUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and message are provided
		if progressUpdate.ScanID == "" || progressUpdate.Message == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and message are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", progressUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Update progress message
		record.Set("progress_message", progressUpdate.Message)

		// Update progress percentage if provided
		if progressUpdate.Progress >= 0 && progressUpdate.Progress <= 100 {
			record.Set("progress_percentage", progressUpdate.Progress)
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan progress",
			})
		}

		log.Printf("Scan %s progress: %s", progressUpdate.ScanID, progressUpdate.Message)

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan progress updated",
		})
	}
}

func HandleUpdateScanLogs(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var logUpdate struct {
			ScanID  string `json:"scan_id"`
			LogsB64 string `json:"logs_b64"`
		}
		if err := c.Bind(&logUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and logs are provided
		if logUpdate.ScanID == "" || logUpdate.LogsB64 == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and logs are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", logUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Decode the base64-encoded logs
		decodedLogs, err := base64.StdEncoding.DecodeString(logUpdate.LogsB64)
		if err != nil {
			log.Printf("Failed to decode logs: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to decode logs",
			})
		}

		// Use the decoded logs
		logsContent := string(decodedLogs)

		// Convert the logs to a log entry with a timestamp
		logEntry := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"content":   logsContent,
		}

		// Initialize empty logs slice
		var existingLogs []interface{}

		// Fetch existing logs from the database
		ansibleLogsValue := record.Get("ansible_logs")
		if ansibleLogsValue != nil {
			// Try to convert directly to slice
			if logsSlice, ok := ansibleLogsValue.([]interface{}); ok {
				existingLogs = logsSlice
			} else {
				// If not a slice, try to unmarshal from JSON string
				var logs []interface{}
				if jsonStr, ok := ansibleLogsValue.(string); ok {
					if err := json.Unmarshal([]byte(jsonStr), &logs); err == nil {
						existingLogs = logs
					}
				} else {
					// Try to marshal and unmarshal the value
					if jsonBytes, err := json.Marshal(ansibleLogsValue); err == nil {
						if err := json.Unmarshal(jsonBytes, &logs); err == nil {
							existingLogs = logs
						}
					}
				}
			}
		}

		// If we still don't have a valid slice, initialize a new one
		if existingLogs == nil {
			existingLogs = make([]interface{}, 0)
		}

		// Append the new log entry
		existingLogs = append(existingLogs, logEntry)

		// Update the record
		record.Set("ansible_logs", existingLogs)
		if err := app.Dao().SaveRecord(record); err != nil {
			log.Printf("Failed to save record: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update record with logs",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Logs updated",
		})
	}
}

func HandleUpdateNucleiScanArchives(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var archiveUpdate struct {
			ClientID     string `json:"client_id"`
			S3ProviderID string `json:"s3_provider_id"`
			ScanID       string `json:"scan_id"`
			S3FullPath   string `json:"s3_full_path"`
			S3SmallPath  string `json:"s3_small_path"`
		}
		if err := c.Bind(&archiveUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure all required fields are provided
		if archiveUpdate.ClientID == "" || archiveUpdate.ScanID == "" || archiveUpdate.S3FullPath == "" || archiveUpdate.S3SmallPath == "" || archiveUpdate.S3ProviderID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "All fields are required",
			})
		}

		// Retrieve the collection
		collection, err := app.Dao().FindCollectionByNameOrId("nuclei_scan_archives")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find collection nuclei_scan_archives",
			})
		}

		// Create a new record
		record := models.NewRecord(collection)
		record.Set("client_id", archiveUpdate.ClientID)
		record.Set("s3_provider_id", archiveUpdate.S3ProviderID)
		record.Set("scan_id", archiveUpdate.ScanID)
		record.Set("s3_full_path", archiveUpdate.S3FullPath)
		record.Set("s3_small_path", archiveUpdate.S3SmallPath)

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update nuclei scan archives",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Nuclei scan archives updated",
		})
	}
}

func HandleUpdateScanIP(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var ipUpdate struct {
			ScanID    string `json:"scan_id"`
			IPAddress string `json:"ip_address"`
		}
		if err := c.Bind(&ipUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and IP address are provided
		if ipUpdate.ScanID == "" || ipUpdate.IPAddress == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and IP address are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", ipUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Update the scan IP address
		record.Set("ip_address", ipUpdate.IPAddress)
		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan IP address",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Scan IP address updated",
		})
	}
}

func HandleUpdateScanCost(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var costUpdate struct {
			ScanID    string `json:"scan_id"`
			StartTime string `json:"start_time"`
			EndTime   string `json:"end_time"`
			VMSize    string `json:"vm_size"`
		}
		if err := c.Bind(&costUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure all required fields are provided
		if costUpdate.ScanID == "" || costUpdate.StartTime == "" || costUpdate.EndTime == "" || costUpdate.VMSize == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "All fields are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", costUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// If cost is already set, don't recalculate
		if record.Get("cost") != nil {
			return c.JSON(http.StatusOK, map[string]string{
				"status": "Cost already calculated",
			})
		}

		// Parse start and end times
		startTime, err := time.Parse(time.RFC3339, costUpdate.StartTime)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid start time format",
			})
		}

		endTime, err := time.Parse(time.RFC3339, costUpdate.EndTime)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid end time format",
			})
		}

		// Calculate duration in hours (rounded up to nearest hour)
		duration := endTime.Sub(startTime)
		hours := math.Ceil(duration.Hours())

		// Get the VM provider record to fetch pricing
		vmProviderID := record.Get("vm_provider").(string)
		vmProvider, err := app.Dao().FindRecordById("providers", vmProviderID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find VM provider",
			})
		}

		// Get the provider's API key
		apiKey, err := getAPIKey(app, vmProvider)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get provider API key",
			})
		}

		// Get the region from provider settings
		providerSettings := vmProvider.Get("settings").(map[string]interface{})
		region := providerSettings["region"].(string)

		// Fetch droplet sizes from DigitalOcean
		sizes, err := fetchDigitalOceanSizes(apiKey, region)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch droplet sizes",
			})
		}

		// Find the matching size and get hourly price
		var hourlyPrice float64
		for _, size := range sizes {
			if size["slug"] == costUpdate.VMSize {
				hourlyPrice = size["price_hourly"].(float64)
				break
			}
		}

		// Calculate total cost
		totalCost := hourlyPrice * hours

		// Update the record with the calculated cost
		record.Set("cost", totalCost)
		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update scan cost",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "Cost updated",
			"cost":   totalCost,
		})
	}
}

func HandleGetCurrentCost(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		scanID := c.QueryParam("scan_id")
		log.Printf("Calculating cost for scan ID: %s", scanID)

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
		if err != nil {
			log.Printf("Error finding scan: %v", err)
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// If cost is already set, return it
		if cost := record.Get("cost"); cost != nil {
			log.Printf("Cost already set: %v", cost)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"cost":  cost,
				"final": true,
			})
		}

		// Get start time
		startTimeStr := record.Get("start_time")
		log.Printf("Start time: %v", startTimeStr)
		if startTimeStr == nil {
			log.Printf("No start time found")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"cost":  0,
				"final": false,
			})
		}

		startTime, err := time.Parse(time.RFC3339, startTimeStr.(string))
		if err != nil {
			log.Printf("Error parsing start time: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid start time format",
			})
		}

		// Calculate duration until now
		duration := time.Since(startTime)
		hours := math.Ceil(duration.Hours())
		log.Printf("Duration in hours: %v", hours)

		// Get the VM provider record to fetch pricing
		vmProviderID := record.Get("vm_provider").(string)
		log.Printf("VM Provider ID: %s", vmProviderID)
		vmProvider, err := app.Dao().FindRecordById("providers", vmProviderID)
		if err != nil {
			log.Printf("Error finding VM provider: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to find VM provider",
			})
		}

		// Get the provider's API key
		apiKey, err := getAPIKey(app, vmProvider)
		if err != nil {
			log.Printf("Error getting API key: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get provider API key",
			})
		}

		// Get the region from provider settings
		providerSettings := vmProvider.Get("settings").(map[string]interface{})
		region := providerSettings["region"].(string)
		log.Printf("Region: %s", region)

		// Get the VM size from the scan record
		vmSize := record.Get("vm_size")
		log.Printf("VM Size from scan record: %v", vmSize)
		if vmSize == nil {
			// Try getting it from the provider settings
			vmSize = providerSettings["size"]
			log.Printf("VM Size from provider settings: %v", vmSize)
			if vmSize == nil {
				log.Printf("Could not determine VM size")
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Could not determine VM size",
				})
			}
		}

		// Fetch droplet sizes from DigitalOcean
		sizes, err := fetchDigitalOceanSizes(apiKey, region)
		if err != nil {
			log.Printf("Error fetching droplet sizes: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch droplet sizes",
			})
		}
		log.Printf("Retrieved %d sizes from DigitalOcean", len(sizes))

		// Find the matching size and get hourly price
		var hourlyPrice float64
		vmSizeStr := vmSize.(string)
		log.Printf("Looking for price of size: %s", vmSizeStr)
		for _, size := range sizes {
			log.Printf("Checking size: %v", size["slug"])
			if size["slug"] == vmSizeStr {
				hourlyPrice = size["price_hourly"].(float64)
				log.Printf("Found hourly price: %v", hourlyPrice)
				break
			}
		}

		if hourlyPrice == 0 {
			log.Printf("Could not find price for VM size: %s", vmSizeStr)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Could not find price for VM size",
			})
		}

		// Calculate current cost
		currentCost := hourlyPrice * hours
		log.Printf("Calculated cost: %v (hourly price: %v × hours: %v)", currentCost, hourlyPrice, hours)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"cost":  currentCost,
			"final": false,
		})
	}
}

func HandleUpdateVMTimes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var timeUpdate struct {
			ScanID    string `json:"scan_id"`
			StartTime string `json:"start_time"`
			StopTime  string `json:"stop_time,omitempty"`
		}
		if err := c.Bind(&timeUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and start time are provided
		if timeUpdate.ScanID == "" || timeUpdate.StartTime == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and start time are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", timeUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Update the VM times
		record.Set("vm_start_time", timeUpdate.StartTime)
		if timeUpdate.StopTime != "" {
			record.Set("vm_stop_time", timeUpdate.StopTime)
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update VM times",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "VM times updated",
		})
	}
}

func HandleUpdateNucleiTimes(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var timeUpdate struct {
			ScanID    string `json:"scan_id"`
			StartTime string `json:"start_time"`
			StopTime  string `json:"stop_time,omitempty"`
		}
		if err := c.Bind(&timeUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and start time are provided
		if timeUpdate.ScanID == "" || timeUpdate.StartTime == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and start time are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", timeUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Update the Nuclei times
		record.Set("nuclei_start_time", timeUpdate.StartTime)
		if timeUpdate.StopTime != "" {
			record.Set("nuclei_stop_time", timeUpdate.StopTime)
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update Nuclei times",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Nuclei times updated",
		})
	}
}

func HandleUpdateSkippedHosts(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the request payload
		var skippedUpdate struct {
			ScanID  string `json:"scan_id"`
			LogsB64 string `json:"logs_b64"`
		}
		if err := c.Bind(&skippedUpdate); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request",
			})
		}

		// Ensure scan ID and logs are provided
		if skippedUpdate.ScanID == "" || skippedUpdate.LogsB64 == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Scan ID and logs are required",
			})
		}

		// Find the scan record
		record, err := app.Dao().FindRecordById("nuclei_scans", skippedUpdate.ScanID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Scan not found",
			})
		}

		// Decode the base64-encoded logs
		decodedLogs, err := base64.StdEncoding.DecodeString(skippedUpdate.LogsB64)
		if err != nil {
			log.Printf("Failed to decode logs: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to decode logs",
			})
		}

		// Split the logs into lines and extract skipped hosts
		skippedHosts := []string{}
		for _, line := range bytes.Split(decodedLogs, []byte("\n")) {
			if len(line) > 0 {
				skippedHosts = append(skippedHosts, string(line))
			}
		}

		// Update the record with skipped hosts
		record.Set("skipped_hosts", skippedHosts)
		if err := app.Dao().SaveRecord(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update record with skipped hosts",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "Skipped hosts updated",
		})
	}
}

// Helper function to get API key
func getAPIKey(app *pocketbase.PocketBase, provider *models.Record) (string, error) {
	apiKeys, err := app.Dao().FindRecordsByFilter("api_keys", "provider = {:provider} && key_type = 'api_key'", "created", 1, 1, dbx.Params{
		"provider": provider.Id,
	})
	if err != nil {
		return "", fmt.Errorf("failed to find API key: %w", err)
	}
	if len(apiKeys) == 0 {
		return "", fmt.Errorf("no API key found for provider")
	}
	return apiKeys[0].Get("key").(string), nil
}

// Helper function to fetch DigitalOcean sizes
func fetchDigitalOceanSizes(apiKey string, region string) ([]map[string]interface{}, error) {
	client := godo.NewFromToken(apiKey)
	ctx := context.Background()

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	var allSizes []godo.Size
	for {
		sizes, resp, err := client.Sizes.List(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %w", err)
		}
		allSizes = append(allSizes, sizes...)

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("failed to get current page: %w", err)
		}
		opt.Page = page + 1
	}

	var result []map[string]interface{}
	for _, size := range allSizes {
		for _, availableRegion := range size.Regions {
			if availableRegion == region {
				result = append(result, map[string]interface{}{
					"slug":         size.Slug,
					"price_hourly": size.PriceHourly,
				})
				break
			}
		}
	}

	return result, nil
}
