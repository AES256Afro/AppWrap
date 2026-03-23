package cmd

import (
	"os"
	"path/filepath"
)

// findConfigDir locates the configs directory (for system DLLs list, etc.).
// Used by all commands that create a service instance.
func findConfigDir() string {
	// Check relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "configs")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	// Check current directory
	if _, err := os.Stat("configs"); err == nil {
		return "configs"
	}
	return ""
}
