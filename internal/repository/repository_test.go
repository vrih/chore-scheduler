package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/models"
)

func setupTestDB(t *testing.T) *db.DB {
	database, err := db.New(":memory:")
	require.NoError(t, err)
	require.NoError(t, database.Initialize())
	return database
}

// Task Repository Tests

func TestTaskRepository_Create(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	task := &models.Task{
		Name:          "Test Task",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}

	err := repo.Create(task)
	require.NoError(t, err)
	assert.NotZero(t, task.ID)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskRepository_Create_Validation(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Empty name should fail
	task := &models.Task{
		Name:          "",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}

	err := repo.Create(task)
	assert.Equal(t, models.ErrEmptyName, err)

	// Empty room should fail
	task2 := &models.Task{
		Name:          "Test",
		Room:          "",
		Effort:        2,
		FrequencyDays: 7,
	}

	err = repo.Create(task2)
	assert.Equal(t, models.ErrEmptyRoom, err)
}

func TestTaskRepository_Get(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Create a task
	task := &models.Task{
		Name:          "Test Task",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}
	require.NoError(t, repo.Create(task))

	// Get it back
	retrieved, err := repo.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.Name, retrieved.Name)
	assert.Equal(t, task.Room, retrieved.Room)
	assert.Equal(t, task.Effort, retrieved.Effort)
	assert.Equal(t, task.FrequencyDays, retrieved.FrequencyDays)
}

func TestTaskRepository_Get_NotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	_, err := repo.Get(999)
	assert.Equal(t, ErrTaskNotFound, err)
}

func TestTaskRepository_GetAll(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Create multiple tasks
	tasks := []*models.Task{
		{Name: "Task 1", Room: "Kitchen", Effort: 1, FrequencyDays: 3},
		{Name: "Task 2", Room: "Bathroom", Effort: 2, FrequencyDays: 5},
		{Name: "Task 3", Room: "Kitchen", Effort: 3, FrequencyDays: 7},
	}

	for _, task := range tasks {
		require.NoError(t, repo.Create(task))
	}

	// Get all
	retrieved, err := repo.GetAll()
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
}

func TestTaskRepository_Update(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	task := &models.Task{
		Name:          "Original Name",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}
	require.NoError(t, repo.Create(task))

	// Update
	task.Name = "Updated Name"
	task.Room = "Bathroom"
	task.Effort = 3
	err := repo.Update(task)
	require.NoError(t, err)

	// Verify
	retrieved, err := repo.Get(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Bathroom", retrieved.Room)
	assert.Equal(t, 3, retrieved.Effort)
}

func TestTaskRepository_Update_NotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	task := &models.Task{
		ID:            999,
		Name:          "Test",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}

	err := repo.Update(task)
	assert.Equal(t, ErrTaskNotFound, err)
}

func TestTaskRepository_Delete(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	task := &models.Task{
		Name:          "To Delete",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
	}
	require.NoError(t, repo.Create(task))

	// Delete
	err := repo.Delete(task.ID)
	require.NoError(t, err)

	// Verify gone
	_, err = repo.Get(task.ID)
	assert.Equal(t, ErrTaskNotFound, err)
}

func TestTaskRepository_Delete_NotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	err := repo.Delete(999)
	assert.Equal(t, ErrTaskNotFound, err)
}

func TestTaskRepository_GetOverdue(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	// Create overdue and non-overdue tasks
	overdueTask := &models.Task{
		Name:          "Overdue",
		Room:          "Kitchen",
		Effort:        2,
		FrequencyDays: 7,
		NextScheduled: &yesterday,
	}
	futureTask := &models.Task{
		Name:          "Future",
		Room:          "Bathroom",
		Effort:        2,
		FrequencyDays: 7,
		NextScheduled: &tomorrow,
	}

	require.NoError(t, repo.Create(overdueTask))
	require.NoError(t, repo.Create(futureTask))

	overdue, err := repo.GetOverdue()
	require.NoError(t, err)
	assert.Len(t, overdue, 1)
	assert.Equal(t, overdueTask.ID, overdue[0].ID)
}

// Completion Repository Tests

func TestCompletionRepository_Create(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	compRepo := NewCompletionRepository(database)

	// Create a task first
	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Create completion
	completion := models.NewCompletion(task.ID)
	err := compRepo.Create(completion)
	require.NoError(t, err)
	assert.NotZero(t, completion.ID)
}

