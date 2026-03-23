package builder

import "github.com/theencryptedafro/appwrap/internal/profile"

// BuildStrategy generates Dockerfiles and prepares build contexts.
type BuildStrategy interface {
	// Name returns the strategy identifier.
	Name() string

	// BaseImage returns the Docker base image to use.
	BaseImage(p *profile.AppProfile) string

	// GenerateDockerfile produces the Dockerfile content from a profile.
	GenerateDockerfile(p *profile.AppProfile) (string, error)

	// PrepareContext copies required files into the Docker build context directory.
	PrepareContext(p *profile.AppProfile, contextDir string) error
}
