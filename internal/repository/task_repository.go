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

const taskSelectColumns = `t.id, t.name, r.name, t.effort, t.frequency_days,
	t.last_completed, t.next_scheduled, t.created_at, t.updated_at, t.room_id`

// scanTask scans a row produced with taskSelectColumns into a Task.
func scanTask(s interface {
	Scan(dest ...interface{}) error
}) (*models.Task, error) {
	task := &models.Task{}
	if err := s.Scan(
		&task.ID, &task.Name, &task.Room, &task.Effort, &task.FrequencyDays,
		&task.LastCompleted, &task.NextScheduled, &task.CreatedAt, &task.UpdatedAt, &task.RoomID,
	); err != nil {
		return nil, err
	}
	return task, nil
}

// resolveRoomID returns the id of the room with the given name, creating the
// room row if it does not yet exist.
func (r *taskRepository) resolveRoomID(name string) (int64, error) {
	if _, err := r.db.Exec(
		"INSERT INTO rooms (name) VALUES (?) ON CONFLICT(name) DO NOTHING", name,
	); err != nil {
		return 0, fmt.Errorf("failed to ensure room: %w", err)
	}
	var id int64
	if err := r.db.QueryRow("SELECT id FROM rooms WHERE name = ?", name).Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to resolve room id: %w", err)
	}
	return id, nil
}

// Create inserts a new task into the database
func (r *taskRepository) Create(task *models.Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	roomID, err := r.resolveRoomID(task.Room)
	if err != nil {
		return err
	}
	task.RoomID = roomID

	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	result, err := r.db.Exec(`
		INSERT INTO tasks (name, room_id, effort, frequency_days, last_completed, next_scheduled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.Name, task.RoomID, task.Effort, task.FrequencyDays, task.LastCompleted, task.NextScheduled, task.CreatedAt, task.UpdatedAt)
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
	task, err := scanTask(r.db.QueryRow(`
		SELECT `+taskSelectColumns+`
		FROM tasks t JOIN rooms r ON r.id = t.room_id
		WHERE t.id = ?
	`, id))
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
		SELECT ` + taskSelectColumns + `
		FROM tasks t JOIN rooms r ON r.id = t.room_id
		ORDER BY t.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
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

	roomID, err := r.resolveRoomID(task.Room)
	if err != nil {
		return err
	}
	task.RoomID = roomID

	task.UpdatedAt = time.Now()

	result, err := r.db.Exec(`
		UPDATE tasks SET name = ?, room_id = ?, effort = ?, frequency_days = ?,
		last_completed = ?, next_scheduled = ?, updated_at = ?
		WHERE id = ?
	`, task.Name, task.RoomID, task.Effort, task.FrequencyDays,
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
		SELECT `+taskSelectColumns+`
		FROM tasks t JOIN rooms r ON r.id = t.room_id
		WHERE t.next_scheduled < ?
		ORDER BY t.next_scheduled
	`, today)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
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
		SELECT `+taskSelectColumns+`
		FROM tasks t JOIN rooms r ON r.id = t.room_id
		WHERE t.next_scheduled IS NULL OR t.next_scheduled <= ?
		ORDER BY t.next_scheduled NULLS FIRST
	`, futureDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks needing schedule: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetByRoom retrieves all tasks for a specific room
func (r *taskRepository) GetByRoom(room string) ([]*models.Task, error) {
	rows, err := r.db.Query(`
		SELECT `+taskSelectColumns+`
		FROM tasks t JOIN rooms r ON r.id = t.room_id
		WHERE r.name = ?
		ORDER BY t.id
	`, room)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by room: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetAllRooms retrieves a list of all distinct rooms
func (r *taskRepository) GetAllRooms() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT name FROM rooms ORDER BY name
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
