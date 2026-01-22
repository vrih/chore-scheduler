package scheduler

import (
	"fmt"
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

func TestScheduler_RescheduleFromDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	// Set max effort to 5 - only one task with effort 3 can fit per day
	require.NoError(t, configRepo.SetMaxDailyEffort(5))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create two tasks with effort 3 each
	task1 := &models.Task{Name: "Task 1", Room: "Kitchen", Effort: 3, FrequencyDays: 7, NextScheduled: &today}
	task2 := &models.Task{Name: "Task 2", Room: "Bathroom", Effort: 3, FrequencyDays: 7, NextScheduled: &today}
	require.NoError(t, taskRepo.Create(task1))
	require.NoError(t, taskRepo.Create(task2))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Schedule both - task1 should be today, task2 pushed to tomorrow
	require.NoError(t, scheduler.Schedule())

	// Verify initial schedule
	scheduled1, _ := scheduledRepo.GetByTask(task1.ID)
	scheduled2, _ := scheduledRepo.GetByTask(task2.ID)
	require.Len(t, scheduled1, 1)
	require.Len(t, scheduled2, 1)

	task1Date := time.Date(scheduled1[0].ScheduledDate.Year(), scheduled1[0].ScheduledDate.Month(), scheduled1[0].ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)
	task2Date := time.Date(scheduled2[0].ScheduledDate.Year(), scheduled2[0].ScheduledDate.Month(), scheduled2[0].ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)

	// Task2 should be on a different day than task1
	assert.NotEqual(t, task1Date, task2Date)

	// Clear task1's schedule (simulating early completion)
	require.NoError(t, scheduledRepo.ClearForTask(task1.ID))

	// Reschedule from today - task2 should now move to fill freed capacity
	err := scheduler.RescheduleFromDate(today)
	require.NoError(t, err)

	// Task2 should now be scheduled on today since there's capacity
	scheduled2After, _ := scheduledRepo.GetByTask(task2.ID)
	require.Len(t, scheduled2After, 1)
	task2DateAfter := time.Date(scheduled2After[0].ScheduledDate.Year(), scheduled2After[0].ScheduledDate.Month(), scheduled2After[0].ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)
	assert.Equal(t, today, task2DateAfter, "task2 should be pulled forward to today")
}

func TestScheduler_RescheduleFromDate_NothingToReschedule(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Call with no scheduled tasks - should not error
	futureDate := time.Now().AddDate(0, 0, 30)
	err := scheduler.RescheduleFromDate(futureDate)
	require.NoError(t, err)
}

func TestScheduler_RescheduleFromDate_PastDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	// Create a task scheduled for tomorrow
	task := &models.Task{Name: "Task", Room: "Kitchen", Effort: 2, FrequencyDays: 7, NextScheduled: &tomorrow}
	require.NoError(t, taskRepo.Create(task))
	require.NoError(t, scheduledRepo.Create(models.NewScheduledTask(task.ID, tomorrow)))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Call with past date - should use today
	pastDate := today.AddDate(0, 0, -5)
	err := scheduler.RescheduleFromDate(pastDate)
	require.NoError(t, err)

	// Task should still be scheduled (on today since we're rescheduling from today)
	scheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, scheduled, 1)
}

func TestScheduler_CompleteTaskAndReschedule_EarlyCompletion(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	inFiveDays := today.AddDate(0, 0, 5)

	// Create a task scheduled for 5 days from now
	task := &models.Task{Name: "Future Task", Room: "Kitchen", Effort: 2, FrequencyDays: 7, NextScheduled: &inFiveDays}
	require.NoError(t, taskRepo.Create(task))
	require.NoError(t, scheduledRepo.Create(models.NewScheduledTask(task.ID, inFiveDays)))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Complete early
	originalDate, wasEarly, err := scheduler.CompleteTaskAndReschedule(task.ID)
	require.NoError(t, err)
	assert.True(t, wasEarly, "should be early completion")
	assert.NotNil(t, originalDate, "should have original date")

	origDateNorm := time.Date(originalDate.Year(), originalDate.Month(), originalDate.Day(), 0, 0, 0, 0, time.UTC)
	assert.Equal(t, inFiveDays, origDateNorm, "original date should be 5 days from now")

	// Schedule should be cleared
	scheduled, err := scheduledRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, scheduled, 0)
}

