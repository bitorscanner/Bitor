package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/execute/configuration"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/pocketbase/pocketbase"
)

var (
	showLogsInTerminal bool
	dbBasePath         string // Store the absolute path to the database directory
)

// InitDBPath initializes the database path - call this when the app starts
func InitDBPath(app *pocketbase.PocketBase) {
	// Get the absolute path to the database directory
	if app.DataDir() != "" {
		dbBasePath = filepath.Clean(app.DataDir())
		if !filepath.IsAbs(dbBasePath) {
			if abs, err := filepath.Abs(dbBasePath); err == nil {
				dbBasePath = abs
			}
		}
	}
}

// realTimeLogger implements io.Writer to handle real-time log processing
type realTimeLogger struct {
	app        *pocketbase.PocketBase
	scanID     string
	buffer     bytes.Buffer
	mutex      sync.Mutex
	logEntries []interface{} // Store logs in memory
	lastUpdate time.Time
	isStderr   bool // Flag to indicate if this logger is for stderr
}

// bufferLogs stores logs in memory until we can write them to the database
func (l *realTimeLogger) bufferLogs(logEntry map[string]interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.logEntries = append(l.logEntries, logEntry)
}

// Write implements io.Writer interface
func (l *realTimeLogger) Write(p []byte) (n int, err error) {
	// Write to buffer
	n, err = l.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Process complete lines
	for {
		line, err := l.buffer.ReadString('\n')
		if err == io.EOF {
			// Put the incomplete line back into the buffer
			l.buffer.WriteString(line)
			break
		}
		if err != nil {
			return n, err
		}

		// Skip empty lines
		if len(line) == 0 || line == "\n" {
			continue
		}

		// Create log entry
		logEntry := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"content":   line,
			"type":      "stdout",
		}
		if l.isStderr {
			logEntry["type"] = "stderr"
		}

		// Buffer the log entry
		l.bufferLogs(logEntry)

		// If this is stderr or contains "ERROR!", force an immediate flush
		if l.isStderr || (len(line) >= 6 && line[:6] == "ERROR!") {
			if err := l.flushLogs(); err != nil {
				log.Printf("Failed to flush error logs: %v", err)
			}
			continue
		}

		// Update database every second or when we have a lot of entries
		if time.Since(l.lastUpdate) > time.Second || len(l.logEntries) > 100 {
			if err := l.flushLogs(); err != nil {
				log.Printf("Failed to flush logs: %v", err)
			}
		}
	}

	return n, nil
}

// flushLogs writes buffered logs to the database
func (l *realTimeLogger) flushLogs() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if len(l.logEntries) == 0 {
		return nil
	}

	// Store current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Could not get current working directory: %v", err)
	}

	// If we have a stored database path and we're in a different directory, temporarily change back
	if dbBasePath != "" && currentDir != filepath.Dir(dbBasePath) {
		if err := os.Chdir(filepath.Dir(dbBasePath)); err != nil {
			log.Printf("Warning: Could not change to database directory: %v", err)
		} else {
			// Ensure we change back when done
			defer func() {
				if err := os.Chdir(currentDir); err != nil {
					log.Printf("Warning: Could not restore working directory: %v", err)
				}
			}()
		}
	}

	// Create a copy of the logs we're about to flush
	logsToFlush := make([]interface{}, len(l.logEntries))
	copy(logsToFlush, l.logEntries)

	// Use a retry mechanism for database operations
	maxRetries := 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Get the scan record
		record, err := l.app.Dao().FindRecordById("nuclei_scans", l.scanID)
		if err != nil {
			lastErr = fmt.Errorf("failed to find scan record: %v", err)
			if i < maxRetries-1 {
				time.Sleep(time.Second * time.Duration(i+1))
			}
			continue
		}

		// Get existing logs
		var existingLogs []interface{}
		if logsValue := record.Get("ansible_logs"); logsValue != nil {
			// Try to unmarshal the logs regardless of type
			var logs []interface{}
			rawData, err := json.Marshal(logsValue)
			if err == nil {
				if err := json.Unmarshal(rawData, &logs); err != nil {
					log.Printf("Failed to unmarshal logs: %v", err)
					existingLogs = make([]interface{}, 0)
				} else {
					existingLogs = logs
				}
			}
		}
		if existingLogs == nil {
			existingLogs = make([]interface{}, 0)
		}

		// Append the logs we're trying to flush
		existingLogs = append(existingLogs, logsToFlush...)

		// Keep only the last 20000 log entries
		maxLogEntries := 20000
		if len(existingLogs) > maxLogEntries {
			existingLogs = existingLogs[len(existingLogs)-maxLogEntries:]
		}

		// Update the record with the combined logs
		record.Set("ansible_logs", existingLogs)
		if err := l.app.Dao().SaveRecord(record); err != nil {
			lastErr = fmt.Errorf("failed to save logs: %v", err)
			if i < maxRetries-1 {
				time.Sleep(time.Second * time.Duration(i+1))
			}
			continue
		}

		// Success - clear the flushed logs from memory
		l.logEntries = l.logEntries[len(logsToFlush):]
		l.lastUpdate = time.Now()
		return nil
	}

	// If we failed to save, keep the logs in memory
	log.Printf("Failed to save logs after %d retries", maxRetries)
	return lastErr
}

