package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/user/chore-scheduler/internal/models"
)

// ConfigData is the view model for the config page.
type ConfigData struct {
	MaxDailyEffort int
	EmailTo        string
	EmailFrom      string
	Flash          string
	Error          string
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	data := s.loadConfigData()
	render(w, "config.html", data)
}

func (s *Server) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	data := s.loadConfigData()

	if v := strings.TrimSpace(r.FormValue("max_daily_effort")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			data.Error = "max_daily_effort must be a positive integer"
			render(w, "config.html", data)
			return
		}
		if err := s.configRepo.SetMaxDailyEffort(n); err != nil {
			data.Error = err.Error()
			render(w, "config.html", data)
			return
		}
		if err := s.scheduler.Reschedule(); err != nil {
			data.Error = "saved but reschedule failed: " + err.Error()
			render(w, "config.html", data)
			return
		}
		data.MaxDailyEffort = n
	}

	if v := strings.TrimSpace(r.FormValue("email_to")); v != "" {
		if err := s.configRepo.Set(models.ConfigKeyEmailTo, v); err != nil {
			data.Error = err.Error()
			render(w, "config.html", data)
			return
		}
		data.EmailTo = v
	}

	if v := strings.TrimSpace(r.FormValue("email_from")); v != "" {
		if err := s.configRepo.Set(models.ConfigKeyEmailFrom, v); err != nil {
			data.Error = err.Error()
			render(w, "config.html", data)
			return
		}
		data.EmailFrom = v
	}

	data.Flash = "Settings saved."
	render(w, "config.html", data)
}

func (s *Server) loadConfigData() ConfigData {
	effort, _ := s.configRepo.GetMaxDailyEffort()
	emailTo, _ := s.configRepo.Get(models.ConfigKeyEmailTo)
	emailFrom, _ := s.configRepo.Get(models.ConfigKeyEmailFrom)
	if emailFrom == "" {
		emailFrom = models.DefaultEmailFrom
	}
	return ConfigData{MaxDailyEffort: effort, EmailTo: emailTo, EmailFrom: emailFrom}
}
