package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"github.com/pocketbase/pocketbase"
)

// GenerateAPIKey generates a secure random API key.
func GenerateAPIKey() string {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Error generating random bytes: %v", err)
		return ""
	}
	return hex.EncodeToString(bytes)
}

// isValidAPIKey is a helper function to validate a scan's API key.
// Currently unused but retained for future use in API key validation.
// It will be used for additional security validation in upcoming features.
// nolint:unused
func isValidAPIKey(app *pocketbase.PocketBase, scanID, apiKey string) bool {
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		log.Printf("Error finding record: %v", err)
		return false
	}
	storedApiKey := record.GetString("api_key")
	log.Printf("Comparing API keys: provided=%s, stored=%s", apiKey, storedApiKey)
	return storedApiKey == apiKey
}
