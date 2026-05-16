package models

import (
	"errors"
	"time"
)

// Task represents a household chore to be scheduled
type Task struct {
	ID            int64
	Name          string
	RoomID        int64  // FK to rooms.id
	Room          string // Room name this task belongs to (e.g., "Kitchen", "Bathroom")
	Effort        int    // 1-3 (1=quick, 2=medium, 3=long)
	FrequencyDays int
	LastCompleted *time.Time
	NextScheduled *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validation errors
var (
	ErrEmptyName        = errors.New("task name cannot be empty")
	ErrEmptyRoom        = errors.New("task room cannot be empty")
	ErrInvalidEffort    = errors.New("effort must be between 1 and 3")
	ErrInvalidFrequency = errors.New("frequency must be greater than 0")
)

// Validate checks if the task has valid field values
func (t *Task) Validate() error {
	if t.Name == "" {
		return ErrEmptyName
	}
	if t.Room == "" {
		return ErrEmptyRoom
	}
	if t.Effort < 1 || t.Effort > 3 {
		return ErrInvalidEffort
	}
	if t.FrequencyDays < 1 {
		return ErrInvalidFrequency
	}
	return nil
}

// DaysUntilDue returns the number of days until this task is due
// Returns 0 if the task is due today or overdue
// Returns -1 if NextScheduled is nil (unscheduled)
func (t *Task) DaysUntilDue() int {
	if t.NextScheduled == nil {
		return -1
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	due := time.Date(t.NextScheduled.Year(), t.NextScheduled.Month(), t.NextScheduled.Day(), 0, 0, 0, 0, time.UTC)

	days := int(due.Sub(today).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// IsOverdue returns true if the task is past its scheduled date
func (t *Task) IsOverdue() bool {
	if t.NextScheduled == nil {
		return false
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	due := time.Date(t.NextScheduled.Year(), t.NextScheduled.Month(), t.NextScheduled.Day(), 0, 0, 0, 0, time.UTC)

	return due.Before(today)
}

// DaysOverdue returns the number of days the task is overdue
// Returns 0 if not overdue
func (t *Task) DaysOverdue() int {
	if t.NextScheduled == nil {
		return 0
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	due := time.Date(t.NextScheduled.Year(), t.NextScheduled.Month(), t.NextScheduled.Day(), 0, 0, 0, 0, time.UTC)

	days := int(today.Sub(due).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// IsDueToday returns true if the task is due today
func (t *Task) IsDueToday() bool {
	if t.NextScheduled == nil {
		return false
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	due := time.Date(t.NextScheduled.Year(), t.NextScheduled.Month(), t.NextScheduled.Day(), 0, 0, 0, 0, time.UTC)

	return due.Equal(today)
}

// CalculateNextScheduled calculates the next scheduled date based on frequency
// If lastCompleted is nil, uses today as the base date
func (t *Task) CalculateNextScheduled() time.Time {
	var base time.Time
	if t.LastCompleted != nil {
		base = *t.LastCompleted
	} else {
		base = time.Now()
	}

	return base.AddDate(0, 0, t.FrequencyDays)
}

// Cleanliness status constants
const (
	CleanlinessClean     = "Clean"
	CleanlinessDue       = "Due"
	CleanlinessDirty     = "Dirty"
	CleanlinessVeryDirty = "Very Dirty"
	CleanlinessUnknown   = "Unknown"
)

// CleanlinessStatus returns the cleanliness status based on how long ago
// the task was last completed relative to its frequency.
func (t *Task) CleanlinessStatus() string {
	if t.LastCompleted == nil {
		return CleanlinessUnknown
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	lastDone := time.Date(t.LastCompleted.Year(), t.LastCompleted.Month(), t.LastCompleted.Day(), 0, 0, 0, 0, time.UTC)

	daysSinceCompleted := int(today.Sub(lastDone).Hours() / 24)
	ratio := float64(daysSinceCompleted) / float64(t.FrequencyDays)

	switch {
	case ratio < 1.0:
		return CleanlinessClean
	case ratio <= 1.5:
		return CleanlinessDue
	case ratio <= 2.0:
		return CleanlinessDirty
	default:
		return CleanlinessVeryDirty
	}
}
