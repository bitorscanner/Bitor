package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"orbit/utils/crypto"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// getProviderKeys retrieves and decrypts the API keys for a provider
func getProviderKeys(app *pocketbase.PocketBase, provider *models.Record) (map[string]string, error) {
	// Get API keys for the provider
	apiKeysCollection, err := app.Dao().FindCollectionByNameOrId("api_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to find api_keys collection: %w", err)
	}

	// Log the provider ID we're searching for
	log.Printf("Searching for API keys for provider: %s", provider.Id)

	// Get all API keys for this provider
	apiKeys, err := app.Dao().FindRecordsByFilter(
		apiKeysCollection.Id,
		"provider = {:provider}",
		"",
		0,
		0,
		dbx.Params{
			"provider": provider.Id,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find API keys: %w", err)
	}

	// Log how many keys were found
	log.Printf("Found %d API keys for provider %s", len(apiKeys), provider.Id)

	// Decrypt and store keys
	keys := make(map[string]string)
	encryptionKey := os.Getenv("API_ENCRYPTION_KEY")
	for _, apiKey := range apiKeys {
		// Get the key and its type
		encryptedKey := apiKey.GetString("key")
		keyType := apiKey.GetString("key_type")

		if encryptedKey != "" {
			decryptedBytes, err := crypto.Decrypt(encryptedKey, encryptionKey)
			if err != nil {
				continue
			}

			// Store the decrypted key with its type
			if provider.GetString("provider_type") == "s3" {
				if keyType == "access_key" {
					keys["access_key"] = string(decryptedBytes)
					log.Printf("Successfully decrypted access key for provider %s", provider.Id)
				} else if keyType == "secret_key" {
					keys["secret_key"] = string(decryptedBytes)
					log.Printf("Successfully decrypted secret key for provider %s", provider.Id)
				}
			} else if provider.GetString("provider_type") == "digitalocean" {
				// For DigitalOcean, we store the key as api_key regardless of key_type
				keys["api_key"] = string(decryptedBytes)
				log.Printf("Successfully decrypted API key for provider %s", provider.Id)
			}
		}
	}

	// Check for required keys based on provider type
	if provider.GetString("provider_type") == "s3" {
		if _, hasAccess := keys["access_key"]; !hasAccess {
			return nil, fmt.Errorf("access_key not found for provider %s", provider.Id)
		}
		if _, hasSecret := keys["secret_key"]; !hasSecret {
			return nil, fmt.Errorf("secret_key not found for provider %s", provider.Id)
		}
		log.Printf("Found both required keys for S3 provider")
	} else if provider.GetString("provider_type") == "digitalocean" {
		if _, hasKey := keys["api_key"]; !hasKey {
			return nil, fmt.Errorf("api_key not found for provider %s", provider.Id)
		}
		log.Printf("Found API key for DigitalOcean provider")
	}

	return keys, nil
}

