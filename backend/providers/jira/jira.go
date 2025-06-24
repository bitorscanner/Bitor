package jira

import (
	"fmt"
	"bitor/utils/crypto"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// GetCredentials retrieves the Jira credentials for a provider
func GetCredentials(app *pocketbase.PocketBase, provider *models.Record) (string, string, error) {
	// Get username
	username, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = 'username'",
		dbx.Params{
			"provider": provider.Id,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find username: %w", err)
	}
	if username == nil {
		return "", "", fmt.Errorf("no username found for provider")
	}

	// Get API key
	apiKey, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = 'api_key'",
		dbx.Params{
			"provider": provider.Id,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find API key: %w", err)
	}
	if apiKey == nil {
		return "", "", fmt.Errorf("no API key found for provider")
	}

	// Decrypt username
	encryptedUsername := username.GetString("key")
	if encryptedUsername == "" {
		return "", "", fmt.Errorf("username record exists but 'key' field is empty")
	}
	decryptedUsername, err := crypto.Decrypt(encryptedUsername, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt username: %w", err)
	}

	// Decrypt API key
	encryptedKey := apiKey.GetString("key")
	if encryptedKey == "" {
		return "", "", fmt.Errorf("API key record exists but 'key' field is empty")
	}
	decryptedKey, err := crypto.Decrypt(encryptedKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return string(decryptedUsername), string(decryptedKey), nil
}
