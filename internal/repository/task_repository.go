package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
)

// ErrTaskNotFound is returned when a task cannot be found
var ErrTaskNotFound = errors.New("task not found")

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	Create(task *models.Task) error
	Get(id int64) (*models.Task, error)
	GetAll() ([]*models.Task, error)
	Update(task *models.Task) error
	Delete(id int64) error
	GetOverdue() ([]*models.Task, error)
	GetNeedingSchedule() ([]*models.Task, error)
	GetByRoom(room string) ([]*models.Task, error)
	GetAllRooms() ([]string, error)
}

type taskRepository struct {
	db *db.DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(database *db.DB) TaskRepository {
	return &taskRepository{db: database}
}

// Create inserts a new task into the database
func (r *taskRepository) Create(task *models.Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	result, err := r.db.Exec(`
		INSERT INTO tasks (name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.Name, task.Room, task.Effort, task.FrequencyDays, task.LastCompleted, task.NextScheduled, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get task ID: %w", err)
	}
	task.ID = id

	return nil
}

// Get retrieves a task by ID
func (r *taskRepository) Get(id int64) (*models.Task, error) {
	task := &models.Task{}
	err := r.db.QueryRow(`
		SELECT id, name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at
		FROM tasks WHERE id = ?
	`, id).Scan(
		&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
		&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// GetAll retrieves all tasks
func (r *taskRepository) GetAll() ([]*models.Task, error) {
	rows, err := r.db.Query(`
		SELECT id, name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at
		FROM tasks ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(
			&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
			&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// Update modifies an existing task
func (r *taskRepository) Update(task *models.Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	task.UpdatedAt = time.Now()

	result, err := r.db.Exec(`
		UPDATE tasks SET name = ?, room = ?, effort = ?, frequency_days = ?,
		last_completed = ?, next_scheduled = ?, updated_at = ?
		WHERE id = ?
	`, task.Name, task.Room, task.Effort, task.FrequencyDays,
		task.LastCompleted, task.NextScheduled, task.UpdatedAt, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// Delete removes a task by ID
func (r *taskRepository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// GetOverdue retrieves all tasks that are past their scheduled date
func (r *taskRepository) GetOverdue() ([]*models.Task, error) {
	today := time.Now().Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT id, name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at
		FROM tasks
		WHERE next_scheduled < ?
		ORDER BY next_scheduled
	`, today)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(
			&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
			&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetNeedingSchedule retrieves tasks that need to be scheduled
// This includes tasks without a next_scheduled date or tasks within the scheduling window
func (r *taskRepository) GetNeedingSchedule() ([]*models.Task, error) {
	// Get tasks that need scheduling: no next_scheduled or within 30 days
	futureDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT id, name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at
		FROM tasks
		WHERE next_scheduled IS NULL OR next_scheduled <= ?
		ORDER BY next_scheduled NULLS FIRST
	`, futureDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks needing schedule: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(
			&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
			&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetByRoom retrieves all tasks for a specific room
func (r *taskRepository) GetByRoom(room string) ([]*models.Task, error) {
	rows, err := r.db.Query(`
		SELECT id, name, room, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at
		FROM tasks
		WHERE room = ?
		ORDER BY id
	`, room)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by room: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(
			&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
			&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetAllRooms retrieves a list of all distinct rooms
func (r *taskRepository) GetAllRooms() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT room FROM tasks ORDER BY room
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms: %w", err)
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var room string
		if err := rows.Scan(&room); err != nil {
			return nil, fmt.Errorf("failed to scan room: %w", err)
		}
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}
