package server

import (
	"net/http"
)

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Static assets and PWA files
	mux.HandleFunc("GET /static/", s.handleStatic)
	mux.HandleFunc("GET /sw.js", s.handleServiceWorker)
	mux.HandleFunc("GET /manifest.json", s.handleManifest)

	// Today / upcoming
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /today", s.handleToday)
	mux.HandleFunc("GET /upcoming", s.handleUpcoming)

	// Tasks
	mux.HandleFunc("GET /tasks", s.handleTaskList)
	mux.HandleFunc("GET /tasks/new", s.handleTaskNew)
	mux.HandleFunc("POST /tasks", s.handleTaskCreate)
	mux.HandleFunc("GET /tasks/{id}/edit", s.handleTaskEdit)
	mux.HandleFunc("POST /tasks/{id}", s.handleTaskUpdate) // htmx uses POST with _method override
	mux.HandleFunc("POST /tasks/{id}/delete", s.handleTaskDelete)
	mux.HandleFunc("POST /tasks/{id}/complete", s.handleTaskComplete)
	mux.HandleFunc("POST /tasks/{id}/postpone", s.handleTaskPostpone)
	mux.HandleFunc("POST /tasks/complete-batch", s.handleTaskCompleteBatch)

	// Rooms
	mux.HandleFunc("GET /rooms", s.handleRooms)
	mux.HandleFunc("GET /rooms/{name}", s.handleRoomDetail)
	mux.HandleFunc("POST /rooms/{name}/floor", s.handleRoomSetFloor)

	// Config
	mux.HandleFunc("GET /config", s.handleConfig)
	mux.HandleFunc("POST /config", s.handleConfigSave)

	// Reschedule
	mux.HandleFunc("POST /reschedule", s.handleReschedule)

	return mux
}
