package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// ProfileSummary is a lightweight profile listing entry.
type ProfileSummary struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	AppName  string `json:"appName"`
	Strategy string `json:"strategy"`
	Arch     string `json:"arch"`
	Format   string `json:"format"` // yaml or json
}

// ListProfiles scans a directory for profile files.
func (s *AppService) ListProfiles(dir string) ([]ProfileSummary, error) {
	if dir == "" {
		dir = s.profileDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var profiles []ProfileSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		// Quick check: try to load as profile
		fullPath := filepath.Join(dir, name)
		p, err := profile.Load(fullPath)
		if err != nil {
			continue // Not a valid profile
		}

		format := "yaml"
		if ext == ".json" {
			format = "json"
		}

		profiles = append(profiles, ProfileSummary{
			Name:     name,
			Path:     fullPath,
			AppName:  p.App.Name,
			Strategy: p.Build.Strategy,
			Arch:     p.Binary.Arch,
			Format:   format,
		})
	}

	return profiles, nil
}

// LoadProfile loads a profile from file.
func (s *AppService) LoadProfile(path string) (*profile.AppProfile, error) {
	return profile.Load(path)
}

// SaveProfile writes a profile to file.
func (s *AppService) SaveProfile(p *profile.AppProfile, path, format string) error {
	switch format {
	case "json":
		return profile.WriteJSON(p, path)
	default:
		return profile.WriteYAML(p, path)
	}
}

// DeleteProfile removes a profile file.
func (s *AppService) DeleteProfile(path string) error {
	return os.Remove(path)
}
