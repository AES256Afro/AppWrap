package web

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/theencryptedafro/appwrap/internal/service"
)

const appVersion = "0.1.0"

// --- helpers ---

// validatePath rejects paths with traversal sequences or suspicious patterns.
func validatePath(p string) bool {
	if p == "" {
		return true // empty is ok, other validation handles required fields
	}
	cleaned := filepath.Clean(p)
	// Reject if cleaning the path changes it to go above the starting directory
	if strings.Contains(cleaned, "..") {
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log but don't write more to w — headers already sent
		_ = err // Response write failures are not actionable
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// --- handlers ---

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"docker":  s.svc.DockerAvailable(),
		"version": appVersion,
	})
}

func (s *server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	profiles, err := s.svc.ListProfiles(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if profiles == nil {
		profiles = []service.ProfileSummary{}
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	// Load from profile dir
	path := filepath.Join(s.svc.ProfileDir(), name)
	p, err := s.svc.LoadProfile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := filepath.Join(s.svc.ProfileDir(), name)
	if err := s.svc.DeleteProfile(path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Profile deleted"})
}

// --- scan (long-running) ---

type scanRequest struct {
	TargetPath string `json:"targetPath"`
	Strategy   string `json:"strategy"`
	Format     string `json:"format"`
	OutputPath string `json:"outputPath"`
	Encrypt    bool   `json:"encrypt"`
	Firewall   string `json:"firewall"`
	VPNConfig  string `json:"vpnConfig"`
	Verbose    bool   `json:"verbose"`
}

func (s *server) handleScan(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.TargetPath == "" {
		writeError(w, http.StatusBadRequest, "targetPath is required")
		return
	}
	if !validatePath(req.TargetPath) || !validatePath(req.OutputPath) || !validatePath(req.VPNConfig) {
		writeError(w, http.StatusBadRequest, "invalid path: directory traversal not allowed")
		return
	}

	opID := uuid.New().String()
	events := make(chan service.Event, 128)
	s.hub.Register(opID, events)

	go func() {
		defer s.hub.Complete(opID)
		defer close(events)

		result, err := s.svc.ScanApp(context.Background(), service.ScanOpts{
			TargetPath: req.TargetPath,
			Strategy:   req.Strategy,
			Format:     req.Format,
			OutputPath: req.OutputPath,
			Encrypt:    req.Encrypt,
			Firewall:   req.Firewall,
			VPNConfig:  req.VPNConfig,
			Verbose:    req.Verbose,
		}, events)

		if err != nil {
			service.Emit(events, service.EventError, err.Error())
			return
		}
		// Send final result as data event
		service.Emit(events, service.EventComplete, "Scan complete: "+result.OutputPath)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"operationId": opID,
		"message":     "Scan started",
	})
}

// --- build (long-running) ---

type buildRequest struct {
	ProfilePath string `json:"profilePath"`
	Tag         string `json:"tag"`
	NoCache     bool   `json:"noCache"`
	GenerateDir string `json:"generateDir"`
}

func (s *server) handleBuild(w http.ResponseWriter, r *http.Request) {
	var req buildRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.ProfilePath == "" {
		writeError(w, http.StatusBadRequest, "profilePath is required")
		return
	}
	if !validatePath(req.ProfilePath) || !validatePath(req.GenerateDir) {
		writeError(w, http.StatusBadRequest, "invalid path: directory traversal not allowed")
		return
	}

	opID := uuid.New().String()
	events := make(chan service.Event, 128)
	s.hub.Register(opID, events)

	go func() {
		defer s.hub.Complete(opID)
		defer close(events)

		result, err := s.svc.BuildImage(context.Background(), service.BuildOpts{
			ProfilePath: req.ProfilePath,
			Tag:         req.Tag,
			NoCache:     req.NoCache,
			GenerateDir: req.GenerateDir,
		}, events)

		if err != nil {
			service.Emit(events, service.EventError, err.Error())
			return
		}
		service.Emit(events, service.EventComplete, "Build complete: "+result.ImageTag)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"operationId": opID,
		"message":     "Build started",
	})
}

// --- run ---

type runRequest struct {
	Image      string `json:"image"`
	Display    string `json:"display"`
	Detach     bool   `json:"detach"`
	Remove     bool   `json:"remove"`
	Name       string `json:"name"`
	Profile    string `json:"profile"`
	AgeKey     string `json:"ageKey"`
	Passphrase string `json:"passphrase"`
}

func (s *server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}
	if !validatePath(req.Profile) || !validatePath(req.AgeKey) {
		writeError(w, http.StatusBadRequest, "invalid path: directory traversal not allowed")
		return
	}

	err := s.svc.RunContainer(context.Background(), service.RunOpts{
		Image:      req.Image,
		Display:    req.Display,
		Detach:     req.Detach,
		Remove:     req.Remove,
		Name:       req.Name,
		Profile:    req.Profile,
		AgeKey:     req.AgeKey,
		Passphrase: req.Passphrase,
	}, nil)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container started"})
}

// --- inspect ---

func (s *server) handleInspect(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "target query parameter is required")
		return
	}
	if !validatePath(target) {
		writeError(w, http.StatusBadRequest, "invalid path: directory traversal not allowed")
		return
	}

	result, err := s.svc.InspectBinary(context.Background(), service.InspectOpts{
		TargetPath: target,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- keygen ---

type keygenRequest struct {
	OutputDir string `json:"outputDir"`
}

func (s *server) handleKeygen(w http.ResponseWriter, r *http.Request) {
	var req keygenRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.OutputDir == "" {
		req.OutputDir = "."
	}

	result, err := s.svc.GenerateKeys(context.Background(), service.KeygenOpts{
		OutputDir: req.OutputDir,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- installed apps ---

func (s *server) handleListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.svc.ListInstalledApps()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if apps == nil {
		apps = []service.InstalledApp{}
	}
	writeJSON(w, http.StatusOK, apps)
}

// --- containers ---

func (s *server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := s.svc.ListContainers(context.Background())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if containers == nil {
		containers = []service.ContainerInfo{}
	}
	writeJSON(w, http.StatusOK, containers)
}

func (s *server) handleStopContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.svc.StopContainer(context.Background(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Container stopped"})
}
