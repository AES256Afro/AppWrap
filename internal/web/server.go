package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/theencryptedafro/appwrap/internal/service"
)

//go:embed static/*
var staticFS embed.FS

// Serve starts the web UI server.
func Serve(svc *service.AppService, addr string) error {
	s, err := newServer(svc)
	if err != nil {
		return err
	}
	fmt.Printf("Starting AppWrap Web UI on http://%s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

type server struct {
	svc *service.AppService
	hub *wsHub
	mux *http.ServeMux
}

func newServer(svc *service.AppService) (*server, error) {
	s := &server{
		svc: svc,
		hub: newWSHub(),
		mux: http.NewServeMux(),
	}
	if err := s.routes(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *server) routes() error {
	// API routes
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/profiles", s.handleListProfiles)
	s.mux.HandleFunc("GET /api/profiles/{name}", s.handleGetProfile)
	s.mux.HandleFunc("DELETE /api/profiles/{name}", s.handleDeleteProfile)
	s.mux.HandleFunc("POST /api/scan", s.handleScan)
	s.mux.HandleFunc("POST /api/build", s.handleBuild)
	s.mux.HandleFunc("POST /api/run", s.handleRun)
	s.mux.HandleFunc("GET /api/inspect", s.handleInspect)
	s.mux.HandleFunc("POST /api/keygen", s.handleKeygen)
	s.mux.HandleFunc("GET /api/apps", s.handleListApps)
	s.mux.HandleFunc("GET /api/containers", s.handleListContainers)
	s.mux.HandleFunc("POST /api/containers/{id}/stop", s.handleStopContainer)

	// WebSocket
	s.mux.HandleFunc("GET /ws/events/{opId}", s.handleWS)

	// Static files - serve embedded frontend
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("loading static files: %w", err)
	}
	s.mux.Handle("GET /", http.FileServer(http.FS(staticSub)))
	return nil
}
