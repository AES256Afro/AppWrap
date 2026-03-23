package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// Generator orchestrates Dockerfile generation and image building.
type Generator struct {
	strategies map[string]BuildStrategy
}

func NewGenerator() *Generator {
	g := &Generator{
		strategies: make(map[string]BuildStrategy),
	}
	// Register built-in strategies
	g.RegisterStrategy(NewWineStrategy())
	return g
}

func (g *Generator) RegisterStrategy(s BuildStrategy) {
	g.strategies[s.Name()] = s
}

// Build generates a Dockerfile, prepares the build context, and builds a Docker image.
func (g *Generator) Build(ctx context.Context, p *profile.AppProfile, tag string, runtime Runtime) error {
	strategy, ok := g.strategies[p.Build.Strategy]
	if !ok {
		return fmt.Errorf("unknown build strategy: %s (available: wine, windows-servercore)", p.Build.Strategy)
	}

	// Create temp build context
	contextDir, err := os.MkdirTemp("", "appwrap-build-*")
	if err != nil {
		return fmt.Errorf("create build context: %w", err)
	}
	defer os.RemoveAll(contextDir)

	fmt.Printf("Preparing build context in: %s\n", contextDir)

	// Stage files into build context
	if err := strategy.PrepareContext(p, contextDir); err != nil {
		return fmt.Errorf("prepare context: %w", err)
	}

	// Generate Dockerfile
	dockerfile, err := strategy.GenerateDockerfile(p)
	if err != nil {
		return fmt.Errorf("generate Dockerfile: %w", err)
	}

	// Write Dockerfile to context
	dockerfilePath := filepath.Join(contextDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}

	fmt.Printf("Generated Dockerfile (%d bytes)\n", len(dockerfile))

	// Build the image
	fmt.Printf("Building image: %s\n", tag)
	return runtime.Build(ctx, contextDir, tag)
}

// GetStrategy returns a strategy by name.
func (g *Generator) GetStrategy(name string) (BuildStrategy, bool) {
	s, ok := g.strategies[name]
	return s, ok
}

// GenerateOnly generates the Dockerfile and build context without building.
// Useful for debugging or manual builds.
func (g *Generator) GenerateOnly(p *profile.AppProfile, outputDir string) error {
	strategy, ok := g.strategies[p.Build.Strategy]
	if !ok {
		return fmt.Errorf("unknown build strategy: %s", p.Build.Strategy)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Stage files
	if err := strategy.PrepareContext(p, outputDir); err != nil {
		return fmt.Errorf("prepare context: %w", err)
	}

	// Generate Dockerfile
	dockerfile, err := strategy.GenerateDockerfile(p)
	if err != nil {
		return fmt.Errorf("generate Dockerfile: %w", err)
	}

	dockerfilePath := filepath.Join(outputDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}

	fmt.Printf("Build context written to: %s\n", outputDir)
	fmt.Printf("To build manually: docker build -t <tag> %s\n", outputDir)

	return nil
}
