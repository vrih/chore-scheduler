package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
	"github.com/user/chore-scheduler/internal/repository"
)

func setupTestDB(t *testing.T) *db.DB {
	database, err := db.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, database.Initialize())
	return database
}

func TestCalculatePriority(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     *models.Task
		expected float64
	}{
		{
			name:     "unscheduled task",
			task:     &models.Task{NextScheduled: nil},
			expected: 50,
		},
		{
			name:     "due today",
			task:     &models.Task{NextScheduled: &today},
			expected: 1000,
		},
		{
			name: "due tomorrow",
			task: &models.Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 1)
				return &t
			}()},
			expected: 100, // 100 / 1
		},
		{
			name: "due in 5 days",
			task: &models.Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 5)
				return &t
			}()},
			expected: 20, // 100 / 5
		},
		{
			name: "overdue by 1 day",
			task: &models.Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -1)
				return &t
			}()},
			expected: 1001, // 1000 + 1
		},
		{
			name: "overdue by 5 days",
			task: &models.Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -5)
				return &t
			}()},
			expected: 1005, // 1000 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := CalculatePriority(tt.task)
			assert.Equal(t, tt.expected, priority)
		})
	}
}

func TestSortByPriority(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	inFiveDays := today.AddDate(0, 0, 5)

	tasks := []*models.Task{
		{ID: 1, NextScheduled: &inFiveDays},   // priority 20
		{ID: 2, NextScheduled: &yesterday},    // priority 1001 (overdue)
		{ID: 3, NextScheduled: nil},           // priority 50
		{ID: 4, NextScheduled: &today},        // priority 1000
	}

	sorted := SortByPriority(tasks)

	// Should be sorted: overdue, due today, unscheduled, future
	assert.Equal(t, int64(2), sorted[0].ID) // overdue - highest priority
	assert.Equal(t, int64(4), sorted[1].ID) // due today
	assert.Equal(t, int64(3), sorted[2].ID) // unscheduled
	assert.Equal(t, int64(1), sorted[3].ID) // future - lowest priority
}

func TestScheduler_BasicScheduling(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	// Set max effort
	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	// Create tasks
	tasks := []*models.Task{
		{Name: "Easy task", Room: "Kitchen", Effort: 1, FrequencyDays: 2},
		{Name: "Medium task", Room: "Bathroom", Effort: 2, FrequencyDays: 3},
		{Name: "Hard task", Room: "Kitchen", Effort: 3, FrequencyDays: 5},
	}

	for _, task := range tasks {
		require.NoError(t, taskRepo.Create(task))
	}

	// Run scheduler
	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	err := scheduler.Schedule()
	require.NoError(t, err)

	// Verify all tasks are scheduled
	for _, task := range tasks {
		scheduled, err := scheduledRepo.GetByTask(task.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, scheduled, "task %d should be scheduled", task.ID)
	}
}

func TestScheduler_RespectsEffortLimits(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	// Set low max effort
	require.NoError(t, configRepo.SetMaxDailyEffort(3))

	// Create tasks that together exceed daily limit
	tasks := []*models.Task{
		{Name: "Task 1", Room: "Kitchen", Effort: 2, FrequencyDays: 7},
		{Name: "Task 2", Room: "Bathroom", Effort: 2, FrequencyDays: 7},
		{Name: "Task 3", Room: "Kitchen", Effort: 2, FrequencyDays: 7},
	}

	for _, task := range tasks {
		require.NoError(t, taskRepo.Create(task))
	}

	// Run scheduler
	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	require.NoError(t, scheduler.Schedule())

	// Verify no day exceeds max effort
	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, i)
		effort, err := scheduler.GetDailyEffort(date)
		require.NoError(t, err)
		assert.LessOrEqual(t, effort, 3, "day %d effort should not exceed max", i)
	}
}

func TestScheduler_PrioritizesOverdue(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(3))

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	nextWeek := now.AddDate(0, 0, 7)

	// Create an overdue task and a future task
	overdueTask := &models.Task{
		Name:          "Overdue",
		Room:          "Kitchen",
		Effort:        3,
		FrequencyDays: 7,
		NextScheduled: &yesterday,
	}
	futureTask := &models.Task{
		Name:          "Future",
		Room:          "Bathroom",
		Effort:        3,
		FrequencyDays: 7,
		NextScheduled: &nextWeek,
	}

	require.NoError(t, taskRepo.Create(overdueTask))
	require.NoError(t, taskRepo.Create(futureTask))

	// Run scheduler
	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	require.NoError(t, scheduler.Schedule())

	// Check what's scheduled for today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayScheduled, err := scheduledRepo.GetByDate(today)
	require.NoError(t, err)

	// Overdue task should be scheduled for today
	foundOverdue := false
	for _, st := range todayScheduled {
		if st.TaskID == overdueTask.ID {
			foundOverdue = true
			break
		}
	}
	assert.True(t, foundOverdue, "overdue task should be scheduled for today")
}

func TestScheduler_ScheduleTask(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	task := &models.Task{
		Name:          "Test Task",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}
	require.NoError(t, taskRepo.Create(task))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	err := scheduler.ScheduleTask(task)
	require.NoError(t, err)

	// Verify scheduled
	scheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, scheduled, 1)
}

func TestScheduler_FindNextAvailableDay(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	// Create a task and fill today's capacity
	task := &models.Task{Name: "Existing", Room: "Kitchen", Effort: 3, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	today := time.Now()
	st := models.NewScheduledTask(task.ID, today)
	require.NoError(t, scheduledRepo.Create(st))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Try to find day for effort 3 with max 5
	// Today has 3, so remaining is 2, should go to tomorrow
	date, err := scheduler.FindNextAvailableDay(3, today, 5)
	require.NoError(t, err)

	tomorrow := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	foundDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	assert.Equal(t, tomorrow, foundDate)
}

func TestScheduler_Reschedule(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	// Create and schedule a task
	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	require.NoError(t, scheduler.Schedule())

	// Get original schedule
	original, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	require.Len(t, original, 1)

	// Reschedule
	err = scheduler.Reschedule()
	require.NoError(t, err)

	// Should still be scheduled
	rescheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, rescheduled, 1)
}

func TestScheduler_Idempotent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Schedule multiple times
	require.NoError(t, scheduler.Schedule())
	require.NoError(t, scheduler.Schedule())
	require.NoError(t, scheduler.Schedule())

	// Should still only have one schedule entry
	scheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, scheduled, 1)
}

func TestScheduler_ClearSchedule(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Create schedule entry
	st := models.NewScheduledTask(task.ID, time.Now())
	require.NoError(t, scheduledRepo.Create(st))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
	err := scheduler.ClearSchedule(task.ID)
	require.NoError(t, err)

	// Verify cleared
	scheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, scheduled, 0)
}
