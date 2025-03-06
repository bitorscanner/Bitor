package services

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
)

// ScanEventService handles scan lifecycle events
type ScanEventService struct {
	app            *pocketbase.PocketBase
	findingManager *FindingManager
}

// NewScanEventService creates a new scan event service
func NewScanEventService(app *pocketbase.PocketBase, findingManager *FindingManager) *ScanEventService {
	return &ScanEventService{
		app:            app,
		findingManager: findingManager,
	}
}

// HandleScanFinished processes a scan completion event
func (s *ScanEventService) HandleScanFinished(scanID string) error {
	// Get scan details
	scan, err := s.app.Dao().FindRecordById("nuclei_scans", scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan: %v", err)
	}

	// Only update status if it's not a manual scan
	currentStatus := scan.GetString("status")
	if currentStatus != "Manual" {
		scan.Set("status", "Finished")
	}
	scan.Set("end_time", time.Now())

	if err := s.app.Dao().SaveRecord(scan); err != nil {
		log.Printf("Failed to update scan status: %v", err)
	}

	// Get finding summary and create rollup
	summary, err := s.findingManager.GetRollupSummary(scanID)
	if err != nil {
		return fmt.Errorf("failed to get finding summary: %v", err)
	}

	// Mark rollup notification as sent
	if err := s.findingManager.MarkRollupNotificationSent(scanID); err != nil {
		log.Printf("Failed to mark rollup notification as sent: %v", err)
	}

	log.Printf("Scan %s completed with summary: %v", scanID, summary)
	return nil
}
