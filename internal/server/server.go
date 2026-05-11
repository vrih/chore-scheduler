package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/repository"
	"github.com/user/chore-scheduler/internal/scheduler"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	db             *db.DB
	taskRepo       repository.TaskRepository
	completionRepo repository.CompletionRepository
	configRepo     repository.ConfigRepository
	scheduledRepo  repository.ScheduledTaskRepository
	scheduler      *scheduler.Scheduler
	http           *http.Server
}

// New initialises the database, repositories and HTTP server.
func New(addr, dbPath string) (*Server, error) {
	if dbPath == "" {
		dbPath = os.Getenv("CHORE_SCHEDULER_DB")
	}
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".chore-scheduler", "chore.db")
	}

	database, err := db.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := database.Initialize(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	taskRepo := repository.NewTaskRepository(database)
	completionRepo := repository.NewCompletionRepository(database)
	configRepo := repository.NewConfigRepository(database)
	scheduledRepo := repository.NewScheduledTaskRepository(database)
	sched := scheduler.NewScheduler(taskRepo, configRepo, scheduledRepo)

	if err := sched.RefreshSchedule(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to refresh schedule: %w", err)
	}

	s := &Server{
		db:             database,
		taskRepo:       taskRepo,
		completionRepo: completionRepo,
		configRepo:     configRepo,
		scheduledRepo:  scheduledRepo,
		scheduler:      sched,
	}

	mux := s.routes()
	s.http = &http.Server{Addr: addr, Handler: mux}

	return s, nil
}

// Run starts the HTTP server and blocks until it is shut down.
func (s *Server) Run() error {
	fmt.Printf("Chore Scheduler web server listening on http://%s\n", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server and closes the database.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.http.Shutdown(ctx)
	s.db.Close()
	return err
}

// refreshSchedule is called after any write operation to keep the schedule current.
func (s *Server) refreshSchedule() {
	_ = s.scheduler.RefreshSchedule()
}
