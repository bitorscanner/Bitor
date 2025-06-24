package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"

	"bitor/models"
	"bitor/utils/crypto"
)

// HandleSignedURL generates a signed URL for accessing S3 objects
func HandleSignedURL(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Bind the JSON request body into SignedURLRequest struct
		var req models.SignedURLRequest
		if err := c.Bind(&req); err != nil {
			log.Println("Error binding request:", err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request payload",
			})
		}

		scanId := req.ScanID
		fileType := req.FileType

		// Check if required fields are present
		if scanId == "" {
			log.Println("HandleSignedURL: 'scanId' parameter is missing")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "scanId is required",
			})
		}

		if fileType == "" {
			log.Println("HandleSignedURL: 'fileType' parameter is missing")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "fileType is required",
			})
		}

		log.Printf("HandleSignedURL: scanId=%s, fileType=%s", scanId, fileType)

		// Query the database for the latest archive with the given scanId
		var archives []models.NucleiScanArchive
		err := app.Dao().DB().NewQuery(
			"SELECT * FROM nuclei_scan_archives WHERE scan_id = {:scanId} ORDER BY created DESC LIMIT 1",
		).Bind(dbx.Params{"scanId": scanId}).All(&archives)

		if err != nil {
			log.Println("HandleSignedURL: Query error:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Database query failed",
			})
		}
		if len(archives) == 0 {
			log.Printf("HandleSignedURL: No archives found for scanId=%s", scanId)
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "No archives found",
			})
		}

		archive := archives[0]

		// Access the 's3_provider' from the archive
		s3ProviderId := archive.S3ProviderID
		if s3ProviderId == "" {
			log.Println("HandleSignedURL: 's3_provider' is empty")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid S3 provider ID",
			})
		}

		var filePath string
		if fileType == "full" {
			filePath = archive.S3FullPath
		} else if fileType == "small" {
			filePath = archive.S3SmallPath
		} else {
			log.Printf("HandleSignedURL: Unsupported fileType=%s", fileType)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Unsupported fileType",
			})
		}

		if filePath == "" {
			log.Println("HandleSignedURL: file path is empty")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "File path is empty",
			})
		}

		// Fetch the provider record
		providerRecord, err := app.Dao().FindRecordById("providers", s3ProviderId)
		if err != nil {
			log.Println("HandleSignedURL: Failed to find provider record:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch provider record",
			})
		}

		// Check if provider is configured for scan storage
		uses := providerRecord.GetStringSlice("use")
		hasStorageUse := false
		for _, use := range uses {
			if use == "scan_storage" {
				hasStorageUse = true
				break
			}
		}
		if !hasStorageUse {
			log.Println("HandleSignedURL: Provider is not configured for scan storage")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Provider is not configured for scan storage",
			})
		}

		// Get S3 settings from the provider's settings field
		var s3Settings struct {
			Region       string `json:"region"`
			Endpoint     string `json:"endpoint"`
			UsePathStyle bool   `json:"use_path_style"`
			Bucket       string `json:"bucket"`
		}
		settings := providerRecord.GetString("settings")
		if settings == "" {
			log.Println("HandleSignedURL: Provider settings are empty")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Provider settings are not configured",
			})
		}

		if err := json.Unmarshal([]byte(settings), &s3Settings); err != nil {
			log.Printf("HandleSignedURL: Failed to parse provider settings: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid provider settings",
			})
		}

		// Validate S3 settings
		if s3Settings.Region == "" {
			log.Println("HandleSignedURL: Region is empty in settings")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "S3 region not configured in provider settings",
			})
		}

		if s3Settings.Endpoint == "" {
			log.Println("HandleSignedURL: Endpoint is empty in settings")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "S3 endpoint not configured in provider settings",
			})
		}

		if s3Settings.Bucket == "" {
			log.Println("HandleSignedURL: Bucket is empty in settings")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "S3 bucket not configured in provider settings",
			})
		}

		log.Printf("HandleSignedURL: Using S3 config - region: %s, endpoint: %s, usePathStyle: %v, bucket: %s",
			s3Settings.Region, s3Settings.Endpoint, s3Settings.UsePathStyle, s3Settings.Bucket)

		// Find both the access key and secret key for this provider
		var apiKeys []struct {
			ID      string `db:"id"`
			Key     string `db:"key"`
			KeyType string `db:"key_type"`
		}
		err = app.Dao().DB().NewQuery(
			"SELECT id, key, key_type FROM api_keys WHERE provider = {:providerId} AND (key_type = 'access_key' OR key_type = 'secret_key')",
		).Bind(dbx.Params{
			"providerId": s3ProviderId,
		}).All(&apiKeys)

		if err != nil {
			log.Println("HandleSignedURL: Failed to find API keys:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch API keys",
			})
		}

		// Extract access key and secret key from results
		var accessKey, secretKey string
		for _, key := range apiKeys {
			// Decrypt the key value using the crypto package
			decryptedBytes, err := crypto.Decrypt(key.Key, "") // Empty string as key since it uses env var
			if err != nil {
				log.Printf("HandleSignedURL: Failed to decrypt %s: %v", key.KeyType, err)
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to decrypt %s", key.KeyType),
				})
			}

			if key.KeyType == "access_key" {
				accessKey = string(decryptedBytes)
			} else if key.KeyType == "secret_key" {
				secretKey = string(decryptedBytes)
			}
		}

		if accessKey == "" {
			log.Println("HandleSignedURL: Access key not found")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Access key not found for provider",
			})
		}

		if secretKey == "" {
			log.Println("HandleSignedURL: Secret key not found")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Secret key not found for provider",
			})
		}

		// Create an AWS session
		sess, err := session.NewSession(&aws.Config{
			Region:           aws.String(s3Settings.Region),
			Endpoint:         aws.String(s3Settings.Endpoint),
			Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
			S3ForcePathStyle: aws.Bool(s3Settings.UsePathStyle),
		})
		if err != nil {
			log.Println("HandleSignedURL: Failed to create AWS session:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create AWS session",
			})
		}

		// Generate the signed URL
		s3Client := s3.New(sess)
		s3Req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(s3Settings.Bucket),
			Key:    aws.String(filePath),
		})
		signedURL, err := s3Req.Presign(15 * time.Minute)
		if err != nil {
			log.Println("HandleSignedURL: Failed to generate signed URL:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to generate signed URL",
			})
		}

		log.Println("HandleSignedURL: Successfully generated signed URL")
		return c.JSON(http.StatusOK, map[string]string{
			"signedUrl": signedURL,
		})
	}
}
