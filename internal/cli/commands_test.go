package cli

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
	"github.com/user/chore-scheduler/internal/repository"
	"github.com/user/chore-scheduler/internal/scheduler"
)

// setupTestCLI creates a CLI instance with an in-memory database for testing
func setupTestCLI(t *testing.T) *CLI {
	database, err := db.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, database.Initialize())

	cli := &CLI{
		db:             database,
		taskRepo:       repository.NewTaskRepository(database),
		completionRepo: repository.NewCompletionRepository(database),
		configRepo:     repository.NewConfigRepository(database),
		scheduledRepo:  repository.NewScheduledTaskRepository(database),
	}
	cli.scheduler = scheduler.NewScheduler(cli.taskRepo, cli.configRepo, cli.scheduledRepo)

	t.Cleanup(func() {
		database.Close()
	})

	return cli
}

// createTestTask creates a task for testing and returns it
func createTestTask(t *testing.T, cli *CLI, name, room string) *models.Task {
	task := &models.Task{
		Name:          name,
		Room:          room,
		Effort:        2,
		FrequencyDays: 7,
	}
	require.NoError(t, cli.taskRepo.Create(task))
	require.NoError(t, cli.scheduler.ScheduleTask(task))
	return task
}

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHandleCompleteMultiple_SingleTask(t *testing.T) {
	cli := setupTestCLI(t)
	task := createTestTask(t, cli, "Test Task", "Kitchen")

	output := captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{task.ID})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Completed 1 task(s)")
	assert.Contains(t, output, task.Name)

	// Verify task was completed
	updated, err := cli.taskRepo.Get(task.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.LastCompleted)
}

func TestHandleCompleteMultiple_MultipleTasks(t *testing.T) {
	cli := setupTestCLI(t)
	task1 := createTestTask(t, cli, "Task 1", "Kitchen")
	task2 := createTestTask(t, cli, "Task 2", "Bathroom")
	task3 := createTestTask(t, cli, "Task 3", "Living Room")

	output := captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{task1.ID, task2.ID, task3.ID})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Completed 3 task(s)")
	assert.Contains(t, output, task1.Name)
	assert.Contains(t, output, task2.Name)
	assert.Contains(t, output, task3.Name)

	// Verify all tasks were completed
	for _, id := range []int64{task1.ID, task2.ID, task3.ID} {
		updated, err := cli.taskRepo.Get(id)
		require.NoError(t, err)
		assert.NotNil(t, updated.LastCompleted)
	}
}

func TestHandleCompleteMultiple_MixedValidInvalid(t *testing.T) {
	cli := setupTestCLI(t)
	task1 := createTestTask(t, cli, "Valid Task 1", "Kitchen")
	task2 := createTestTask(t, cli, "Valid Task 2", "Bathroom")
	invalidID := int64(9999)

	output := captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{task1.ID, invalidID, task2.ID})
		// Should not return error because some tasks succeeded
		require.NoError(t, err)
	})

	// Should report successes
	assert.Contains(t, output, "Completed 2 task(s)")
	assert.Contains(t, output, task1.Name)
	assert.Contains(t, output, task2.Name)

	// Should report failure
	assert.Contains(t, output, "Failed to complete 1 task(s)")
	assert.Contains(t, output, "#9999")

	// Verify valid tasks were completed
	updated1, err := cli.taskRepo.Get(task1.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated1.LastCompleted)

	updated2, err := cli.taskRepo.Get(task2.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated2.LastCompleted)
}

func TestHandleCompleteMultiple_AllInvalid(t *testing.T) {
	cli := setupTestCLI(t)

	output := captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{9998, 9999})
		// Should return error because all tasks failed
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete any tasks")
	})

	// Should report failures
	assert.Contains(t, output, "Failed to complete 2 task(s)")
	assert.Contains(t, output, "#9998")
	assert.Contains(t, output, "#9999")
}

func TestHandleCompleteMultiple_CreatesCompletionRecords(t *testing.T) {
	cli := setupTestCLI(t)
	task := createTestTask(t, cli, "Task with Completion", "Kitchen")

	captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{task.ID})
		require.NoError(t, err)
	})

	// Verify completion record was created
	completions, err := cli.completionRepo.GetByTaskID(task.ID)
	require.NoError(t, err)
	assert.Len(t, completions, 1)
}

func TestHandleCompleteMultiple_ReschedulesTask(t *testing.T) {
	cli := setupTestCLI(t)
	task := createTestTask(t, cli, "Task to Reschedule", "Kitchen")

	// Get original schedule
	originalSchedule, err := cli.scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	require.Len(t, originalSchedule, 1)
	originalDate := originalSchedule[0].ScheduledDate

	captureOutput(func() {
		err := cli.handleCompleteMultiple([]int64{task.ID})
		require.NoError(t, err)
	})

	// Verify task was rescheduled to a later date
	newSchedule, err := cli.scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	require.Len(t, newSchedule, 1)
	assert.True(t, newSchedule[0].ScheduledDate.After(originalDate))
}

func TestBuildCompleteCommand_ParsesMultipleIDs(t *testing.T) {
	cli := setupTestCLI(t)
	cmd := cli.buildCompleteCommand()

	// Test that command accepts multiple arguments
	assert.Equal(t, "complete <id> [id...]", cmd.Use)

	// Verify MinimumNArgs(1) is set by checking Args function behavior
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err) // Should fail with 0 args

	err = cmd.Args(cmd, []string{"1"})
	assert.NoError(t, err) // Should succeed with 1 arg

	err = cmd.Args(cmd, []string{"1", "2", "3"})
	assert.NoError(t, err) // Should succeed with multiple args
}

func TestBuildCompleteCommand_InvalidIDFormat(t *testing.T) {
	cli := setupTestCLI(t)
	cmd := cli.buildCompleteCommand()

	// Set up command for execution
	cmd.SetArgs([]string{"invalid"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestBuildCompleteCommand_MixedValidInvalidFormat(t *testing.T) {
	cli := setupTestCLI(t)
	cmd := cli.buildCompleteCommand()

	// Set up command for execution with a mix of valid format and invalid format
	cmd.SetArgs([]string{"1", "notanumber", "3"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID: notanumber")
}
