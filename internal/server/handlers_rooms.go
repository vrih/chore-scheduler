package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/user/chore-scheduler/internal/models"
)

// RoomSummary holds the cleanliness counts for a single room.
type RoomSummary struct {
	Name        string
	Floor       int
	Monogram    string // first letter of Name, uppercase
	Clean       int
	Due         int
	Dirty       int
	VeryDirty   int
	Unknown     int
	Total       int // total tasks
	DueCount    int // tasks that need attention (Due+Dirty+VeryDirty)
	WorstStatus string
	Overall     string
}

// FloorGroup groups rooms on a single floor.
type FloorGroup struct {
	Floor int
	Label string
	Rooms []RoomSummary
}

// RoomsData is the view model for the rooms list page.
type RoomsData struct {
	FloorGroups []FloorGroup
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

	floors, err := s.roomRepo.GetFloors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byFloor := make(map[int][]RoomSummary)
	for _, room := range rooms {
		tasks, err := s.taskRepo.GetByRoom(room)
		if err != nil {
			continue
		}
		summary := buildRoomSummary(room, tasks)
		summary.Floor = floors[room]
		byFloor[summary.Floor] = append(byFloor[summary.Floor], summary)
	}

	floorNums := make([]int, 0, len(byFloor))
	for f := range byFloor {
		floorNums = append(floorNums, f)
	}
	sort.Ints(floorNums)

	groups := make([]FloorGroup, 0, len(floorNums))
	for _, f := range floorNums {
		groups = append(groups, FloorGroup{Floor: f, Label: floorLabel(f), Rooms: byFloor[f]})
	}

	render(w, "rooms.html", RoomsData{FloorGroups: groups})
}

func (s *Server) handleRoomSetFloor(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	floor, err := strconv.Atoi(r.FormValue("floor"))
	if err != nil {
		http.Error(w, "invalid floor", http.StatusBadRequest)
		return
	}
	if err := s.roomRepo.SetFloor(name, floor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.handleRooms(w, r)
}

func (s *Server) handleRoomDetail(w http.ResponseWriter, r *http.Request) {
	s.renderRoomDetail(w, r.PathValue("name"))
}

func (s *Server) renderRoomDetail(w http.ResponseWriter, name string) {
	tasks, err := s.taskRepo.GetByRoom(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(tasks, func(i, j int) bool {
		ti, tj := tasks[i].NextScheduled, tasks[j].NextScheduled
		if ti == nil && tj == nil {
			return false
		}
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.Before(*tj)
	})
	render(w, "room_detail.html", RoomDetailData{Name: name, Tasks: tasks})
}

func buildRoomSummary(name string, tasks []*models.Task) RoomSummary {
	mono := ""
	if len(name) > 0 {
		mono = strings.ToUpper(string([]rune(name)[:1]))
	}
	s := RoomSummary{Name: name, Monogram: mono, Total: len(tasks), WorstStatus: models.CleanlinessClean}
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
	s.DueCount = s.Due + s.Dirty + s.VeryDirty
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
