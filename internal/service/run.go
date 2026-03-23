package service

import (
	"context"
	"fmt"

	"github.com/theencryptedafro/appwrap/internal/builder"
	"github.com/theencryptedafro/appwrap/internal/profile"
	"github.com/theencryptedafro/appwrap/internal/security"
)

// RunOpts configures a container run.
type RunOpts struct {
	Image      string
	Display    string // none, vnc, novnc, rdp
	Detach     bool
	Remove     bool
	Name       string
	Profile    string // path to profile for security features
	AgeKey     string
	Passphrase string
}

// RunContainer starts a containerized application.
func (s *AppService) RunContainer(ctx context.Context, opts RunOpts, events chan<- Event) error {
	if !s.runtime.Available() {
		return fmt.Errorf("Docker is not available. Make sure Docker is installed and running")
	}

	config := builder.RunConfig{
		Name:    opts.Name,
		Remove:  opts.Remove,
		Detach:  opts.Detach,
		Ports:   make(map[int]int),
		Volumes: make(map[string]string),
		Env:     make(map[string]string),
	}

	// Configure display
	switch opts.Display {
	case "", "none":
		// No display
	case "vnc":
		config.Ports[5901] = 5901
		Emit(events, EventInfo, "VNC server will be available on port 5901")
	case "novnc":
		config.Ports[6080] = 6080
		config.Ports[5901] = 5901
		Emit(events, EventInfo, "noVNC will be available at http://localhost:6080")
	case "rdp":
		config.Ports[3389] = 3389
		Emit(events, EventInfo, "RDP will be available on port 3389")
	default:
		return fmt.Errorf("unknown display mode: %s (use none, vnc, novnc, rdp)", opts.Display)
	}

	// Load profile for security features
	if opts.Profile != "" {
		p, err := profile.Load(opts.Profile)
		if err != nil {
			return fmt.Errorf("load profile: %w", err)
		}

		secArgs := security.CollectDockerRunArgs(p.Security)
		config.ExtraArgs = append(config.ExtraArgs, secArgs...)

		if p.Security.Encryption.Enabled {
			if opts.AgeKey != "" {
				p.Security.Encryption.KeyFile = opts.AgeKey
				config.ExtraArgs = append(config.ExtraArgs,
					"-v", opts.AgeKey+":/run/secrets/age-key:ro")
			} else if p.Security.Encryption.KeyFile != "" {
				config.ExtraArgs = append(config.ExtraArgs,
					"-v", p.Security.Encryption.KeyFile+":/run/secrets/age-key:ro")
			}
			if opts.Passphrase != "" {
				config.Env["AGE_PASSPHRASE"] = opts.Passphrase
			}
		}

		if p.Security.VPN.Enabled {
			Emit(events, EventInfo, "WireGuard VPN: enabled")
		}
		if p.Security.Firewall.Enabled {
			Emit(events, EventInfo, fmt.Sprintf("Firewall: enabled (default: %s)", p.Security.Firewall.DefaultPolicy))
		}
		if p.Security.Encryption.Enabled {
			Emit(events, EventInfo, "Encryption: enabled (Age + tmpfs)")
		}
	}

	Emit(events, EventInfo, fmt.Sprintf("Starting container: %s", opts.Image))
	return s.runtime.Run(ctx, opts.Image, config)
}
