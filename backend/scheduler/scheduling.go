// scheduler/scheduler.go

package scheduler

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/robfig/cron/v3"

	pbModels "github.com/pocketbase/pocketbase/models"

	"orbit/models"
	"orbit/scan/utils"
)

type ScanScheduler struct {
	Cron            *cron.Cron
	App             *pocketbase.PocketBase
	AnsibleBasePath string
}

func NewScanScheduler(app *pocketbase.PocketBase, ansibleBasePath string) *ScanScheduler {
	log.Printf("Creating new scan scheduler with ansible base path: %s", ansibleBasePath)
	c := cron.New(cron.WithSeconds())
	return &ScanScheduler{
		Cron:            c,
		App:             app,
		AnsibleBasePath: ansibleBasePath,
	}
}

func (s *ScanScheduler) Start() {
	log.Printf("Starting scan scheduler with ansible base path: %s", s.AnsibleBasePath)
	s.scheduleExistingScans()
	s.Cron.Start()
}

func (s *ScanScheduler) Stop() {
	s.Cron.Stop()
}

func (s *ScanScheduler) scheduleExistingScans() {
	log.Printf("Scheduling existing scans with ansible base path: %s", s.AnsibleBasePath)
	// Fetch scheduled scans from the database
	records, err := s.App.Dao().FindRecordsByExpr("scheduled_scans", nil)
	if err != nil {
		log.Println("Failed to retrieve scheduled scans:", err)
		return
	}

	for _, record := range records {
		schedule := recordToScheduledScan(record)
		s.scheduleScan(schedule)
	}
}

func (s *ScanScheduler) scheduleScan(schedule models.ScheduledScan) {
	// Skip if end date is in the past
	if !schedule.EndDate.IsZero() && schedule.EndDate.Before(time.Now()) {
		log.Printf("Skipping expired schedule %s", schedule.ID)
		return
	}

	// Generate cron expression based on schedule details
	cronExpr := generateCronExpression(schedule)
	if cronExpr == "" {
		log.Printf("Invalid schedule configuration for %s", schedule.ID)
		return
	}

	log.Printf("Scheduling scan %s with cron expression: %s", schedule.ID, cronExpr)

	entryID, err := s.Cron.AddFunc(cronExpr, func() {
		s.executeScan(schedule)
	})

	if err != nil {
		log.Printf("Failed to schedule scan %s: %v", schedule.ID, err)
	} else {
		log.Printf("Successfully scheduled scan %s with entry ID %d", schedule.ID, entryID)
	}
}

func (s *ScanScheduler) executeScan(schedule models.ScheduledScan) {
	log.Printf("Executing scheduled scan %s with ansible base path: %s", schedule.ID, s.AnsibleBasePath)

	// Check if the schedule is still valid
	record, err := s.App.Dao().FindRecordById("scheduled_scans", schedule.ID)
	if err != nil {
		log.Printf("Failed to find schedule %s: %v", schedule.ID, err)
		return
	}

	// Check if end date has passed
	endDate := record.GetDateTime("end_date")
	if !endDate.IsZero() && endDate.Time().Before(time.Now()) {
		log.Printf("Schedule %s has expired", schedule.ID)
		return
	}

	// Fetch the scan record
	scanRecord, err := s.App.Dao().FindRecordById("nuclei_scans", schedule.ScanID)
	if err != nil {
		log.Printf("Failed to find scan %s: %v", schedule.ScanID, err)
		return
	}

	// Update scan status to "Deploying"
	scanRecord.Set("status", "Deploying")
	scanRecord.Set("start_time", time.Now())
	if err := s.App.Dao().SaveRecord(scanRecord); err != nil {
		log.Printf("Failed to update scan status: %v", err)
		return
	}

	// Define paths using the scheduler's AnsibleBasePath
	playbookPath := filepath.Join(s.AnsibleBasePath, "scans", schedule.ScanID, "deploy.yml")
	logDir := filepath.Join(s.AnsibleBasePath, "scans", schedule.ScanID, "logs")
	yamlFile := filepath.Join(s.AnsibleBasePath, "scans", schedule.ScanID, "scan.yaml")
	inventoryPath := filepath.Join(s.AnsibleBasePath, "scans", schedule.ScanID, "inventory")

	log.Printf("Using ansible base path: %s for scan %s", s.AnsibleBasePath, schedule.ScanID)
	log.Printf("Playbook path: %s", playbookPath)
	log.Printf("Log directory: %s", logDir)
	log.Printf("YAML file: %s", yamlFile)
	log.Printf("Inventory path: %s", inventoryPath)

	// Execute Ansible playbook
	if err := utils.ExecuteAnsiblePlaybook(
		playbookPath,
		logDir,
		yamlFile,
		inventoryPath,
		s.AnsibleBasePath,
		s.App,
		schedule.ScanID,
	); err != nil {
		log.Printf("Failed to execute scan: %v", err)
		scanRecord.Set("status", "Failed")
		if err := s.App.Dao().SaveRecord(scanRecord); err != nil {
			log.Printf("Failed to save scan record: %v", err)
		}
		return
	}

	// Update scan status to "Running"
	scanRecord.Set("status", "Running")
	if err := s.App.Dao().SaveRecord(scanRecord); err != nil {
		log.Printf("Failed to save scan record: %v", err)
	}
}

