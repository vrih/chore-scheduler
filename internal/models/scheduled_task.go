package models

import (
	"time"
)

// ScheduledTask represents a task assigned to a specific date
type ScheduledTask struct {
	ID            int64
	TaskID        int64
	ScheduledDate time.Time
	CreatedAt     time.Time
}

// NewScheduledTask creates a new scheduled task entry
func NewScheduledTask(taskID int64, date time.Time) *ScheduledTask {
	// Normalize date to midnight UTC
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return &ScheduledTask{
		TaskID:        taskID,
		ScheduledDate: normalizedDate,
		CreatedAt:     time.Now(),
	}
}

// IsSameDate checks if this scheduled task is for the given date
func (st *ScheduledTask) IsSameDate(date time.Time) bool {
	d := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	sd := time.Date(st.ScheduledDate.Year(), st.ScheduledDate.Month(), st.ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)
	return d.Equal(sd)
}
