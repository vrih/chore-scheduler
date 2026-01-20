package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_InMemory(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	assert.NotNil(t, db)
}

func TestInitialize_CreatesSchema(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	// Verify tables exist
	tables := []string{"tasks", "completions", "config", "scheduled_tasks", "schema_migrations"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestInitialize_CreatesIndexes(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	// Verify indexes exist
	indexes := []string{
		"idx_tasks_next_scheduled",
		"idx_tasks_last_completed",
		"idx_completions_task_id",
		"idx_completions_completed_at",
		"idx_scheduled_tasks_date",
		"idx_scheduled_tasks_task_id",
	}
	for _, idx := range indexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		require.NoError(t, err, "index %s should exist", idx)
		assert.Equal(t, idx, name)
	}
}

func TestInitialize_SetsDefaultConfig(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	var value string
	err = db.QueryRow("SELECT value FROM config WHERE key = 'max_daily_effort'").Scan(&value)
	require.NoError(t, err)
	assert.Equal(t, "10", value)
}

func TestInitialize_Idempotent(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize twice should not error
	err = db.Initialize()
	require.NoError(t, err)

	err = db.Initialize()
	require.NoError(t, err)

	// Verify only one default config entry
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM config WHERE key = 'max_daily_effort'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTasksTable_EnforcesConstraints(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	// Test effort constraint (must be 1-3)
	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 0, 7)")
	assert.Error(t, err, "effort < 1 should fail")

	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 4, 7)")
	assert.Error(t, err, "effort > 3 should fail")

	// Test frequency constraint (must be > 0)
	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 2, 0)")
	assert.Error(t, err, "frequency_days <= 0 should fail")

	// Valid insert should work
	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 2, 7)")
	assert.NoError(t, err)
}

func TestScheduledTasks_UniqueConstraint(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	// Create a task first
	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 2, 7)")
	require.NoError(t, err)

	// First scheduled_task should work
	_, err = db.Exec("INSERT INTO scheduled_tasks (task_id, scheduled_date) VALUES (1, '2024-01-15')")
	assert.NoError(t, err)

	// Duplicate should fail
	_, err = db.Exec("INSERT INTO scheduled_tasks (task_id, scheduled_date) VALUES (1, '2024-01-15')")
	assert.Error(t, err, "duplicate task_id + date should fail")

	// Different date should work
	_, err = db.Exec("INSERT INTO scheduled_tasks (task_id, scheduled_date) VALUES (1, '2024-01-16')")
	assert.NoError(t, err)
}

func TestForeignKeys_CascadeDelete(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Initialize()
	require.NoError(t, err)

	// Create a task
	_, err = db.Exec("INSERT INTO tasks (name, effort, frequency_days) VALUES ('test', 2, 7)")
	require.NoError(t, err)

	// Create completion and scheduled_task
	_, err = db.Exec("INSERT INTO completions (task_id) VALUES (1)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO scheduled_tasks (task_id, scheduled_date) VALUES (1, '2024-01-15')")
	require.NoError(t, err)

	// Delete task
	_, err = db.Exec("DELETE FROM tasks WHERE id = 1")
	require.NoError(t, err)

	// Verify cascade delete worked
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM completions WHERE task_id = 1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "completions should be deleted")

	err = db.QueryRow("SELECT COUNT(*) FROM scheduled_tasks WHERE task_id = 1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "scheduled_tasks should be deleted")
}
