package models

import "time"

// ScheduleDetails represents the detailed scheduling configuration
type ScheduleDetails struct {
	Type         string   `json:"type"`
	Frequency    string   `json:"frequency"`
	SelectedDays []string `json:"selectedDays,omitempty"`
	MonthlyType  string   `json:"monthlyType,omitempty"`
	MonthlyDate  int      `json:"monthlyDate,omitempty"`
	MonthlyDay   string   `json:"monthlyDay,omitempty"`
	MonthlyWeek  string   `json:"monthlyWeek,omitempty"`
}

type ScheduleRequest struct {
	ScanID          string           `json:"scan_id"`
	Frequency       string           `json:"frequency"`
	CronExpression  string           `json:"cron_expression"`
	StartDate       time.Time        `json:"start_date"`
	EndDate         time.Time        `json:"end_date"`
	ScheduleDetails *ScheduleDetails `json:"schedule_details,omitempty"`
}

type ScheduledScan struct {
	ID              string           `json:"id"`
	ScanID          string           `json:"scan_id"`
	Frequency       string           `json:"frequency"`
	CronExpression  string           `json:"cron_expression"`
	StartDate       time.Time        `json:"start_date"`
	EndDate         time.Time        `json:"end_date"`
	ScheduleDetails *ScheduleDetails `json:"schedule_details,omitempty"`
	Created         time.Time        `json:"created"`
}
