package db

import (
	"database/sql"
	"fmt"
)

// runMigrations executes all database migrations
func runMigrations(db *sql.DB) error {
	// Create migrations table to track applied migrations
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations := []struct {
		version int
		up      string
	}{
		{
			version: 1,
			up: `
				CREATE TABLE tasks (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					effort INTEGER NOT NULL CHECK(effort >= 1 AND effort <= 3),
					frequency_days INTEGER NOT NULL CHECK(frequency_days > 0),
					last_completed DATETIME,
					next_scheduled DATETIME,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX idx_tasks_next_scheduled ON tasks(next_scheduled);
				CREATE INDEX idx_tasks_last_completed ON tasks(last_completed);
			`,
		},
		{
			version: 2,
			up: `
				CREATE TABLE completions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					task_id INTEGER NOT NULL,
					completed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
				);

				CREATE INDEX idx_completions_task_id ON completions(task_id);
				CREATE INDEX idx_completions_completed_at ON completions(completed_at);
			`,
		},
		{
			version: 3,
			up: `
				CREATE TABLE config (
					key TEXT PRIMARY KEY,
					value TEXT NOT NULL
				);

				INSERT INTO config (key, value) VALUES ('max_daily_effort', '10');
			`,
		},
		{
			version: 4,
			up: `
				CREATE TABLE scheduled_tasks (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					task_id INTEGER NOT NULL,
					scheduled_date DATE NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
					UNIQUE(task_id, scheduled_date)
				);

				CREATE INDEX idx_scheduled_tasks_date ON scheduled_tasks(scheduled_date);
				CREATE INDEX idx_scheduled_tasks_task_id ON scheduled_tasks(task_id);
			`,
		},
		{
			version: 5,
			up: `
				ALTER TABLE tasks ADD COLUMN room TEXT NOT NULL DEFAULT 'Unassigned';
				CREATE INDEX idx_tasks_room ON tasks(room);
			`,
		},
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if _, err := db.Exec(m.up); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
		}

		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}
	}

	return nil
}

func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	return count > 0, nil
}