func GenerateYAMLVars(app *pocketbase.PocketBase, scanID string) (string, error) {
	log.Printf("Starting YAML generation for scan ID: %s", scanID)

	// Get the existing scan record
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		log.Printf("Failed to find scan record: %v", err)
		return "", fmt.Errorf("failed to find record: %w", err)
	}
	log.Printf("Found scan record with ID: %s", record.Id)

	// Expand the necessary relations
	relations := []string{"client", "nuclei_interact", "scan_profile"}
	log.Printf("Expanding relations: %v", relations)
	expandErrors := app.Dao().ExpandRecord(record, relations, nil)
	if len(expandErrors) > 0 {
		for field, err := range expandErrors {
			log.Printf("Failed to expand relation %s: %v", field, err)
			return "", fmt.Errorf("failed to expand relation %s: %v", field, err)
		}
	}

	// Log expanded relations
	expanded := record.Expand()
	for relation := range expanded {
		log.Printf("Successfully expanded relation: %s", relation)
	}

	// Access the expanded "client" record
	clientRecord, ok := record.Expand()["client"].(*models.Record)
	if !ok {
		log.Printf("Client relation not found or invalid")
		return "", fmt.Errorf("client relation not found or invalid")
	}
	log.Printf("Found client record: %s", clientRecord.Id)

	// Access the expanded "nuclei_interact" record
	var interactURL, interactToken string
	if interactRecord, ok := record.Expand()["nuclei_interact"].(*models.Record); ok {
		log.Printf("Found nuclei interact record: %s", interactRecord.Id)
		interactURL = interactRecord.GetString("url")
		interactToken = interactRecord.GetString("token")
	} else {
		log.Printf("No nuclei interact record found, using default settings")
		interactURL = ""
		interactToken = ""
	}

	// Start building YAML content with client info
	yamlContent := fmt.Sprintf(`---
scan_id: "%s"
client: "%s"
client_hidden_name: "%s"
client_id: "%s"
interact_url: "%s"
interact_token: "%s"
api_key: "%s"`,
		scanID,
		clientRecord.GetString("name"),
		clientRecord.GetString("hidden_name"),
		clientRecord.GetString("id"),
		interactURL,
		interactToken,
		record.GetString("api_key"))

	// Get the scan's API key and encrypt it with Ansible vault
	scanApiKey := record.GetString("api_key")
	if scanApiKey == "" {
		log.Printf("Scan API key not found")
		return "", fmt.Errorf("scan API key not found")
	}

	// Create a temporary file for the vault password
	tmpVaultPass, err := os.CreateTemp("", "vault-pass-*")
	if err != nil {
		return "", fmt.Errorf("failed to create vault pass file: %w", err)
	}
	defer os.Remove(tmpVaultPass.Name())

	// Write the scan API key to the vault password file
	if err := os.WriteFile(tmpVaultPass.Name(), []byte(scanApiKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write vault pass file: %w", err)
	}

	// Access the expanded "scan_profile" record
	scanProfileRecord, ok := record.Expand()["scan_profile"].(*models.Record)
	if !ok {
		log.Printf("Scan profile relation not found or invalid")
		return "", fmt.Errorf("scan profile relation not found or invalid")
	}
	log.Printf("Found scan profile record: %s", scanProfileRecord.Id)

	// Expand the provider relations for the scan profile
	log.Printf("Expanding provider relations for scan profile")
	providerRelations := []string{"vm_provider", "state_bucket", "scan_bucket"}
	expandErrors = app.Dao().ExpandRecord(scanProfileRecord, providerRelations, nil)
	if len(expandErrors) > 0 {
		for field, err := range expandErrors {
			log.Printf("Failed to expand relation %s: %v", field, err)
			return "", fmt.Errorf("failed to expand relation %s: %v", field, err)
		}
	}

	// Log expanded relations
	expanded = scanProfileRecord.Expand()
	for relation := range expanded {
		log.Printf("Successfully expanded scan profile relation: %s", relation)
	}

	// Get the VM provider record
	vmProvider, ok := scanProfileRecord.Expand()["vm_provider"].(*models.Record)
	if !ok {
		log.Printf("VM provider relation not found or invalid")
		return "", fmt.Errorf("vm provider relation not found or invalid")
	}
	log.Printf("Found VM provider: %s with type: %s", vmProvider.Id, vmProvider.GetString("provider_type"))

	// Get provider keys for the VM provider
	providerKeys, err := getProviderKeys(app, vmProvider)
	if err != nil {
		log.Printf("Failed to get provider keys: %v", err)
		return "", fmt.Errorf("failed to get provider keys: %v", err)
	}
	log.Printf("Successfully retrieved provider keys for VM provider")

	// Only encrypt keys if this is an S3 provider
	if vmProvider.GetString("provider_type") == "s3" {
		// Encrypt access key
		cmdAccess := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "access_key")
		cmdAccess.Stdin = bytes.NewBufferString(providerKeys["access_key"])
		var outAccess bytes.Buffer
		var errAccess bytes.Buffer
		cmdAccess.Stdout = &outAccess
		cmdAccess.Stderr = &errAccess
		if err := cmdAccess.Run(); err != nil {
			log.Printf("Error encrypting access key: %v", err)
			return "", fmt.Errorf("failed to encrypt access key: %w", err)
		}

		// Clean up the vault output to only include the encrypted content
		encryptedAccessKey := cleanVaultOutput(outAccess.String())

		// Encrypt secret key
		cmdSecret := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "secret_key")
		cmdSecret.Stdin = bytes.NewBufferString(providerKeys["secret_key"])
		var outSecret bytes.Buffer
		var errSecret bytes.Buffer
		cmdSecret.Stdout = &outSecret
		cmdSecret.Stderr = &errSecret
		if err := cmdSecret.Run(); err != nil {
			log.Printf("Error encrypting secret key: %v", err)
			return "", fmt.Errorf("failed to encrypt secret key: %w", err)
		}

		// Clean up the vault output to only include the encrypted content
		encryptedSecretKey := cleanVaultOutput(outSecret.String())

		// Add both encrypted keys to the YAML content
		yamlContent += "\n" + encryptedAccessKey + "\n" + encryptedSecretKey
	}

	// Initialize s3 section
	var s3Content string

	// Process state bucket provider
	if stateBucketProvider, ok := scanProfileRecord.Expand()["state_bucket"].(*models.Record); ok {
		log.Printf("Found state bucket provider: %s with type: %s and ID: %s",
			stateBucketProvider.GetString("name"),
			stateBucketProvider.GetString("provider_type"),
			stateBucketProvider.Id)
		if stateBucketProvider.GetString("provider_type") == "s3" {
			log.Printf("Processing state bucket provider: %s", stateBucketProvider.Id)

			// Get and decrypt keys
			keys, err := getProviderKeys(app, stateBucketProvider)
			if err != nil {
				return "", fmt.Errorf("failed to get provider keys: %w", err)
			}

			settings := stateBucketProvider.Get("settings")
			var settingsMap map[string]any
			var parseError bool

			// Convert JsonRaw to string first
			settingsStr := fmt.Sprintf("%s", settings)
			if err := json.Unmarshal([]byte(settingsStr), &settingsMap); err != nil {
				log.Printf("Failed to parse settings JSON: %v", err)
				parseError = true
			}

			if parseError {
				log.Printf("Failed to parse settings, using empty map")
				settingsMap = make(map[string]any)
			} else {
				log.Printf("Successfully parsed settings")
			}

			if !parseError && settingsMap != nil {
				bucket, _ := settingsMap["bucket"].(string)
				endpoint, _ := settingsMap["endpoint"].(string)
				region, _ := settingsMap["region"].(string)
				usePathStyle, _ := settingsMap["use_path_style"].(bool)
				statefilePath, _ := settingsMap["statefile_path"].(string)

				// Get the scan's API key for vault encryption
				scanApiKey := record.GetString("api_key")
				if scanApiKey == "" {
					return "", fmt.Errorf("scan API key not found")
				}

				// Create a temporary file for the vault password
				tmpVaultPass, err := os.CreateTemp("", "vault-pass-*")
				if err != nil {
					return "", fmt.Errorf("failed to create vault pass file: %w", err)
				}
				defer os.Remove(tmpVaultPass.Name())

				// Write the scan API key to the vault password file
				if err := os.WriteFile(tmpVaultPass.Name(), []byte(scanApiKey), 0600); err != nil {
					return "", fmt.Errorf("failed to write vault pass file: %w", err)
				}

				// Log the raw decrypted keys before encryption
				log.Printf("Raw decrypted access_key before vault encryption: %s", keys["access_key"])
				log.Printf("Raw decrypted secret_key before vault encryption: %s", keys["secret_key"])

				// Encrypt access key
				cmdAccess := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "access_key")
				cmdAccess.Stdin = bytes.NewBufferString(keys["access_key"])
				var outAccess bytes.Buffer
				var errAccess bytes.Buffer
				cmdAccess.Stdout = &outAccess
				cmdAccess.Stderr = &errAccess
				if err := cmdAccess.Run(); err != nil {
					log.Printf("Error encrypting access key: %v", err)
					return "", fmt.Errorf("failed to encrypt access key: %w", err)
				}

				// Clean up the vault output to only include the encrypted content
				encryptedAccessKey := cleanVaultOutput(outAccess.String())

				// Encrypt secret key
				cmdSecret := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "secret_key")
				cmdSecret.Stdin = bytes.NewBufferString(keys["secret_key"])
				var outSecret bytes.Buffer
				var errSecret bytes.Buffer
				cmdSecret.Stdout = &outSecret
				cmdSecret.Stderr = &errSecret
				if err := cmdSecret.Run(); err != nil {
					log.Printf("Error encrypting secret key: %v", err)
					return "", fmt.Errorf("failed to encrypt secret key: %w", err)
				}

				// Clean up the vault output to only include the encrypted content
				encryptedSecretKey := cleanVaultOutput(outSecret.String())

				s3Content = fmt.Sprintf(`
s3:
  state:
    bucket: "%s"
    endpoint: "%s"
    region: "%s"
    use_path_style: %t
    path: "%s"
    provider_id: "%s"
    access_key: !vault |
%s
    secret_key: !vault |
%s`,
					bucket,
					endpoint,
					region,
					usePathStyle,
					statefilePath,
					stateBucketProvider.Id,
					indentVaultContent(encryptedAccessKey),
					indentVaultContent(encryptedSecretKey))
				log.Printf("Added state bucket to s3Content")
			}
		} else {
			log.Printf("State bucket provider is not of type s3")
		}
	} else {
		log.Printf("No state bucket provider found")
	}

	// Process scan bucket provider
	if scanBucketProvider, ok := scanProfileRecord.Expand()["scan_bucket"].(*models.Record); ok {
		log.Printf("Found scan bucket provider: %s with type: %s and ID: %s",
			scanBucketProvider.GetString("name"),
			scanBucketProvider.GetString("provider_type"),
			scanBucketProvider.Id)
		if scanBucketProvider.GetString("provider_type") == "s3" {
			log.Printf("Processing scan bucket provider: %s", scanBucketProvider.Id)

			// Get and decrypt keys
			keys, err := getProviderKeys(app, scanBucketProvider)
			if err != nil {
				return "", fmt.Errorf("failed to get provider keys: %w", err)
			}

			settings := scanBucketProvider.Get("settings")
			var settingsMap map[string]any
			var parseError bool

			// Convert JsonRaw to string first
			settingsStr := fmt.Sprintf("%s", settings)
			if err := json.Unmarshal([]byte(settingsStr), &settingsMap); err != nil {
				log.Printf("Failed to parse settings JSON: %v", err)
				parseError = true
			}

			if parseError {
				log.Printf("Failed to parse settings, using empty map")
				settingsMap = make(map[string]any)
			} else {
				log.Printf("Successfully parsed settings")
			}

			if !parseError && settingsMap != nil {
				bucket, _ := settingsMap["bucket"].(string)
				endpoint, _ := settingsMap["endpoint"].(string)
				region, _ := settingsMap["region"].(string)
				usePathStyle, _ := settingsMap["use_path_style"].(bool)
				scansPath, _ := settingsMap["scans_path"].(string)

				// Get the scan's API key for vault encryption
				scanApiKey := record.GetString("api_key")
				if scanApiKey == "" {
					return "", fmt.Errorf("scan API key not found")
				}

				// Create a temporary file for the vault password
				tmpVaultPass, err := os.CreateTemp("", "vault-pass-*")
				if err != nil {
					return "", fmt.Errorf("failed to create vault pass file: %w", err)
				}
				defer os.Remove(tmpVaultPass.Name())

				// Write the scan API key to the vault password file
				if err := os.WriteFile(tmpVaultPass.Name(), []byte(scanApiKey), 0600); err != nil {
					return "", fmt.Errorf("failed to write vault pass file: %w", err)
				}

				// Encrypt access key
				cmdAccess := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "access_key")
				cmdAccess.Stdin = bytes.NewBufferString(keys["access_key"])
				var outAccess bytes.Buffer
				var errAccess bytes.Buffer
				cmdAccess.Stdout = &outAccess
				cmdAccess.Stderr = &errAccess
				if err := cmdAccess.Run(); err != nil {
					log.Printf("Error encrypting access key: %v", err)
					return "", fmt.Errorf("failed to encrypt access key: %w", err)
				}

				// Clean up the vault output to only include the encrypted content
				encryptedAccessKey := cleanVaultOutput(outAccess.String())

				// Encrypt secret key
				cmdSecret := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "secret_key")
				cmdSecret.Stdin = bytes.NewBufferString(keys["secret_key"])
				var outSecret bytes.Buffer
				var errSecret bytes.Buffer
				cmdSecret.Stdout = &outSecret
				cmdSecret.Stderr = &errSecret
				if err := cmdSecret.Run(); err != nil {
					log.Printf("Error encrypting secret key: %v", err)
					return "", fmt.Errorf("failed to encrypt secret key: %w", err)
				}

				// Clean up the vault output to only include the encrypted content
				encryptedSecretKey := cleanVaultOutput(outSecret.String())

				if s3Content == "" {
					s3Content = fmt.Sprintf(`
s3:
  scan:
    bucket: "%s"
    endpoint: "%s"
    region: "%s"
    use_path_style: %t
    path: "%s"
    provider_id: "%s"
    access_key: !vault |
%s
    secret_key: !vault |
%s`,
						bucket,
						endpoint,
						region,
						usePathStyle,
						scansPath,
						scanBucketProvider.Id,
						indentVaultContent(encryptedAccessKey),
						indentVaultContent(encryptedSecretKey))
				} else {
					s3Content += fmt.Sprintf(`
  scan:
    bucket: "%s"
    endpoint: "%s"
    region: "%s"
    use_path_style: %t
    path: "%s"
    provider_id: "%s"
    access_key: !vault |
%s
    secret_key: !vault |
%s`,
						bucket,
						endpoint,
						region,
						usePathStyle,
						scansPath,
						scanBucketProvider.Id,
						indentVaultContent(encryptedAccessKey),
						indentVaultContent(encryptedSecretKey))
				}
				log.Printf("Added scan bucket to s3Content")
			}
		} else {
			log.Printf("Scan bucket provider is not of type s3")
		}
	} else {
		log.Printf("No scan bucket provider found")
	}

	// Add s3 section if it exists
	if s3Content != "" {
		log.Printf("Adding s3 section to YAML content")
		yamlContent += "\n" + s3Content
	} else {
		log.Printf("No s3 content to add")
	}

	// Process VM provider
	if vmProvider, ok := scanProfileRecord.Expand()["vm_provider"].(*models.Record); ok {
		log.Printf("Processing VM provider: %s", vmProvider.Id)

		// Get and decrypt keys
		keys, err := getProviderKeys(app, vmProvider)
		if err != nil {
			log.Printf("Failed to get VM provider keys: %v", err)
		} else {
			providerType := vmProvider.GetString("provider_type")
			settings := vmProvider.Get("settings")
			var settingsMap map[string]any
			var parseError bool

			// Handle settings parsing
			settingsStr := fmt.Sprintf("%s", settings)
			if err := json.Unmarshal([]byte(settingsStr), &settingsMap); err != nil {
				log.Printf("Failed to parse settings JSON: %v", err)
				parseError = true
			}

			if parseError {
				log.Printf("Failed to parse settings, using empty map")
				settingsMap = make(map[string]any)
			} else {
				log.Printf("Successfully parsed settings")
			}

			switch providerType {
			case "aws":
				region := ""
				accountID := ""
				if settingsMap != nil {
					region, _ = settingsMap["region"].(string)
					accountID, _ = settingsMap["account_id"].(string)
				}
				yamlContent += fmt.Sprintf(`

vm:
  provider_service: "AWS"
  account_id: "%s"
provider:
  name: "aws"
  region: "%s"
  api_key: "%s"
  secret_key: "%s"`,
					accountID,
					region,
					keys["access_key"],
					keys["secret_key"])

			case "digitalocean":
				region := ""
				doProject := ""
				var tagString string

				if settingsMap != nil {
					region, _ = settingsMap["region"].(string)

					// Handle project ID
					if projectRaw, ok := settingsMap["do_project"]; ok {
						if doProject, ok = projectRaw.(string); !ok {
							log.Printf("Project value is not a string")
						}
					}

					// Handle tags properly
					if tagsRaw, ok := settingsMap["tags"]; ok {
						switch v := tagsRaw.(type) {
						case string:
							if v != "" {
								tagString = v
							}
						case []interface{}:
							var validTags []string
							for _, tag := range v {
								if str, ok := tag.(string); ok && str != "" {
									validTags = append(validTags, str)
								}
							}
							if len(validTags) > 0 {
								tagString = strings.Join(validTags, ",")
							}
						}
					}
				}

				// Get and verify the decrypted API key
				apiKey, ok := keys["api_key"]
				if !ok {
					return "", fmt.Errorf("api_key not found for provider")
				}

				// Get the scan's API key for vault encryption
				scanApiKey := record.GetString("api_key")
				if scanApiKey == "" {
					return "", fmt.Errorf("scan API key not found")
				}

				// Get the VM size from the scan profile
				vmSize := scanProfileRecord.GetString("vm_size")
				if vmSize == "" {
					return "", fmt.Errorf("vm_size not found in scan profile")
				}

				// Create a temporary file for the vault password
				tmpVaultPass, err := os.CreateTemp("", "vault-pass-*")
				if err != nil {
					return "", fmt.Errorf("failed to create vault pass file: %w", err)
				}
				defer os.Remove(tmpVaultPass.Name())

				// Write the scan API key to the vault password file
				if err := os.WriteFile(tmpVaultPass.Name(), []byte(scanApiKey), 0600); err != nil {
					return "", fmt.Errorf("failed to write vault pass file: %w", err)
				}

				// Run ansible-vault to encrypt the value
				cmd := exec.Command("ansible-vault", "encrypt_string", "--vault-password-file", tmpVaultPass.Name(), "--stdin-name", "provider_key")
				cmd.Stdin = bytes.NewBufferString(apiKey)
				var out bytes.Buffer
				cmd.Stdout = &out
				if err := cmd.Run(); err != nil {
					// Fallback to unencrypted key
					vmConfig := fmt.Sprintf(`
vm:
  provider_service: "DigitalOcean"
  provider_key: "%s"
  do_project: "%s"
  do_region: "%s"
  do_size: "%s"`,
						apiKey,
						doProject,
						region,
						vmSize)

					// Add tags only if they exist
					if tagString != "" {
						vmConfig = strings.Replace(vmConfig, `do_size:`, fmt.Sprintf(`tags: "%s"
  do_size:`, tagString), 1)
					}
					yamlContent += vmConfig
				} else {
					encryptedKey := out.String()
					// Remove the "provider_key: " prefix from the encrypted output
					encryptedKey = strings.TrimPrefix(encryptedKey, "provider_key: ")
					vmConfig := fmt.Sprintf(`
vm:
  provider_service: "DigitalOcean"
  provider_key: !vault |
%s
  do_project: "%s"
  do_region: "%s"
  do_size: "%s"`,
						indentVaultContent(encryptedKey),
						doProject,
						region,
						vmSize)

					// Add tags only if they exist
					if tagString != "" {
						vmConfig = strings.Replace(vmConfig, `do_size:`, fmt.Sprintf(`tags: "%s"
  do_size:`, tagString), 1)
					}
					yamlContent += vmConfig
				}
			}
		}
	}

	log.Printf("Successfully generated YAML content")
	return yamlContent, nil
}

