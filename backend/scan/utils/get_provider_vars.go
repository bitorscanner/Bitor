package utils

import (
	"fmt"
	"log"

	"bitor/models"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	pbModels "github.com/pocketbase/pocketbase/models"
)

func GetProviderVars(app *pocketbase.PocketBase, provider *pbModels.Record) (*models.ProviderVars, error) {
	log.Printf("Getting provider vars for provider ID: %s", provider.Id)

	// Get API keys for this provider
	apiKeys, err := app.Dao().FindRecordsByFilter(
		"api_keys",
		"provider = {:provider}",
		"",
		0,
		100,
		dbx.Params{
			"provider": provider.Id,
		},
	)
	if err != nil {
		log.Printf("Failed to find API keys: %v", err)
		return nil, fmt.Errorf("failed to find API keys: %w", err)
	}

	log.Printf("Found %d API keys for provider", len(apiKeys))

	var apiKey, secretKey string
	for _, key := range apiKeys {
		keyType := key.GetString("key_type")
		if provider.GetString("provider_type") == "s3" {
			if keyType == "access_key" {
				apiKey = key.GetString("key")
			} else if keyType == "secret_key" {
				secretKey = key.GetString("key")
			}
		} else if keyType == "api_key" {
			apiKey = key.GetString("key")
		}
	}

	if apiKey == "" || (provider.GetString("provider_type") == "s3" && secretKey == "") {
		log.Println("API key or secret key not found")
		return nil, fmt.Errorf("API key or secret key not found")
	}

	// Create settings based on provider type
	settings := models.Settings{}
	providerType := provider.GetString("provider_type")

	switch providerType {
	case "s3":
		settings.Bucket = provider.GetString("settings.bucket")
		settings.Endpoint = provider.GetString("settings.endpoint")
		settings.Region = provider.GetString("settings.region")
		settings.UsePathStyle = provider.GetBool("settings.use_path_style")
		settings.StatefilePath = provider.GetString("settings.statefile_path")
		settings.ScansPath = provider.GetString("settings.scans_path")
	case "digitalocean":
		settings.Project = provider.GetString("settings.project")
		settings.Tags = provider.GetStringSlice("settings.tags")
	case "aws":
		settings.AWSRegion = provider.GetString("settings.region")
		settings.AccountID = provider.GetString("settings.account_id")
		settings.VPC = provider.GetString("settings.vpc")
		settings.Subnet = provider.GetString("settings.subnet")
		settings.InstanceType = provider.GetString("settings.instance_type")
		settings.AWSTags = provider.GetStringSlice("settings.tags")
	}

	vars := &models.ProviderVars{
		ProviderType: providerType,
		Name:         provider.GetString("name"),
		Key:          apiKey,
		SecretKey:    secretKey,
		Settings:     settings,
		Uses:         provider.GetStringSlice("use"),
	}

	log.Println("Successfully retrieved provider vars")
	return vars, nil
}