func TestScheduler_CompleteTaskAndReschedule_OnTime(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create a task scheduled for today
	task := &models.Task{Name: "Today Task", Room: "Kitchen", Effort: 2, FrequencyDays: 7, NextScheduled: &today}
	require.NoError(t, taskRepo.Create(task))
	require.NoError(t, scheduledRepo.Create(models.NewScheduledTask(task.ID, today)))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Complete on time
	originalDate, wasEarly, err := scheduler.CompleteTaskAndReschedule(task.ID)
	require.NoError(t, err)
	assert.False(t, wasEarly, "should not be early completion")
	assert.Nil(t, originalDate, "should not have original date for on-time completion")
}

func TestScheduler_CompleteTaskAndReschedule_Overdue(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	// Create a task scheduled for yesterday (overdue)
	task := &models.Task{Name: "Overdue Task", Room: "Kitchen", Effort: 2, FrequencyDays: 7, NextScheduled: &yesterday}
	require.NoError(t, taskRepo.Create(task))
	require.NoError(t, scheduledRepo.Create(models.NewScheduledTask(task.ID, yesterday)))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Complete overdue task
	originalDate, wasEarly, err := scheduler.CompleteTaskAndReschedule(task.ID)
	require.NoError(t, err)
	assert.False(t, wasEarly, "should not be early completion for overdue task")
	assert.Nil(t, originalDate, "should not have original date for overdue completion")
}

func TestScheduler_CompleteTaskAndReschedule_NotScheduled(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	require.NoError(t, configRepo.SetMaxDailyEffort(10))

	// Create a task with no schedule entry
	task := &models.Task{Name: "Unscheduled Task", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Complete unscheduled task
	originalDate, wasEarly, err := scheduler.CompleteTaskAndReschedule(task.ID)
	require.NoError(t, err)
	assert.False(t, wasEarly, "should not be early completion for unscheduled task")
	assert.Nil(t, originalDate, "should not have original date for unscheduled task")
}

func TestScheduler_EarlyCompletion_Integration(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := repository.NewTaskRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)

	// Set max effort to 4 - forces spreading across days
	require.NoError(t, configRepo.SetMaxDailyEffort(4))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create 4 tasks with effort 2 each - will need 2 days
	tasks := make([]*models.Task, 4)
	for i := 0; i < 4; i++ {
		nextSched := today
		tasks[i] = &models.Task{
			Name:          fmt.Sprintf("Task %d", i+1),
			Room:          "Kitchen",
			Effort:        2,
			FrequencyDays: 7,
			NextScheduled: &nextSched,
		}
		require.NoError(t, taskRepo.Create(tasks[i]))
	}

	scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)

	// Schedule all tasks
	require.NoError(t, scheduler.Schedule())

	// Count tasks per day
	countByDay := func() map[string]int {
		counts := make(map[string]int)
		for _, task := range tasks {
			sched, _ := scheduledRepo.GetByTask(task.ID)
			if len(sched) > 0 {
				dateStr := sched[0].ScheduledDate.Format("2006-01-02")
				counts[dateStr]++
			}
		}
		return counts
	}

	initialCounts := countByDay()
	// With effort 2 and max 4, should have 2 tasks per day
	t.Logf("Initial distribution: %v", initialCounts)

	// Get task scheduled for tomorrow (or day after)
	var futureTask *models.Task
	var futureTaskOrigDate time.Time
	for _, task := range tasks {
		sched, _ := scheduledRepo.GetByTask(task.ID)
		if len(sched) > 0 {
			schedDate := time.Date(sched[0].ScheduledDate.Year(), sched[0].ScheduledDate.Month(), sched[0].ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)
			if schedDate.After(today) {
				futureTask = task
				futureTaskOrigDate = schedDate
				break
			}
		}
	}

	if futureTask != nil {
		t.Logf("Completing task %d early (was scheduled for %s)", futureTask.ID, futureTaskOrigDate.Format("2006-01-02"))

		// Complete the future task early
		originalDate, wasEarly, err := scheduler.CompleteTaskAndReschedule(futureTask.ID)
		require.NoError(t, err)
		assert.True(t, wasEarly)
		assert.NotNil(t, originalDate)

		// After early completion, remaining tasks should be optimized
		finalCounts := countByDay()
		t.Logf("Final distribution: %v", finalCounts)

		// Verify the completed task is no longer scheduled
		sched, _ := scheduledRepo.GetByTask(futureTask.ID)
		assert.Len(t, sched, 0, "completed task should not be scheduled")
	}
}
