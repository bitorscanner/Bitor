package scheduler

import (
	"log"
)

func (s *Scheduler) ScheduleScan(scanRecord *ScanRecord) error {
	if err := s.App.Dao().SaveRecord(scanRecord); err != nil {
		log.Printf("Failed to save scan record: %v", err)
		return err
	}
	return nil
}
