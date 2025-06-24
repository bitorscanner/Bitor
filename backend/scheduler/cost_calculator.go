package scheduler

import (
	"fmt"
	"log"
	"math"
	"time"

	"bitor/utils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// CalculateScanCosts runs periodically to calculate and store costs for completed scans
func CalculateScanCosts(app *pocketbase.PocketBase) error {
	// Find all scans that:
	// 1. Don't have a cost set
	// 2. Have both vm_start_time and vm_stop_time
	// 3. Are not manual scans
	records, err := app.Dao().FindRecordsByFilter(
		"nuclei_scans",
		"cost = null && vm_start_time != '' && vm_stop_time != '' && status != 'Manual'",
		"-created",
		100,
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to find scans: %w", err)
	}

	for _, record := range records {
		if err := calculateAndStoreCost(app, record); err != nil {
			log.Printf("Failed to calculate cost for scan %s: %v", record.Id, err)
			continue
		}
	}

	return nil
}

func calculateAndStoreCost(app *pocketbase.PocketBase, scan *models.Record) error {
	// Parse start and end times
	startTime, err := time.Parse(time.RFC3339, scan.Get("vm_start_time").(string))
	if err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}

	stopTime, err := time.Parse(time.RFC3339, scan.Get("vm_stop_time").(string))
	if err != nil {
		return fmt.Errorf("invalid stop time format: %w", err)
	}

	// Calculate duration in hours (rounded up to nearest hour)
	duration := stopTime.Sub(startTime)
	hours := math.Ceil(duration.Hours())

	// Get the VM provider record to fetch pricing
	vmProviderID := scan.Get("vm_provider").(string)
	vmProvider, err := app.Dao().FindRecordById("providers", vmProviderID)
	if err != nil {
		return fmt.Errorf("failed to find VM provider: %w", err)
	}

	// Get the provider's API key
	apiKeys, err := app.Dao().FindRecordsByFilter("api_keys", "provider = {:provider} && key_type = 'api_key'", "created", 1, 1, dbx.Params{
		"provider": vmProvider.Id,
	})
	if err != nil {
		return fmt.Errorf("failed to find API key: %w", err)
	}
	if len(apiKeys) == 0 {
		return fmt.Errorf("no API key found for provider")
	}
	apiKey := apiKeys[0].Get("key").(string)

	// Get the region from provider settings
	providerSettings := vmProvider.Get("settings").(map[string]interface{})
	region := providerSettings["region"].(string)

	// Get the VM size
	vmSize := scan.Get("vm_size")
	if vmSize == nil {
		vmSize = providerSettings["size"]
		if vmSize == nil {
			return fmt.Errorf("could not determine VM size")
		}
	}

	// Fetch droplet sizes from DigitalOcean using the shared utility
	sizes, err := utils.FetchDigitalOceanSizes(apiKey, region)
	if err != nil {
		return fmt.Errorf("failed to fetch droplet sizes: %w", err)
	}

	// Find the matching size and get hourly price
	var hourlyPrice float64
	vmSizeStr := vmSize.(string)
	for _, size := range sizes {
		if size["slug"] == vmSizeStr {
			hourlyPrice = size["price_hourly"].(float64)
			break
		}
	}

	if hourlyPrice == 0 {
		return fmt.Errorf("could not find price for VM size: %s", vmSizeStr)
	}

	// Calculate total cost
	totalCost := hourlyPrice * hours

	// Update the record with the calculated cost
	scan.Set("cost", totalCost)
	if err := app.Dao().SaveRecord(scan); err != nil {
		return fmt.Errorf("failed to update scan cost: %w", err)
	}

	return nil
}
