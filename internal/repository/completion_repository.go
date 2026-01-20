package repository

import (
	"fmt"
	"time"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
)

// CompletionRepository defines the interface for completion data access
type CompletionRepository interface {
	Create(completion *models.Completion) error
	GetByTaskID(taskID int64) ([]*models.Completion, error)
	GetRecent(limit int) ([]*models.Completion, error)
}

type completionRepository struct {
	db *db.DB
}

// NewCompletionRepository creates a new CompletionRepository
func NewCompletionRepository(database *db.DB) CompletionRepository {
	return &completionRepository{db: database}
}

// Create inserts a new completion record
func (r *completionRepository) Create(completion *models.Completion) error {
	if err := completion.Validate(); err != nil {
		return err
	}

	if completion.CompletedAt.IsZero() {
		completion.CompletedAt = time.Now()
	}

	result, err := r.db.Exec(`
		INSERT INTO completions (task_id, completed_at)
		VALUES (?, ?)
	`, completion.TaskID, completion.CompletedAt)
	if err != nil {
		return fmt.Errorf("failed to create completion: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get completion ID: %w", err)
	}
	completion.ID = id

	return nil
}

// GetByTaskID retrieves all completions for a specific task
func (r *completionRepository) GetByTaskID(taskID int64) ([]*models.Completion, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, completed_at
		FROM completions
		WHERE task_id = ?
		ORDER BY completed_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completions: %w", err)
	}
	defer rows.Close()

	var completions []*models.Completion
	for rows.Next() {
		c := &models.Completion{}
		if err := rows.Scan(&c.ID, &c.TaskID, &c.CompletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}
		completions = append(completions, c)
	}

	return completions, rows.Err()
}

// GetRecent retrieves the most recent completions across all tasks
func (r *completionRepository) GetRecent(limit int) ([]*models.Completion, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, completed_at
		FROM completions
		ORDER BY completed_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent completions: %w", err)
	}
	defer rows.Close()

	var completions []*models.Completion
	for rows.Next() {
		c := &models.Completion{}
		if err := rows.Scan(&c.ID, &c.TaskID, &c.CompletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}
		completions = append(completions, c)
	}

	return completions, rows.Err()
}