// indentVaultContent properly indents vault-encrypted content for YAML
func indentVaultContent(vaultContent string) string {
	// Remove the "api_key: " or "secret_key: " prefix if present
	vaultContent = strings.TrimPrefix(vaultContent, "api_key: ")
	vaultContent = strings.TrimPrefix(vaultContent, "secret_key: ")

	// Split the content into lines
	lines := strings.Split(vaultContent, "\n")

	// Remove empty lines and the "!vault |" line
	var contentLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, "!vault |") {
			// Indent each line with 8 spaces (6 for YAML structure + 2 for vault content)
			contentLines = append(contentLines, "      "+trimmed)
		}
	}

	return strings.Join(contentLines, "\n")
}

// cleanVaultOutput removes any debug output and formats the vault content correctly
func cleanVaultOutput(output string) string {
	// Split the output into lines
	lines := strings.Split(output, "\n")

	var cleanedLines []string
	inVault := false

	for _, line := range lines {
		// Skip debug output and empty lines
		if strings.Contains(line, "ansible-vault") ||
			strings.Contains(line, "config file") ||
			strings.Contains(line, "python version") ||
			strings.Contains(line, "jinja version") ||
			strings.Contains(line, "libyaml") ||
			strings.Contains(line, "No config file found") ||
			strings.Contains(line, "encrypt_vault_id") ||
			strings.TrimSpace(line) == "" {
			continue
		}

		// Start collecting vault content when we see the header
		if strings.Contains(line, "$ANSIBLE_VAULT") {
			inVault = true
			cleanedLines = append(cleanedLines, line)
			continue
		}

		// Only include lines while we're in the vault content
		if inVault {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}
