package repository

import (
	"fmt"
	"time"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
)

// ScheduledTaskRepository defines the interface for scheduled task data access
type ScheduledTaskRepository interface {
	Create(st *models.ScheduledTask) error
	Delete(id int64) error
	GetByDate(date time.Time) ([]*models.ScheduledTask, error)
	GetByTask(taskID int64) ([]*models.ScheduledTask, error)
	GetFromDate(date time.Time) ([]*models.ScheduledTask, error)
	ClearForTask(taskID int64) error
	GetDailyEffort(date time.Time) (int, error)
	ClearAll() error
	IsTaskScheduledOnDate(taskID int64, date time.Time) (bool, error)
}

type scheduledTaskRepository struct {
	db *db.DB
}

// NewScheduledTaskRepository creates a new ScheduledTaskRepository
func NewScheduledTaskRepository(database *db.DB) ScheduledTaskRepository {
	return &scheduledTaskRepository{db: database}
}

// Create inserts a new scheduled task entry
func (r *scheduledTaskRepository) Create(st *models.ScheduledTask) error {
	if st.CreatedAt.IsZero() {
		st.CreatedAt = time.Now()
	}

	// Normalize date to just the date portion
	dateStr := st.ScheduledDate.Format("2006-01-02")

	result, err := r.db.Exec(`
		INSERT INTO scheduled_tasks (task_id, scheduled_date, created_at)
		VALUES (?, ?, ?)
	`, st.TaskID, dateStr, st.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get scheduled task ID: %w", err)
	}
	st.ID = id

	return nil
}

// Delete removes a scheduled task entry
func (r *scheduledTaskRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM scheduled_tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}
	return nil
}

// GetByDate retrieves all scheduled tasks for a specific date
func (r *scheduledTaskRepository) GetByDate(date time.Time) ([]*models.ScheduledTask, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT id, task_id, scheduled_date, created_at
		FROM scheduled_tasks
		WHERE scheduled_date = ?
		ORDER BY id
	`, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		st := &models.ScheduledTask{}
		if err := rows.Scan(&st.ID, &st.TaskID, &st.ScheduledDate, &st.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		tasks = append(tasks, st)
	}

	return tasks, rows.Err()
}

// GetByTask retrieves all scheduled entries for a specific task
func (r *scheduledTaskRepository) GetByTask(taskID int64) ([]*models.ScheduledTask, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, scheduled_date, created_at
		FROM scheduled_tasks
		WHERE task_id = ?
		ORDER BY scheduled_date
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		st := &models.ScheduledTask{}
		if err := rows.Scan(&st.ID, &st.TaskID, &st.ScheduledDate, &st.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		tasks = append(tasks, st)
	}

	return tasks, rows.Err()
}

// ClearForTask removes all scheduled entries for a specific task
func (r *scheduledTaskRepository) ClearForTask(taskID int64) error {
	_, err := r.db.Exec("DELETE FROM scheduled_tasks WHERE task_id = ?", taskID)
	if err != nil {
		return fmt.Errorf("failed to clear scheduled tasks: %w", err)
	}
	return nil
}

// GetDailyEffort calculates the total effort for tasks scheduled on a specific date
func (r *scheduledTaskRepository) GetDailyEffort(date time.Time) (int, error) {
	dateStr := date.Format("2006-01-02")
	var effort int
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(t.effort), 0)
		FROM scheduled_tasks st
		JOIN tasks t ON st.task_id = t.id
		WHERE st.scheduled_date = ?
	`, dateStr).Scan(&effort)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily effort: %w", err)
	}
	return effort, nil
}

// ClearAll removes all scheduled task entries
func (r *scheduledTaskRepository) ClearAll() error {
	_, err := r.db.Exec("DELETE FROM scheduled_tasks")
	if err != nil {
		return fmt.Errorf("failed to clear all scheduled tasks: %w", err)
	}
	return nil
}

// IsTaskScheduledOnDate checks if a task is already scheduled for a specific date
func (r *scheduledTaskRepository) IsTaskScheduledOnDate(taskID int64, date time.Time) (bool, error) {
	dateStr := date.Format("2006-01-02")
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM scheduled_tasks
		WHERE task_id = ? AND scheduled_date = ?
	`, taskID, dateStr).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check task schedule: %w", err)
	}
	return count > 0, nil
}

// GetFromDate retrieves all scheduled tasks from a specific date onwards
func (r *scheduledTaskRepository) GetFromDate(date time.Time) ([]*models.ScheduledTask, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT id, task_id, scheduled_date, created_at
		FROM scheduled_tasks
		WHERE scheduled_date >= ?
		ORDER BY scheduled_date, id
	`, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled tasks from date: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		st := &models.ScheduledTask{}
		if err := rows.Scan(&st.ID, &st.TaskID, &st.ScheduledDate, &st.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		tasks = append(tasks, st)
	}

	return tasks, rows.Err()
}
