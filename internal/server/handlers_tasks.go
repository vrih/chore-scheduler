package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/user/chore-scheduler/internal/models"
)

// TaskListData is the view model for the task list page.
type TaskListData struct {
	Tasks       []*models.Task
	OverdueOnly bool
	Flash       string
}

// TaskFormData is the view model for add/edit forms.
type TaskFormData struct {
	Task  *models.Task
	Error string
	IsNew bool
}

func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) {
	overdueOnly := r.URL.Query().Get("overdue") == "1"

	var tasks []*models.Task
	var err error
	if overdueOnly {
		tasks, err = s.taskRepo.GetOverdue()
	} else {
		tasks, err = s.taskRepo.GetAll()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render(w, "tasks_list.html", TaskListData{Tasks: tasks, OverdueOnly: overdueOnly})
}

func (s *Server) handleTaskNew(w http.ResponseWriter, r *http.Request) {
	render(w, "task_form.html", TaskFormData{Task: &models.Task{Effort: 2, FrequencyDays: 7}, IsNew: true})
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	task, errMsg := parseTaskForm(r)
	if errMsg != "" {
		render(w, "task_form.html", TaskFormData{Task: task, Error: errMsg, IsNew: true})
		return
	}

	if err := s.taskRepo.Create(task); err != nil {
		render(w, "task_form.html", TaskFormData{Task: task, Error: err.Error(), IsNew: true})
		return
	}
	if err := s.scheduler.ScheduleTask(task); err != nil {
		// non-fatal: task is created, scheduling failed
		_ = err
	}
	s.refreshSchedule()

	// Full page redirect after create
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (s *Server) handleTaskEdit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	task, err := s.taskRepo.Get(id)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	render(w, "task_form.html", TaskFormData{Task: task, IsNew: false})
}

func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	task, err := s.taskRepo.Get(id)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	updated, errMsg := parseTaskForm(r)
	if errMsg != "" {
		updated.ID = task.ID
		render(w, "task_form.html", TaskFormData{Task: updated, Error: errMsg, IsNew: false})
		return
	}

	effortChanged := updated.Effort != task.Effort
	freqChanged := updated.FrequencyDays != task.FrequencyDays

	task.Name = updated.Name
	task.Room = updated.Room
	task.Effort = updated.Effort
	task.FrequencyDays = updated.FrequencyDays

	if err := s.taskRepo.Update(task); err != nil {
		render(w, "task_form.html", TaskFormData{Task: task, Error: err.Error(), IsNew: false})
		return
	}
	if effortChanged || freqChanged {
		_ = s.scheduler.ScheduleTask(task)
	}
	s.refreshSchedule()

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.taskRepo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.refreshSchedule()

	// htmx: return an empty 200 so the row is removed via hx-target/hx-swap
	w.Header().Set("HX-Trigger", "taskDeleted")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.completeTask(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("return") == "room" {
		if task, err := s.taskRepo.Get(id); err == nil {
			s.renderRoomDetail(w, task.Room)
			return
		}
	}

	// Return refreshed today fragment
	s.handleToday(w, r)
}

func (s *Server) handleTaskCompleteBatch(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ids := r.Form["id"]
	for _, raw := range ids {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		_ = s.completeTask(id)
	}
	s.handleToday(w, r)
}

func (s *Server) handleTaskPostpone(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	days := 1
	if d := r.FormValue("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			days = n
		}
	}

	task, err := s.taskRepo.Get(id)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if err := s.scheduler.ClearSchedule(task.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	minDate := now.AddDate(0, 0, days)
	task.NextScheduled = &minDate
	if err := s.taskRepo.Update(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.scheduler.ScheduleTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.refreshSchedule()

	if r.FormValue("return") == "room" {
		s.renderRoomDetail(w, task.Room)
		return
	}

	s.handleToday(w, r)
}

// completeTask is the shared completion logic.
func (s *Server) completeTask(id int64) error {
	task, err := s.taskRepo.Get(id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	completion := models.NewCompletion(task.ID)
	if err := s.completionRepo.Create(completion); err != nil {
		return err
	}

	now := time.Now()
	task.LastCompleted = &now
	next := task.CalculateNextScheduled()
	task.NextScheduled = &next
	if err := s.taskRepo.Update(task); err != nil {
		return err
	}

	if _, _, err := s.scheduler.CompleteTaskAndReschedule(task.ID); err != nil {
		return err
	}
	if err := s.scheduler.ScheduleTask(task); err != nil {
		return err
	}
	s.refreshSchedule()
	return nil
}

// parseTaskForm reads name/room/effort/frequency from a form submission.
func parseTaskForm(r *http.Request) (*models.Task, string) {
	if err := r.ParseForm(); err != nil {
		return &models.Task{}, "invalid form"
	}

	name := strings.TrimSpace(r.FormValue("name"))
	room := strings.TrimSpace(r.FormValue("room"))
	effort, _ := strconv.Atoi(r.FormValue("effort"))
	frequency, _ := strconv.Atoi(r.FormValue("frequency"))

	task := &models.Task{Name: name, Room: room, Effort: effort, FrequencyDays: frequency}
	if err := task.Validate(); err != nil {
		return task, err.Error()
	}
	return task, ""
}

// pathID extracts the {id} path segment and parses it as int64.
func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
