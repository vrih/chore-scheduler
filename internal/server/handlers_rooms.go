package server

import (
	"net/http"

	"github.com/user/chore-scheduler/internal/models"
)

// RoomSummary holds the cleanliness counts for a single room.
type RoomSummary struct {
	Name        string
	Clean       int
	Due         int
	Dirty       int
	VeryDirty   int
	Unknown     int
	WorstStatus string
	Overall     string
}

// RoomsData is the view model for the rooms list page.
type RoomsData struct {
	Rooms []RoomSummary
}

// RoomDetailData is the view model for a single room page.
type RoomDetailData struct {
	Name  string
	Tasks []*models.Task
}

func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.taskRepo.GetAllRooms()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var summaries []RoomSummary
	for _, room := range rooms {
		tasks, err := s.taskRepo.GetByRoom(room)
		if err != nil {
			continue
		}
		summary := buildRoomSummary(room, tasks)
		summaries = append(summaries, summary)
	}

	render(w, "rooms.html", RoomsData{Rooms: summaries})
}

func (s *Server) handleRoomDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tasks, err := s.taskRepo.GetByRoom(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, "room_detail.html", RoomDetailData{Name: name, Tasks: tasks})
}

func buildRoomSummary(name string, tasks []*models.Task) RoomSummary {
	s := RoomSummary{Name: name, WorstStatus: models.CleanlinessClean}
	order := map[string]int{
		models.CleanlinessClean:     0,
		models.CleanlinessUnknown:   1,
		models.CleanlinessDue:       2,
		models.CleanlinessDirty:     3,
		models.CleanlinessVeryDirty: 4,
	}
	for _, t := range tasks {
		st := t.CleanlinessStatus()
		switch st {
		case models.CleanlinessClean:
			s.Clean++
		case models.CleanlinessDue:
			s.Due++
		case models.CleanlinessDirty:
			s.Dirty++
		case models.CleanlinessVeryDirty:
			s.VeryDirty++
		default:
			s.Unknown++
		}
		if order[st] > order[s.WorstStatus] {
			s.WorstStatus = st
		}
	}
	s.Overall = roomOverall(s.WorstStatus)
	return s
}

func roomOverall(worst string) string {
	switch worst {
	case models.CleanlinessClean:
		return "Spotless"
	case models.CleanlinessUnknown:
		return "Needs Review"
	case models.CleanlinessDue:
		return "Good"
	case models.CleanlinessDirty:
		return "Needs Attention"
	case models.CleanlinessVeryDirty:
		return "Urgent"
	default:
		return "Unknown"
	}
}
