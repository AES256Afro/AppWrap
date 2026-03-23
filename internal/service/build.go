package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// BuildOpts configures a build operation.
type BuildOpts struct {
	ProfilePath string
	Tag         string
	NoCache     bool
	GenerateDir string // non-empty = generate only, don't build
}

// BuildResult holds the output of a build.
type BuildResult struct {
	ImageTag string
	Profile  *profile.AppProfile
}

// BuildImage builds a Docker image from a profile.
func (s *AppService) BuildImage(ctx context.Context, opts BuildOpts, events chan<- Event) (*BuildResult, error) {
	Emit(events, EventInfo, fmt.Sprintf("Loading profile: %s", opts.ProfilePath))

	p, err := profile.Load(opts.ProfilePath)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	// Validate
	v := profile.Validate(p)
	for _, w := range v.Warnings {
		Emit(events, EventWarning, w)
	}
	if !v.IsValid() {
		for _, e := range v.Errors {
			Emit(events, EventError, e)
		}
		return nil, fmt.Errorf("profile validation failed")
	}

	// Determine tag
	tag := opts.Tag
	if tag == "" {
		name := strings.ToLower(p.App.Name)
		name = strings.ReplaceAll(name, " ", "-")
		tag = name + ":latest"
	}

	// Generate-only mode
	if opts.GenerateDir != "" {
		Emit(events, EventInfo, fmt.Sprintf("Generating build context to: %s", opts.GenerateDir))
		if err := s.generator.GenerateOnly(p, opts.GenerateDir); err != nil {
			return nil, fmt.Errorf("generating dockerfile: %w", err)
		}
		EmitComplete(events, fmt.Sprintf("Build context written to: %s", opts.GenerateDir))
		return &BuildResult{ImageTag: tag, Profile: p}, nil
	}

	// Check Docker
	if !s.runtime.Available() {
		return nil, fmt.Errorf("Docker is not available. Make sure Docker is installed and running")
	}

	EmitProgress(events, 10, "Preparing build context...")

	// Get the strategy
	strategy, ok := s.generator.GetStrategy(p.Build.Strategy)
	if !ok {
		return nil, fmt.Errorf("unknown build strategy: %s", p.Build.Strategy)
	}

	// Create temp build context
	contextDir, err := os.MkdirTemp("", "appwrap-build-*")
	if err != nil {
		return nil, fmt.Errorf("create build context: %w", err)
	}
	defer os.RemoveAll(contextDir)

	// Stage files
	if err := strategy.PrepareContext(p, contextDir); err != nil {
		return nil, fmt.Errorf("prepare context: %w", err)
	}

	EmitProgress(events, 30, "Generating Dockerfile...")

	// Generate Dockerfile
	dockerfile, err := strategy.GenerateDockerfile(p)
	if err != nil {
		return nil, fmt.Errorf("generate Dockerfile: %w", err)
	}

	dockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return nil, fmt.Errorf("write Dockerfile: %w", err)
	}

	Emit(events, EventInfo, fmt.Sprintf("Dockerfile generated (%d bytes)", len(dockerfile)))
	EmitProgress(events, 40, fmt.Sprintf("Building image: %s", tag))

	// Build with event streaming
	var output io.Writer
	if events != nil {
		ew := NewEventWriter(events)
		defer ew.Flush()
		output = ew
	}

	if err := s.runtime.BuildWithOutput(ctx, contextDir, tag, output); err != nil {
		return nil, fmt.Errorf("building image: %w", err)
	}

	EmitComplete(events, fmt.Sprintf("Image built: %s", tag))

	return &BuildResult{ImageTag: tag, Profile: p}, nil
}
