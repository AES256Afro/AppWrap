package profile

import (
	"fmt"
	"os"
)

type ValidationResult struct {
	Errors   []string
	Warnings []string
}

func (v *ValidationResult) IsValid() bool {
	return len(v.Errors) == 0
}

// Validate checks an AppProfile for completeness and correctness.
func Validate(p *AppProfile) *ValidationResult {
	v := &ValidationResult{}

	// Required fields
	if p.App.Name == "" {
		v.Errors = append(v.Errors, "app.name is required")
	}
	if p.Binary.Path == "" {
		v.Errors = append(v.Errors, "binary.path is required")
	} else if _, err := os.Stat(p.Binary.Path); err != nil {
		v.Warnings = append(v.Warnings, fmt.Sprintf("binary.path not found on this machine: %s (OK if building from profile)", p.Binary.Path))
	}
	if p.Binary.Arch == "" {
		v.Errors = append(v.Errors, "binary.arch is required (x86, x64, arm64)")
	}
	if p.Binary.Subsystem == "" {
		v.Errors = append(v.Errors, "binary.subsystem is required (gui, console, service)")
	}

	// Build config
	switch p.Build.Strategy {
	case "wine", "windows-servercore", "windows-nanoserver":
		// valid
	case "":
		v.Errors = append(v.Errors, "build.strategy is required")
	default:
		v.Errors = append(v.Errors, fmt.Sprintf("unknown build.strategy: %s", p.Build.Strategy))
	}

	// Check for unresolved DLLs
	unresolvedCount := 0
	for _, dll := range p.Dependencies.DLLs {
		if !dll.IsSystem && dll.FullPath == "" {
			unresolvedCount++
		}
	}
	if unresolvedCount > 0 {
		v.Warnings = append(v.Warnings, fmt.Sprintf("%d non-system DLLs have unresolved paths", unresolvedCount))
	}

	// GUI apps should have display settings
	if p.Binary.Subsystem == "gui" {
		if p.Display.Width == 0 || p.Display.Height == 0 {
			v.Warnings = append(v.Warnings, "GUI app detected but no display resolution set (will use defaults)")
		}
	}

	return v
}
