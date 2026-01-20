package models

import (
	"errors"
	"time"
)

// Completion represents a record of a task being completed
type Completion struct {
	ID          int64
	TaskID      int64
	CompletedAt time.Time
}

// ErrInvalidTaskID indicates the task ID is invalid
var ErrInvalidTaskID = errors.New("task ID must be positive")

// Validate checks if the completion has valid field values
func (c *Completion) Validate() error {
	if c.TaskID <= 0 {
		return ErrInvalidTaskID
	}
	return nil
}

// NewCompletion creates a new completion record for the given task
func NewCompletion(taskID int64) *Completion {
	return &Completion{
		TaskID:      taskID,
		CompletedAt: time.Now(),
	}
}
