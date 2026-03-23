package builder

import "context"

// Runtime abstracts container runtime operations (Docker, Podman, etc.)
type Runtime interface {
	// Build creates a container image from a build context directory.
	Build(ctx context.Context, contextDir, tag string) error

	// Run starts a container from an image.
	Run(ctx context.Context, image string, config RunConfig) error

	// Available checks if the runtime is installed and accessible.
	Available() bool
}

// RunConfig holds container run options.
type RunConfig struct {
	Name       string
	Ports      map[int]int       // host:container port mappings
	Volumes    map[string]string // host:container volume mappings
	Env        map[string]string // environment variables
	ExtraArgs  []string          // additional docker run arguments
	Detach     bool
	Remove     bool // --rm
}