func generateCronExpression(schedule models.ScheduledScan) string {
	if schedule.CronExpression != "" {
		return schedule.CronExpression
	}

	details := schedule.ScheduleDetails
	if details == nil {
		return ""
	}

	switch details.Frequency {
	case "daily":
		// Run at midnight every day
		return "0 0 * * *"
	case "weekly":
		if len(details.SelectedDays) == 0 {
			return ""
		}
		// Convert day names to cron day numbers
		dayMap := map[string]string{
			"sunday":    "0",
			"monday":    "1",
			"tuesday":   "2",
			"wednesday": "3",
			"thursday":  "4",
			"friday":    "5",
			"saturday":  "6",
		}
		days := make([]string, 0)
		for _, day := range details.SelectedDays {
			if cronDay, ok := dayMap[strings.ToLower(day)]; ok {
				days = append(days, cronDay)
			}
		}
		if len(days) == 0 {
			return ""
		}
		// Run at midnight on selected days
		return "0 0 * * " + strings.Join(days, ",")
	case "monthly":
		if details.MonthlyType == "date" && details.MonthlyDate > 0 {
			// Run at midnight on specific date of each month
			return fmt.Sprintf("0 0 %d * *", details.MonthlyDate)
		} else if details.MonthlyType == "day" && details.MonthlyDay != "" && details.MonthlyWeek != "" {
			// Convert day name to number
			dayMap := map[string]string{
				"sunday":    "0",
				"monday":    "1",
				"tuesday":   "2",
				"wednesday": "3",
				"thursday":  "4",
				"friday":    "5",
				"saturday":  "6",
			}
			day := dayMap[strings.ToLower(details.MonthlyDay)]
			if day == "" {
				return ""
			}

			// Convert week to number
			weekMap := map[string]string{
				"first":  "1",
				"second": "2",
				"third":  "3",
				"fourth": "4",
				"last":   "L",
			}
			week := weekMap[strings.ToLower(details.MonthlyWeek)]
			if week == "" {
				return ""
			}

			// Run at midnight on the specified day of the specified week
			if week == "L" {
				return fmt.Sprintf("0 0 * * %sL", day)
			}
			return fmt.Sprintf("0 0 * * %s#%s", day, week)
		}
	}

	return ""
}

func recordToScheduledScan(record *pbModels.Record) models.ScheduledScan {
	var scheduleDetails *models.ScheduleDetails
	if details := record.Get("schedule_details"); details != nil {
		// Try to convert the details to our struct
		if detailsMap, ok := details.(map[string]interface{}); ok {
			scheduleDetails = &models.ScheduleDetails{
				Type:         getString(detailsMap["type"]),
				Frequency:    getString(detailsMap["frequency"]),
				SelectedDays: convertToStringSlice(detailsMap["selectedDays"]),
				MonthlyType:  getString(detailsMap["monthlyType"]),
				MonthlyDate:  getInt(detailsMap["monthlyDate"]),
				MonthlyDay:   getString(detailsMap["monthlyDay"]),
				MonthlyWeek:  getString(detailsMap["monthlyWeek"]),
			}
		}
	}

	startDate := record.GetDateTime("start_date")
	endDate := record.GetDateTime("end_date")
	created := record.GetDateTime("created")

	return models.ScheduledScan{
		ID:              record.Id,
		ScanID:          record.GetString("scan_id"),
		Frequency:       record.GetString("frequency"),
		CronExpression:  record.GetString("cron_expression"),
		StartDate:       startDate.Time(),
		EndDate:         endDate.Time(),
		ScheduleDetails: scheduleDetails,
		Created:         created.Time(),
	}
}

// Helper functions for type conversion
func convertToStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

func getInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// StartScheduler starts the scheduler for periodic tasks
func StartScheduler(app *pocketbase.PocketBase) (*cron.Cron, error) {
	c := cron.New()

	// Add scheduled tasks
	if _, err := c.AddFunc("@every 1h", func() {
		if err := CalculateScanCosts(app); err != nil {
			log.Printf("Error calculating scan costs: %v", err)
		}
	}); err != nil {
		return nil, err
	}

	c.Start()
	return c, nil
}
