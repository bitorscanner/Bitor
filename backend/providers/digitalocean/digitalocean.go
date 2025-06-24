package digitalocean

import (
	"context"
	"fmt"
	"bitor/utils/crypto"
	"os"

	"github.com/digitalocean/godo"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// FetchProjects fetches all projects from DigitalOcean
func FetchProjects(app *pocketbase.PocketBase, providerID string) ([]map[string]interface{}, error) {
	fmt.Printf("Fetching projects for provider ID: %s\n", providerID)

	// Verify the provider exists
	provider, err := app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		fmt.Printf("Failed to find provider: %v\n", err)
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	fmt.Printf("Found provider: %s\n", provider.Id)

	// Get API key for this provider
	fmt.Printf("Looking for API key with provider=%s and key_type=api_key\n", providerID)

	// First let's see ALL keys for this provider without any filters
	allProviderKeys, err := app.Dao().FindRecordsByFilter(
		"api_keys",
		"provider = {:provider}",
		"",  // No sort needed
		0,   // Page number (0-based)
		100, // Get up to 100 records
		dbx.Params{
			"provider": providerID,
		},
	)
	if err != nil {
		fmt.Printf("Failed to find any provider keys: %v\n", err)
	} else {
		fmt.Printf("Found %d total keys for provider\n", len(allProviderKeys))
		for _, key := range allProviderKeys {
			fmt.Printf("Key ID: %s, Name: %s, Type: %s, Key Type: %s, Provider: %s\n",
				key.Id,
				key.GetString("name"),
				key.GetString("type"),
				key.GetString("key_type"),
				key.GetString("provider"),
			)
		}
	}

	// Now look for the specific API key
	fmt.Printf("Looking for specific API key with provider=%s and key_type=api_key\n", providerID)
	apiKey, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "api_key",
		},
	)
	if err != nil {
		fmt.Printf("Failed to find API key: %v\n", err)
		return nil, fmt.Errorf("failed to find API key: %w", err)
	}
	if apiKey == nil {
		fmt.Println("No API key found for provider")
		return nil, fmt.Errorf("no API key found for provider")
	}
	fmt.Printf("Found API key record with ID: %s\n", apiKey.Id)

	// Decrypt the API key
	encryptedKey := apiKey.GetString("key")
	if encryptedKey == "" {
		fmt.Println("API key record exists but 'key' field is empty")
		return nil, fmt.Errorf("API key record exists but 'key' field is empty")
	}
	fmt.Printf("Found encrypted key of length: %d\n", len(encryptedKey))

	decryptedBytes, err := crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		fmt.Printf("Failed to decrypt API key: %v\n", err)
		// Attempt to re-encrypt with current key
		if reErr := reEncryptAPIKey(app, apiKey); reErr != nil {
			return nil, fmt.Errorf("failed to decrypt and re-encrypt failed: %w", reErr)
		}
		// Try decryption again with the new encryption
		encryptedKey = apiKey.GetString("key")
		decryptedBytes, err = crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt after re-encryption: %w", err)
		}
	}

	return fetchDigitalOceanProjects(string(decryptedBytes))
}

// fetchDigitalOceanProjects makes the actual API call to DigitalOcean
func fetchDigitalOceanProjects(apiKey string) ([]map[string]interface{}, error) {
	fmt.Println("Making API call to DigitalOcean for projects")
	client := godo.NewFromToken(apiKey)
	ctx := context.Background()

	// List all projects
	projects, _, err := client.Projects.List(ctx, &godo.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list projects: %v\n", err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Convert to map format
	var result []map[string]interface{}
	for _, project := range projects {
		result = append(result, map[string]interface{}{
			"id":          project.ID,
			"name":        project.Name,
			"description": project.Description,
		})
	}
	fmt.Printf("Found %d projects\n", len(result))

	return result, nil
}

// FetchRegions fetches all regions from DigitalOcean
func FetchRegions(app *pocketbase.PocketBase, providerID string) ([]map[string]interface{}, error) {
	fmt.Printf("Fetching regions for provider ID: %s\n", providerID)

	// Verify the provider exists
	provider, err := app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		fmt.Printf("Failed to find provider: %v\n", err)
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}
	fmt.Printf("Found provider: %s\n", provider.Id)

	// Get API key for this provider
	fmt.Printf("Looking for API key with provider=%s and key_type=api_key\n", providerID)
	apiKey, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "api_key",
		},
	)
	if err != nil {
		fmt.Printf("Failed to find API key: %v\n", err)
		return nil, fmt.Errorf("failed to find API key: %w", err)
	}
	if apiKey == nil {
		fmt.Println("No API key found for provider")
		return nil, fmt.Errorf("no API key found for provider")
	}
	fmt.Printf("Found API key record with ID: %s\n", apiKey.Id)

	// Decrypt the API key
	encryptedKey := apiKey.GetString("key")
	if encryptedKey == "" {
		fmt.Println("API key record exists but 'key' field is empty")
		return nil, fmt.Errorf("API key record exists but 'key' field is empty")
	}
	fmt.Printf("Found encrypted key of length: %d\n", len(encryptedKey))

	decryptedBytes, err := crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		fmt.Printf("Failed to decrypt API key: %v\n", err)
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return fetchDigitalOceanRegions(string(decryptedBytes))
}

