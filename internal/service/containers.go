package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ContainerInfo describes a running or stopped container.
type ContainerInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Created string `json:"created"`
	Ports   string `json:"ports"`
}

// dockerPSEntry matches docker ps --format json output.
type dockerPSEntry struct {
	ID      string `json:"ID"`
	Names   string `json:"Names"`
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	State   string `json:"State"`
	Created string `json:"CreatedAt"`
	Ports   string `json:"Ports"`
}

// ListContainers returns all containers (running and stopped).
func (s *AppService) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	var containers []ContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry dockerPSEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		containers = append(containers, ContainerInfo{
			ID:      entry.ID,
			Name:    entry.Names,
			Image:   entry.Image,
			Status:  entry.Status,
			State:   entry.State,
			Created: entry.Created,
			Ports:   entry.Ports,
		})
	}

	return containers, nil
}

// StopContainer stops a running container.
func (s *AppService) StopContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker stop failed: %s: %w", string(out), err)
	}
	return nil
}

// RemoveContainer removes a container.
func (s *AppService) RemoveContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm failed: %s: %w", string(out), err)
	}
	return nil
}

// ContainerLogs streams container logs to the events channel.
func (s *AppService) ContainerLogs(ctx context.Context, id string, events chan<- Event) error {
	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", "--tail", "100", id)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("docker logs stdout pipe %s: %w", id, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("docker logs stderr pipe %s: %w", id, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker logs failed: %w", err)
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			EmitLog(events, scanner.Text())
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			EmitLog(events, scanner.Text())
		}
	}()

	return cmd.Wait()
}
