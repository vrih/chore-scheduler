package server

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/user/chore-scheduler/internal/models"
)

// RoomGroup groups today's tasks under a single room.
type RoomGroup struct {
	Room  string
	Tasks []*models.Task
}

// TodayData is the view model for the today page.
type TodayData struct {
	Date        time.Time
	Groups      []RoomGroup
	TotalEffort int
}

// UpcomingDay groups tasks for a single day in the upcoming view.
type UpcomingDay struct {
	Date        time.Time
	IsToday     bool
	IsTomorrow  bool
	Tasks       []*models.Task
	TotalEffort int
}

// UpcomingData is the view model for the upcoming page.
type UpcomingData struct {
	Days int
	Grid []UpcomingDay
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	render(w, "layout.html", nil)
}

func (s *Server) handleToday(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	scheduled, err := s.scheduledRepo.GetByDate(today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byRoom := make(map[string][]*models.Task)
	totalEffort := 0
	for _, st := range scheduled {
		task, err := s.taskRepo.Get(st.TaskID)
		if err != nil {
			continue
		}
		byRoom[task.Room] = append(byRoom[task.Room], task)
		totalEffort += task.Effort
	}

	rooms := make([]string, 0, len(byRoom))
	for room := range byRoom {
		rooms = append(rooms, room)
	}
	sort.Strings(rooms)

	groups := make([]RoomGroup, 0, len(rooms))
	for _, room := range rooms {
		groups = append(groups, RoomGroup{Room: room, Tasks: byRoom[room]})
	}

	render(w, "today.html", TodayData{Date: today, Groups: groups, TotalEffort: totalEffort})
}

func (s *Server) handleUpcoming(w http.ResponseWriter, r *http.Request) {
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 30 {
			days = n
		}
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var grid []UpcomingDay
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, i)
		scheduled, err := s.scheduledRepo.GetByDate(date)
		if err != nil {
			continue
		}

		var tasks []*models.Task
		effort := 0
		for _, st := range scheduled {
			task, err := s.taskRepo.Get(st.TaskID)
			if err != nil {
				continue
			}
			tasks = append(tasks, task)
			effort += task.Effort
		}

		grid = append(grid, UpcomingDay{
			Date:        date,
			IsToday:     i == 0,
			IsTomorrow:  i == 1,
			Tasks:       tasks,
			TotalEffort: effort,
		})
	}

	render(w, "upcoming.html", UpcomingData{Days: days, Grid: grid})
}

func (s *Server) handleReschedule(w http.ResponseWriter, r *http.Request) {
	if err := s.scheduler.Reschedule(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return the refreshed today fragment
	s.handleToday(w, r)
}
