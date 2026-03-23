package service

import (
	"os"
	"path/filepath"

	"github.com/theencryptedafro/appwrap/internal/builder"
	"github.com/theencryptedafro/appwrap/internal/runtime"
)

// AppService is the shared backend for CLI, TUI, and Web UI.
// All business logic funnels through here.
type AppService struct {
	configDir  string
	profileDir string
	generator  *builder.Generator
	runtime    *runtime.DockerRuntime
}

// Option configures an AppService.
type Option func(*AppService)

// WithConfigDir sets the configs directory (for system DLLs list, etc.).
func WithConfigDir(dir string) Option {
	return func(s *AppService) {
		s.configDir = dir
	}
}

// WithProfileDir sets the default directory for profile management.
func WithProfileDir(dir string) Option {
	return func(s *AppService) {
		s.profileDir = dir
	}
}

// New creates a new AppService with the given options.
func New(opts ...Option) *AppService {
	s := &AppService{
		generator: builder.NewGenerator(),
		runtime:   runtime.NewDockerRuntime(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Auto-detect config dir if not set
	if s.configDir == "" {
		s.configDir = findConfigDir()
	}

	// Default profile dir to current directory
	if s.profileDir == "" {
		s.profileDir, _ = os.Getwd()
	}

	return s
}

// DockerAvailable checks if Docker is installed and running.
func (s *AppService) DockerAvailable() bool {
	return s.runtime.Available()
}

// ProfileDir returns the configured profile directory.
func (s *AppService) ProfileDir() string {
	return s.profileDir
}

func findConfigDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "configs")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	if _, err := os.Stat("configs"); err == nil {
		return "configs"
	}
	return ""
}
