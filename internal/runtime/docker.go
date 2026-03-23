package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/builder"
)

// DockerRuntime implements builder.Runtime using the Docker CLI.
// We use the CLI instead of the Docker SDK to avoid heavy dependencies
// and to work with Docker Desktop, Colima, Rancher, etc.
type DockerRuntime struct{}

func NewDockerRuntime() *DockerRuntime {
	return &DockerRuntime{}
}

func (d *DockerRuntime) Available() bool {
	err := exec.Command("docker", "info").Run()
	return err == nil
}

func (d *DockerRuntime) Build(ctx context.Context, contextDir, tag string) error {
	args := []string{"build", "-t", tag, contextDir}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

func (d *DockerRuntime) Run(ctx context.Context, image string, config builder.RunConfig) error {
	args := []string{"run"}

	if config.Remove {
		args = append(args, "--rm")
	}
	if config.Detach {
		args = append(args, "-d")
	} else {
		args = append(args, "-it")
	}
	if config.Name != "" {
		args = append(args, "--name", config.Name)
	}

	// Port mappings
	for host, container := range config.Ports {
		args = append(args, "-p", strconv.Itoa(host)+":"+strconv.Itoa(container))
	}

	// Volume mappings
	for host, container := range config.Volumes {
		args = append(args, "-v", host+":"+container)
	}

	// Environment variables
	for k, v := range config.Env {
		args = append(args, "-e", k+"="+v)
	}

	// Extra args
	args = append(args, config.ExtraArgs...)

	// Image
	args = append(args, image)

	fmt.Printf("Running: docker %s\n", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %w", err)
	}

	return nil
}

// BuildWithOutput builds a Docker image, directing output to the given writer.
// If output is nil, falls back to os.Stdout.
func (d *DockerRuntime) BuildWithOutput(ctx context.Context, contextDir, tag string, output io.Writer) error {
	args := []string{"build", "-t", tag, contextDir}

	cmd := exec.CommandContext(ctx, "docker", args...)
	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = output
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

// Ensure DockerRuntime implements Runtime
var _ builder.Runtime = (*DockerRuntime)(nil)