// fetchDigitalOceanRegions makes the actual API call to DigitalOcean
func fetchDigitalOceanRegions(apiKey string) ([]map[string]interface{}, error) {
	fmt.Println("Making API call to DigitalOcean for regions")
	client := godo.NewFromToken(apiKey)
	ctx := context.Background()

	// List all regions
	regions, _, err := client.Regions.List(ctx, &godo.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list regions: %v\n", err)
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	// Convert to map format and filter out SFO1
	var result []map[string]interface{}
	for _, region := range regions {
		// Skip SFO1 region
		if region.Slug == "sfo1" {
			continue
		}
		// Only add if the region is available
		if region.Available {
			result = append(result, map[string]interface{}{
				"slug":      region.Slug,
				"name":      region.Name,
				"available": region.Available,
			})
		}
	}
	fmt.Printf("Found %d regions\n", len(result))

	return result, nil
}

// FetchDomains fetches all domains from DigitalOcean
func FetchDomains(app *pocketbase.PocketBase, providerID string) ([]map[string]interface{}, error) {
	// Verify the provider exists
	if _, err := app.Dao().FindRecordById("providers", providerID); err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	// Get API key for this provider
	fmt.Printf("Looking for API key with provider=%s and key_type=api_key\n", providerID)
	apiKey, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "api_key",
		},
	)
	if err != nil {
		fmt.Printf("Failed to find API key: %v\n", err)
		return nil, fmt.Errorf("failed to find API key: %w", err)
	}
	if apiKey == nil {
		fmt.Println("No API key found for provider")
		return nil, fmt.Errorf("no API key found for provider")
	}
	fmt.Printf("Found API key record with ID: %s\n", apiKey.Id)

	// Decrypt the API key
	encryptedKey := apiKey.GetString("key")
	if encryptedKey == "" {
		fmt.Println("API key record exists but 'key' field is empty")
		return nil, fmt.Errorf("API key record exists but 'key' field is empty")
	}
	fmt.Printf("Found encrypted key of length: %d\n", len(encryptedKey))

	decryptedBytes, err := crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return fetchDigitalOceanDomains(string(decryptedBytes))
}

