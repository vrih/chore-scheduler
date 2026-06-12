package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
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
	Date         time.Time
	DateLabel    string // "FRI 12 JUN"
	Groups       []RoomGroup
	TodayCount   int
	TotalEffort  int
	MaxEffort    int
	EffortPct    int // 0–100 for the capacity bar
	OverdueCount int
}

// UpcomingDay groups tasks for a single day in the upcoming view.
type UpcomingDay struct {
	Date        time.Time
	DayLabel    string // "Today", "Tomorrow", "Mon 12 Jun"
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

	maxEffort, _ := s.configRepo.GetMaxDailyEffort()

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
	todayCount := 0
	for _, room := range rooms {
		g := RoomGroup{Room: room, Tasks: byRoom[room]}
		groups = append(groups, g)
		todayCount += len(g.Tasks)
	}

	// Count overdue tasks (past their scheduled date, not yet completed)
	allTasks, _ := s.taskRepo.GetAll()
	overdueCount := 0
	for _, t := range allTasks {
		if t.IsOverdue() {
			overdueCount++
		}
	}

	effortPct := 0
	if maxEffort > 0 {
		effortPct = totalEffort * 100 / maxEffort
		if effortPct > 100 {
			effortPct = 100
		}
	}

	render(w, "today.html", TodayData{
		Date:         today,
		DateLabel:    strings.ToUpper(today.Format("Mon 2 Jan")),
		Groups:       groups,
		TodayCount:   todayCount,
		TotalEffort:  totalEffort,
		MaxEffort:    maxEffort,
		EffortPct:    effortPct,
		OverdueCount: overdueCount,
	})
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

		if len(tasks) == 0 {
			continue
		}

		label := date.Format("Mon 2 Jan")
		if i == 0 {
			label = "Today"
		} else if i == 1 {
			label = "Tomorrow"
		}

		grid = append(grid, UpcomingDay{
			Date:        date,
			DayLabel:    label,
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
	s.handleToday(w, r)
}
