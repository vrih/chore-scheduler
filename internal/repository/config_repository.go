package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
)

// ErrConfigNotFound is returned when a config key cannot be found
var ErrConfigNotFound = errors.New("config key not found")

// ConfigRepository defines the interface for config data access
type ConfigRepository interface {
	Get(key string) (string, error)
	Set(key, value string) error
	GetAll() ([]*models.Config, error)
	GetMaxDailyEffort() (int, error)
	SetMaxDailyEffort(effort int) error
}

type configRepository struct {
	db *db.DB
}

// NewConfigRepository creates a new ConfigRepository
func NewConfigRepository(database *db.DB) ConfigRepository {
	return &configRepository{db: database}
}

// Get retrieves a config value by key
func (r *configRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrConfigNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	return value, nil
}

// Set creates or updates a config value
func (r *configRepository) Set(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// GetAll retrieves all config entries
func (r *configRepository) GetAll() ([]*models.Config, error) {
	rows, err := r.db.Query("SELECT key, value FROM config ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("failed to get all config: %w", err)
	}
	defer rows.Close()

	var configs []*models.Config
	for rows.Next() {
		c := &models.Config{}
		if err := rows.Scan(&c.Key, &c.Value); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		configs = append(configs, c)
	}

	return configs, rows.Err()
}

// GetMaxDailyEffort retrieves the max_daily_effort config as an integer
func (r *configRepository) GetMaxDailyEffort() (int, error) {
	value, err := r.Get(models.ConfigKeyMaxDailyEffort)
	if err != nil {
		if err == ErrConfigNotFound {
			return models.DefaultMaxDailyEffort, nil
		}
		return 0, err
	}

	effort, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid max_daily_effort value: %w", err)
	}

	return effort, nil
}

// SetMaxDailyEffort sets the max_daily_effort config
func (r *configRepository) SetMaxDailyEffort(effort int) error {
	if effort < 1 {
		return errors.New("max_daily_effort must be at least 1")
	}
	return r.Set(models.ConfigKeyMaxDailyEffort, strconv.Itoa(effort))
}
