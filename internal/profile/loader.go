package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*AppProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}

	p := &AppProfile{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, p); err != nil {
			return nil, fmt.Errorf("parse YAML profile: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, p); err != nil {
			return nil, fmt.Errorf("parse JSON profile: %w", err)
		}
	default:
		// Try YAML first, fall back to JSON
		if err := yaml.Unmarshal(data, p); err != nil {
			if err2 := json.Unmarshal(data, p); err2 != nil {
				return nil, fmt.Errorf("could not parse as YAML (%v) or JSON (%v)", err, err2)
			}
		}
	}

	return p, nil
}