// ExecuteAnsiblePlaybook executes an ansible playbook
func ExecuteAnsiblePlaybook(playbookPath, logDir, extraVarsFile, inventoryPath, ansibleBasePath string, app *pocketbase.PocketBase, scanID string) error {
	// Get the absolute path of the working directory first
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	// Convert ansible base path to absolute if it isn't already
	if !filepath.IsAbs(ansibleBasePath) {
		ansibleBasePath = filepath.Join(workDir, ansibleBasePath)
	}

	// Convert all paths to absolute paths
	if !filepath.IsAbs(playbookPath) {
		playbookPath = filepath.Join(ansibleBasePath, playbookPath)
	}
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(ansibleBasePath, logDir)
	}
	if !filepath.IsAbs(extraVarsFile) {
		extraVarsFile = filepath.Join(ansibleBasePath, extraVarsFile)
	}
	if !filepath.IsAbs(inventoryPath) {
		inventoryPath = filepath.Join(ansibleBasePath, inventoryPath)
	}

	// Get the show logs flag from environment
	showLogsInTerminal = os.Getenv("SHOW_ANSIBLE_LOGS") == "true"

	// Set required environment variables with absolute paths
	os.Setenv("ANSIBLE_HOST_KEY_CHECKING", "false")
	os.Setenv("ANSIBLE_FORCE_COLOR", "true")
	os.Setenv("ANSIBLE_ACTION_WARNINGS", "false")
	os.Setenv("ANSIBLE_STDOUT_CALLBACK", "default")
	os.Setenv("ANSIBLE_RETRY_FILES_ENABLED", "false")
	os.Setenv("ANSIBLE_INVENTORY", inventoryPath)

	// Get the scan record to use its API key as the vault password
	record, err := app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to find scan record: %v", err)
	}
	scanApiKey := record.GetString("api_key")
	if scanApiKey == "" {
		return fmt.Errorf("scan API key not found")
	}

	// Create a temporary vault password file with absolute path
	vaultPassFile := filepath.Join(logDir, ".vault_pass")
	if err := os.MkdirAll(filepath.Dir(vaultPassFile), 0755); err != nil {
		return fmt.Errorf("failed to create vault pass directory: %v", err)
	}
	if err := os.WriteFile(vaultPassFile, []byte(scanApiKey), 0600); err != nil {
		return fmt.Errorf("failed to write vault password file: %v", err)
	}
	defer os.Remove(vaultPassFile)

	// Ensure ansible base path exists
	if _, err := os.Stat(ansibleBasePath); os.IsNotExist(err) {
		return fmt.Errorf("ansible base path does not exist: %s", ansibleBasePath)
	}

	// Change to the ansible directory
	if err := os.Chdir(ansibleBasePath); err != nil {
		return fmt.Errorf("failed to change to ansible directory: %v", err)
	}

	// Ensure we change back to the original directory when done
	defer func() {
		if err := os.Chdir(workDir); err != nil {
			log.Printf("Warning: Failed to restore original directory: %v", err)
		}
	}()

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with absolute path
	logFilePath := filepath.Join(logDir, "ansible.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Create real-time loggers for stdout and stderr
	stdoutLogger := &realTimeLogger{
		app:        app,
		scanID:     scanID,
		logEntries: make([]interface{}, 0),
		lastUpdate: time.Now(),
		isStderr:   false,
	}

	stderrLogger := &realTimeLogger{
		app:        app,
		scanID:     scanID,
		logEntries: make([]interface{}, 0),
		lastUpdate: time.Now(),
		isStderr:   true,
	}

	// Create multi-writers for stdout and stderr
	var stdoutWriter io.Writer = io.MultiWriter(logFile, stdoutLogger)
	var stderrWriter io.Writer = io.MultiWriter(logFile, stderrLogger)
	if showLogsInTerminal {
		stdoutWriter = io.MultiWriter(os.Stdout, logFile, stdoutLogger)
		stderrWriter = io.MultiWriter(os.Stderr, logFile, stderrLogger)
	}

	// Ensure logs are flushed when we return
	defer func() {
		if err := stdoutLogger.flushLogs(); err != nil {
			log.Printf("Failed to flush stdout logs: %v", err)
		}
		if err := stderrLogger.flushLogs(); err != nil {
			log.Printf("Failed to flush stderr logs: %v", err)
		}
	}()

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVarsFile: []string{fmt.Sprintf("@%s", extraVarsFile)},
		ExtraVars: map[string]interface{}{
			"scan_id": scanID,
		},
	}

	playbookCmd := playbook.NewAnsiblePlaybookCmd(
		playbook.WithPlaybooks(playbookPath),
		playbook.WithPlaybookOptions(ansiblePlaybookOptions),
	)

	// Create a writer that captures both stdout/stderr and errors
	errWriter := &bytes.Buffer{}
	combinedWriter := io.MultiWriter(stderrWriter, errWriter)

	exec := configuration.NewAnsibleWithConfigurationSettingsExecute(
		execute.NewDefaultExecute(
			execute.WithCmd(playbookCmd),
			execute.WithWrite(stdoutWriter),
			execute.WithWrite(combinedWriter),
			execute.WithErrorEnrich(playbook.NewAnsiblePlaybookErrorEnrich()),
		),
		configuration.WithAnsibleForceColor(),
		configuration.WithAnsibleForks(10),
		configuration.WithAnsibleInventory(inventoryPath),
		configuration.WithAnsibleHostKeyChecking(),
		configuration.WithoutAnsibleActionWarnings(),
		configuration.WithAnsibleVaultPasswordFile(vaultPassFile),
	)

	// Execute the playbook
	if err := exec.Execute(context.Background()); err != nil {
		log.Printf("Error executing Ansible playbook: %v", err)
		// Get any error output
		if errOutput := errWriter.String(); errOutput != "" {
			// Write error to stderr logger
			if _, writeErr := stderrLogger.Write([]byte(errOutput)); writeErr != nil {
				log.Printf("Failed to write to stderr logger: %v", writeErr)
			}
		}
		// Force flush any remaining logs
		if err := stdoutLogger.flushLogs(); err != nil {
			log.Printf("Failed to flush stdout logs: %v", err)
		}
		if err := stderrLogger.flushLogs(); err != nil {
			log.Printf("Failed to flush stderr logs: %v", err)
		}
		return fmt.Errorf("error executing command: %v", err)
	}

	return nil
}
