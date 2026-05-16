package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/chore-scheduler/internal/models"
)

func TestRoomRepository_SetFloor_CreatesAndUpserts(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewRoomRepository(database)

	// Setting a floor for an unknown room creates it.
	require.NoError(t, repo.SetFloor("Loft", 3))

	floors, err := repo.GetFloors()
	require.NoError(t, err)
	assert.Equal(t, 3, floors["Loft"])

	// Setting it again updates rather than duplicating.
	require.NoError(t, repo.SetFloor("Loft", 1))

	floors, err = repo.GetFloors()
	require.NoError(t, err)
	assert.Equal(t, 1, floors["Loft"])
}

func TestRoomRepository_GetFloors_DefaultZero(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	taskRepo := NewTaskRepository(database)
	require.NoError(t, taskRepo.Create(&models.Task{
		Name: "Sweep", Room: "Garage", Effort: 1, FrequencyDays: 7,
	}))

	roomRepo := NewRoomRepository(database)
	floors, err := roomRepo.GetFloors()
	require.NoError(t, err)

	// Room created via a task exists with the default floor 0.
	floor, ok := floors["Garage"]
	require.True(t, ok)
	assert.Equal(t, 0, floor)
}

func TestTaskRepository_Create_ReusesExistingRoom(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewTaskRepository(database)

	require.NoError(t, repo.Create(&models.Task{
		Name: "Wipe counters", Room: "Kitchen", Effort: 1, FrequencyDays: 3,
	}))
	require.NoError(t, repo.Create(&models.Task{
		Name: "Mop floor", Room: "Kitchen", Effort: 2, FrequencyDays: 7,
	}))

	rooms, err := repo.GetAllRooms()
	require.NoError(t, err)

	count := 0
	for _, r := range rooms {
		if r == "Kitchen" {
			count++
		}
	}
	assert.Equal(t, 1, count, "Kitchen should exist exactly once")

	tasks, err := repo.GetByRoom("Kitchen")
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, "Kitchen", task.Room)
		assert.NotZero(t, task.RoomID)
	}
	assert.Equal(t, tasks[0].RoomID, tasks[1].RoomID)
}