func TestCompletionRepository_GetByTaskID(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	compRepo := NewCompletionRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Create multiple completions
	for i := 0; i < 3; i++ {
		c := models.NewCompletion(task.ID)
		require.NoError(t, compRepo.Create(c))
	}

	completions, err := compRepo.GetByTaskID(task.ID)
	require.NoError(t, err)
	assert.Len(t, completions, 3)
}

func TestCompletionRepository_GetRecent(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	compRepo := NewCompletionRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Create multiple completions
	for i := 0; i < 5; i++ {
		c := models.NewCompletion(task.ID)
		require.NoError(t, compRepo.Create(c))
	}

	recent, err := compRepo.GetRecent(3)
	require.NoError(t, err)
	assert.Len(t, recent, 3)
}

// Config Repository Tests

func TestConfigRepository_GetSet(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewConfigRepository(database)

	// Set a value
	err := repo.Set("test_key", "test_value")
	require.NoError(t, err)

	// Get it back
	value, err := repo.Get("test_key")
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestConfigRepository_Get_NotFound(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewConfigRepository(database)

	_, err := repo.Get("nonexistent")
	assert.Equal(t, ErrConfigNotFound, err)
}

func TestConfigRepository_GetAll(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewConfigRepository(database)

	// Should have default max_daily_effort
	configs, err := repo.GetAll()
	require.NoError(t, err)
	assert.NotEmpty(t, configs)
}

func TestConfigRepository_MaxDailyEffort(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewConfigRepository(database)

	// Get default
	effort, err := repo.GetMaxDailyEffort()
	require.NoError(t, err)
	assert.Equal(t, 10, effort)

	// Set new value
	err = repo.SetMaxDailyEffort(15)
	require.NoError(t, err)

	// Verify
	effort, err = repo.GetMaxDailyEffort()
	require.NoError(t, err)
	assert.Equal(t, 15, effort)
}

func TestConfigRepository_SetMaxDailyEffort_Invalid(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewConfigRepository(database)

	err := repo.SetMaxDailyEffort(0)
	assert.Error(t, err)

	err = repo.SetMaxDailyEffort(-1)
	assert.Error(t, err)
}

// Scheduled Task Repository Tests

func TestScheduledTaskRepository_Create(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	st := models.NewScheduledTask(task.ID, time.Now())
	err := schedRepo.Create(st)
	require.NoError(t, err)
	assert.NotZero(t, st.ID)
}

func TestScheduledTaskRepository_GetByDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)

	// Schedule for today
	st1 := models.NewScheduledTask(task.ID, today)
	require.NoError(t, schedRepo.Create(st1))

	// Get by today's date
	tasks, err := schedRepo.GetByDate(today)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	// Get by tomorrow's date
	tasks, err = schedRepo.GetByDate(tomorrow)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestScheduledTaskRepository_GetByTask(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Schedule on multiple days
	for i := 0; i < 3; i++ {
		st := models.NewScheduledTask(task.ID, time.Now().AddDate(0, 0, i))
		require.NoError(t, schedRepo.Create(st))
	}

	tasks, err := schedRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestScheduledTaskRepository_ClearForTask(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Schedule on multiple days
	for i := 0; i < 3; i++ {
		st := models.NewScheduledTask(task.ID, time.Now().AddDate(0, 0, i))
		require.NoError(t, schedRepo.Create(st))
	}

	// Clear
	err := schedRepo.ClearForTask(task.ID)
	require.NoError(t, err)

	// Verify cleared
	tasks, err := schedRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestScheduledTaskRepository_GetDailyEffort(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	// Create tasks with different effort
	task1 := &models.Task{Name: "Easy", Room: "Kitchen", Effort: 1, FrequencyDays: 7}
	task2 := &models.Task{Name: "Medium", Room: "Bathroom", Effort: 2, FrequencyDays: 7}
	task3 := &models.Task{Name: "Hard", Room: "Kitchen", Effort: 3, FrequencyDays: 7}

	require.NoError(t, taskRepo.Create(task1))
	require.NoError(t, taskRepo.Create(task2))
	require.NoError(t, taskRepo.Create(task3))

	today := time.Now()

	// Schedule all for today
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task1.ID, today)))
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task2.ID, today)))
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task3.ID, today)))

	// Get daily effort
	effort, err := schedRepo.GetDailyEffort(today)
	require.NoError(t, err)
	assert.Equal(t, 6, effort) // 1 + 2 + 3
}

func TestScheduledTaskRepository_IsTaskScheduledOnDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)

	// Schedule for today
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task.ID, today)))

	// Check
	scheduled, err := schedRepo.IsTaskScheduledOnDate(task.ID, today)
	require.NoError(t, err)
	assert.True(t, scheduled)

	scheduled, err = schedRepo.IsTaskScheduledOnDate(task.ID, tomorrow)
	require.NoError(t, err)
	assert.False(t, scheduled)
}