// fetchDigitalOceanDomains makes the actual API call to DigitalOcean
func fetchDigitalOceanDomains(apiKey string) ([]map[string]interface{}, error) {
	client := godo.NewFromToken(apiKey)
	ctx := context.Background()

	// List all domains
	domains, _, err := client.Domains.List(ctx, &godo.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	// Convert to map format
	var result []map[string]interface{}
	for _, domain := range domains {
		result = append(result, map[string]interface{}{
			"name": domain.Name,
			"ttl":  domain.TTL,
		})
	}

	return result, nil
}

// reEncryptAPIKey attempts to decrypt with the old key and re-encrypt with the new key
func reEncryptAPIKey(app *pocketbase.PocketBase, apiKey *models.Record) error {
	// Try to decrypt with the old key (this is just an example, you'll need to provide the old key)
	oldKey := "your_old_key_here" // Replace this with the actual old key
	encryptedKey := apiKey.GetString("key")

	decryptedBytes, err := crypto.Decrypt(encryptedKey, oldKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt with old key: %w", err)
	}

	// Re-encrypt with the new key
	newEncrypted, err := crypto.Encrypt(decryptedBytes, os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return fmt.Errorf("failed to re-encrypt: %w", err)
	}

	// Update the record
	apiKey.Set("key", newEncrypted)
	if err := app.Dao().SaveRecord(apiKey); err != nil {
		return fmt.Errorf("failed to save re-encrypted key: %w", err)
	}

	return nil
}

// FetchSizes fetches available droplet sizes for a specific region from DigitalOcean
func FetchSizes(app *pocketbase.PocketBase, providerID string, region string) ([]map[string]interface{}, error) {
	// Get max cost setting
	systemSettings, err := app.Dao().FindFirstRecordByFilter("system_settings", "id != ''")
	if err != nil {
		return nil, fmt.Errorf("failed to get system settings: %w", err)
	}
	maxCostPerMonth := 0.0
	if cost, ok := systemSettings.Get("max_cost_per_month").(float64); ok {
		maxCostPerMonth = cost
	}

	// Get provider's API key
	provider, err := app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	apiKey, err := GetAPIKey(app, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	sizes, err := fetchDigitalOceanSizes(apiKey, region)
	if err != nil {
		return nil, err
	}

	// Filter sizes based on max cost
	if maxCostPerMonth > 0 {
		filteredSizes := make([]map[string]interface{}, 0)
		for _, size := range sizes {
			if cost, ok := size["price_monthly"].(float64); ok && cost <= maxCostPerMonth {
				filteredSizes = append(filteredSizes, size)
			}
		}
		sizes = filteredSizes
	}

	return sizes, nil
}

// fetchDigitalOceanSizes makes the actual API call to DigitalOcean
func fetchDigitalOceanSizes(apiKey string, region string) ([]map[string]interface{}, error) {
	client := godo.NewFromToken(apiKey)
	ctx := context.Background()

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200, // Maximum allowed by DigitalOcean API
	}

	var allSizes []godo.Size
	for {
		sizes, resp, err := client.Sizes.List(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %w", err)
		}
		allSizes = append(allSizes, sizes...)

		// Check if we've reached the last page
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, fmt.Errorf("failed to get current page: %w", err)
		}
		opt.Page = page + 1
	}

	// Filter sizes available in the specified region
	var result []map[string]interface{}
	for _, size := range allSizes {
		// Check if this size is available in the specified region
		for _, availableRegion := range size.Regions {
			if availableRegion == region {
				result = append(result, map[string]interface{}{
					"slug":          size.Slug,
					"memory":        size.Memory,
					"vcpus":         size.Vcpus,
					"disk":          size.Disk,
					"transfer":      size.Transfer,
					"price_monthly": size.PriceMonthly,
					"price_hourly":  size.PriceHourly,
					"description":   fmt.Sprintf("%d vCPUs, %d GB RAM, %d GB SSD", size.Vcpus, size.Memory/1024, size.Disk),
				})
				break
			}
		}
	}

	return result, nil
}

// GetAPIKey retrieves the API key for a provider
func GetAPIKey(app *pocketbase.PocketBase, provider *models.Record) (string, error) {
	apiKey, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = 'api_key'",
		dbx.Params{
			"provider": provider.Id,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to find API key: %w", err)
	}
	if apiKey == nil {
		return "", fmt.Errorf("no API key found for provider")
	}

	// Get the encrypted key
	encryptedKey := apiKey.GetString("key")
	if encryptedKey == "" {
		return "", fmt.Errorf("API key record exists but 'key' field is empty")
	}

	// Decrypt the key
	decryptedBytes, err := crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return string(decryptedBytes), nil
}

// GetSizePrice fetches the hourly price for a specific size in a region
func GetSizePrice(apiKey string, region string, size string) (float64, error) {
	sizes, err := fetchDigitalOceanSizes(apiKey, region)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch sizes: %w", err)
	}

	for _, s := range sizes {
		if s["slug"] == size {
			if price, ok := s["price_hourly"].(float64); ok {
				return price, nil
			}
			return 0, fmt.Errorf("invalid price format for size %s", size)
		}
	}

	return 0, fmt.Errorf("size %s not found in region %s", size, region)
}
