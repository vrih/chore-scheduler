package repository

import (
	"fmt"

	"github.com/user/chore-scheduler/internal/db"
)

// RoomRepository defines data access for room metadata (floor assignment).
type RoomRepository interface {
	GetFloors() (map[string]int, error)
	SetFloor(name string, floor int) error
}

type roomRepository struct {
	db *db.DB
}

// NewRoomRepository creates a new RoomRepository
func NewRoomRepository(database *db.DB) RoomRepository {
	return &roomRepository{db: database}
}

// GetFloors returns a map of room name to its assigned floor.
func (r *roomRepository) GetFloors() (map[string]int, error) {
	rows, err := r.db.Query("SELECT name, floor FROM rooms")
	if err != nil {
		return nil, fmt.Errorf("failed to get room floors: %w", err)
	}
	defer rows.Close()

	floors := make(map[string]int)
	for rows.Next() {
		var name string
		var floor int
		if err := rows.Scan(&name, &floor); err != nil {
			return nil, fmt.Errorf("failed to scan room floor: %w", err)
		}
		floors[name] = floor
	}
	return floors, rows.Err()
}

// SetFloor assigns a floor to a room, creating the room if it does not exist.
func (r *roomRepository) SetFloor(name string, floor int) error {
	if _, err := r.db.Exec(`
		INSERT INTO rooms (name, floor) VALUES (?, ?)
		ON CONFLICT(name) DO UPDATE SET floor = excluded.floor
	`, name, floor); err != nil {
		return fmt.Errorf("failed to set room floor: %w", err)
	}
	return nil
}