func TestScheduledTaskRepository_ClearAll(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	task := &models.Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task))

	// Schedule on multiple days
	for i := 0; i < 5; i++ {
		st := models.NewScheduledTask(task.ID, time.Now().AddDate(0, 0, i))
		require.NoError(t, schedRepo.Create(st))
	}

	// Clear all
	err := schedRepo.ClearAll()
	require.NoError(t, err)

	// Verify all cleared
	tasks, err := schedRepo.GetByTask(task.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

// GetByRoom and GetAllRooms Tests

func TestTaskRepository_GetByRoom(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Create tasks in different rooms
	tasks := []*models.Task{
		{Name: "Task 1", Room: "Kitchen", Effort: 1, FrequencyDays: 3},
		{Name: "Task 2", Room: "Kitchen", Effort: 2, FrequencyDays: 5},
		{Name: "Task 3", Room: "Bathroom", Effort: 3, FrequencyDays: 7},
		{Name: "Task 4", Room: "Living Room", Effort: 1, FrequencyDays: 14},
	}

	for _, task := range tasks {
		require.NoError(t, repo.Create(task))
	}

	// Get Kitchen tasks
	kitchenTasks, err := repo.GetByRoom("Kitchen")
	require.NoError(t, err)
	assert.Len(t, kitchenTasks, 2)
	for _, task := range kitchenTasks {
		assert.Equal(t, "Kitchen", task.Room)
	}

	// Get Bathroom tasks
	bathroomTasks, err := repo.GetByRoom("Bathroom")
	require.NoError(t, err)
	assert.Len(t, bathroomTasks, 1)

	// Get non-existent room
	emptyTasks, err := repo.GetByRoom("Garage")
	require.NoError(t, err)
	assert.Len(t, emptyTasks, 0)
}

func TestTaskRepository_GetAllRooms(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Create tasks in different rooms
	tasks := []*models.Task{
		{Name: "Task 1", Room: "Kitchen", Effort: 1, FrequencyDays: 3},
		{Name: "Task 2", Room: "Kitchen", Effort: 2, FrequencyDays: 5},
		{Name: "Task 3", Room: "Bathroom", Effort: 3, FrequencyDays: 7},
		{Name: "Task 4", Room: "Living Room", Effort: 1, FrequencyDays: 14},
	}

	for _, task := range tasks {
		require.NoError(t, repo.Create(task))
	}

	// Get all rooms
	rooms, err := repo.GetAllRooms()
	require.NoError(t, err)
	assert.Len(t, rooms, 3)

	// Rooms should be sorted
	assert.Contains(t, rooms, "Bathroom")
	assert.Contains(t, rooms, "Kitchen")
	assert.Contains(t, rooms, "Living Room")
}

func TestTaskRepository_GetAllRooms_Empty(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	// Get all rooms when no tasks exist
	rooms, err := repo.GetAllRooms()
	require.NoError(t, err)
	assert.Len(t, rooms, 0)
}

func TestScheduledTaskRepository_GetFromDate(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	schedRepo := NewScheduledTaskRepository(database)

	// Create tasks
	task1 := &models.Task{Name: "Task 1", Room: "Kitchen", Effort: 2, FrequencyDays: 7}
	task2 := &models.Task{Name: "Task 2", Room: "Bathroom", Effort: 3, FrequencyDays: 7}
	task3 := &models.Task{Name: "Task 3", Room: "Kitchen", Effort: 1, FrequencyDays: 7}
	require.NoError(t, taskRepo.Create(task1))
	require.NoError(t, taskRepo.Create(task2))
	require.NoError(t, taskRepo.Create(task3))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	inThreeDays := today.AddDate(0, 0, 3)

	// Schedule tasks on different days
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task1.ID, today)))
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task2.ID, tomorrow)))
	require.NoError(t, schedRepo.Create(models.NewScheduledTask(task3.ID, inThreeDays)))

	// Get from today - should return all 3
	tasks, err := schedRepo.GetFromDate(today)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Get from tomorrow - should return 2 (tomorrow and in 3 days)
	tasks, err = schedRepo.GetFromDate(tomorrow)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Get from 3 days out - should return 1
	tasks, err = schedRepo.GetFromDate(inThreeDays)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task3.ID, tasks[0].TaskID)

	// Get from far future - should return empty
	tasks, err = schedRepo.GetFromDate(today.AddDate(0, 0, 10))
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}
