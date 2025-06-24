package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/types"

	"bitor/utils/crypto"
)

// getS3Credentials retrieves and decrypts S3 credentials from the database
func getS3Credentials(app *pocketbase.PocketBase, providerID string) (accessKeyID string, secretAccessKey string, err error) {
	// Get access key for this provider
	accessKeyRecord, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "access_key",
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find access key: %w", err)
	}

	// Get secret key for this provider
	secretKeyRecord, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "secret_key",
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find secret key: %w", err)
	}

	// Decrypt the access key
	encryptedAccessKey := accessKeyRecord.GetString("key")
	decryptedAccessKey, err := crypto.Decrypt(encryptedAccessKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt access key: %w", err)
	}

	// Decrypt the secret key
	encryptedSecretKey := secretKeyRecord.GetString("key")
	decryptedSecretKey, err := crypto.Decrypt(encryptedSecretKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt secret key: %w", err)
	}

	return string(decryptedAccessKey), string(decryptedSecretKey), nil
}

// createS3Config creates an S3 config with the provided credentials and settings
func createS3Config(ctx context.Context, accessKeyID, secretAccessKey, region, endpoint string, usePathStyle bool) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Override endpoint if provided
	if endpoint != "" {
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	return cfg, nil
}

// TestS3Connection tests the S3 connection and ability to write to specific paths
func TestS3Connection(app *pocketbase.PocketBase, providerID string, testPath string) error {
	// Get provider record
	provider, err := app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return fmt.Errorf("failed to find provider: %w", err)
	}

	// Parse provider settings
	rawSettings := provider.Get("settings")
	var settings map[string]interface{}
	
	// Handle different types of settings storage
	switch v := rawSettings.(type) {
	case map[string]interface{}:
		settings = v
	case types.JsonRaw:
		if err := json.Unmarshal(v, &settings); err != nil {
			return fmt.Errorf("failed to parse provider settings: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(v), &settings); err != nil {
			return fmt.Errorf("failed to parse provider settings: %w", err)
		}
	default:
		return fmt.Errorf("provider settings are not properly configured (unsupported type: %T)", rawSettings)
	}
	
	region, ok := settings["region"].(string)
	if !ok || region == "" {
		return fmt.Errorf("region not configured")
	}
	
	endpoint, ok := settings["endpoint"].(string)
	if !ok || endpoint == "" {
		return fmt.Errorf("endpoint not configured")
	}
	
	bucket, ok := settings["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("bucket not configured")
	}
	
	usePathStyle, _ := settings["use_path_style"].(bool)

	// Get credentials
	accessKeyID, secretAccessKey, err := getS3Credentials(app, providerID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Create S3 client
	ctx := context.Background()
	cfg, err := createS3Config(ctx, accessKeyID, secretAccessKey, region, endpoint, usePathStyle)
	if err != nil {
		return fmt.Errorf("failed to create S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	// Test 1: Check if bucket exists and is accessible
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("bucket access test failed: %w", err)
	}

	// Test 2: Create a test file in the specified path
	testFileName := fmt.Sprintf("%s/test-file-%d.txt", strings.TrimPrefix(testPath, "/"), time.Now().Unix())
	testContent := fmt.Sprintf("Test file created on %s", time.Now().Format(time.RFC3339))
	
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(testFileName),
		Body:   bytes.NewReader([]byte(testContent)),
	})
	if err != nil {
		return fmt.Errorf("file upload test failed: %w", err)
	}

	// Test 3: Verify the file exists
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(testFileName),
	})
	if err != nil {
		return fmt.Errorf("file verification test failed: %w", err)
	}

	// Test 4: Delete the test file
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(testFileName),
	})
	if err != nil {
		return fmt.Errorf("file cleanup test failed: %w", err)
	}

	return nil
}

// ValidateCredentials validates the S3 credentials by attempting to list buckets
func ValidateCredentials(app *pocketbase.PocketBase, providerID string) error {
	// Get provider record
	provider, err := app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return fmt.Errorf("failed to find provider: %w", err)
	}

	// Parse provider settings
	rawSettings := provider.Get("settings")
	var settings map[string]interface{}
	
	// Handle different types of settings storage
	switch v := rawSettings.(type) {
	case map[string]interface{}:
		settings = v
	case types.JsonRaw:
		if err := json.Unmarshal(v, &settings); err != nil {
			return fmt.Errorf("failed to parse provider settings: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(v), &settings); err != nil {
			return fmt.Errorf("failed to parse provider settings: %w", err)
		}
	default:
		return fmt.Errorf("provider settings are not properly configured (unsupported type: %T)", rawSettings)
	}
	
	region, ok := settings["region"].(string)
	if !ok || region == "" {
		return fmt.Errorf("region not configured")
	}
	
	endpoint, ok := settings["endpoint"].(string)
	if !ok || endpoint == "" {
		return fmt.Errorf("endpoint not configured")
	}
	
	usePathStyle, _ := settings["use_path_style"].(bool)

	// Get credentials
	accessKeyID, secretAccessKey, err := getS3Credentials(app, providerID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Create S3 client
	ctx := context.Background()
	cfg, err := createS3Config(ctx, accessKeyID, secretAccessKey, region, endpoint, usePathStyle)
	if err != nil {
		return fmt.Errorf("failed to create S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	// Test credentials by listing buckets
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	return nil
} 