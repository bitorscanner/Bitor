package utils

import (
	"fmt"
	"bitor/providers/digitalocean"

	"github.com/pocketbase/pocketbase"
)

// GetVMPrice returns the hourly price for a VM size from a provider
func GetVMPrice(app *pocketbase.PocketBase, providerId string, region string, size string) (float64, error) {
	// Get the provider record
	provider, err := app.Dao().FindRecordById("providers", providerId)
	if err != nil {
		return 0, fmt.Errorf("failed to find provider: %v", err)
	}

	// Get the provider type
	providerType := provider.GetString("provider_type")

	// Handle different provider types
	switch providerType {
	case "digitalocean":
		// Get the API key
		apiKey, err := digitalocean.GetAPIKey(app, provider)
		if err != nil {
			return 0, fmt.Errorf("failed to get API key: %v", err)
		}

		// Get the price from DigitalOcean
		price, err := digitalocean.GetSizePrice(apiKey, region, size)
		if err != nil {
			return 0, fmt.Errorf("failed to get size price: %v", err)
		}

		return price, nil
	default:
		return 0, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
