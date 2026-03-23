package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/discovery"
	"github.com/theencryptedafro/appwrap/internal/discovery/static"
	"github.com/theencryptedafro/appwrap/internal/profile"
	"github.com/theencryptedafro/appwrap/internal/util"
)

// ScanOpts configures a scan operation.
type ScanOpts struct {
	TargetPath string
	Strategy   string // wine, windows-servercore, windows-nanoserver
	Format     string // yaml or json
	OutputPath string
	Encrypt    bool
	Firewall   string // deny or allow
	VPNConfig  string // path to WireGuard .conf
	Verbose    bool
}

// ScanResult holds the output of a scan.
type ScanResult struct {
	Profile    *profile.AppProfile
	OutputPath string
	Warnings   []string
}

// ScanApp discovers dependencies and generates a container profile.
// Events are streamed to the channel for real-time UI updates.
func (s *AppService) ScanApp(ctx context.Context, opts ScanOpts, events chan<- Event) (*ScanResult, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(opts.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("target not found: %s", absPath)
	}

	target := discovery.Target{
		ExePath: absPath,
	}

	// Handle .lnk shortcuts
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == ".lnk" {
		Emit(events, EventInfo, fmt.Sprintf("Resolving shortcut: %s", absPath))
		exePath, args, workDir, err := static.ResolveShortcut(absPath)
		if err != nil {
			return nil, fmt.Errorf("resolve shortcut: %w", err)
		}
		target.ExePath = exePath
		target.Args = args
		target.Shortcut = absPath
		if workDir != "" {
			target.InstallDir = workDir
		}
		Emit(events, EventInfo, fmt.Sprintf("Target: %s", exePath))
	}

	// Infer install directory
	if target.InstallDir == "" {
		target.InstallDir = util.InferInstallDir(target.ExePath)
	}

	// Load system DLL list
	systemDLLs, err := util.LoadSystemDLLs(filepath.Join(s.configDir, "known_system_dlls.yaml"))
	if err != nil {
		Emit(events, EventWarning, "could not load system DLL list: "+err.Error())
	}

	// Set up discovery engine
	engine := discovery.NewEngine()
	engine.Register(static.NewManifestParser())
	engine.Register(static.NewPEAnalyzer(systemDLLs))

	Emit(events, EventInfo, fmt.Sprintf("Scanning: %s", target.ExePath))
	EmitProgress(events, 10, "Running discovery plugins...")

	// Run discovery
	results, err := engine.Run(ctx, target, discovery.DiscoverOpts{
		Verbose: opts.Verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	EmitProgress(events, 50, "Merging results...")

	// Convert and merge
	var partials []profile.PartialResult
	var warnings []string
	for _, r := range results {
		partials = append(partials, profile.PartialResult{
			PluginName:  r.PluginName,
			Binary:      r.Binary,
			App:         r.App,
			DLLs:        r.DLLs,
			COM:         r.COM,
			VCRedist:    r.VCRedist,
			DotNet:      r.DotNet,
			DirectX:     r.DirectX,
			Fonts:       r.Fonts,
			Registry:    r.Registry,
			Environment: r.Environment,
			FileSystem:  r.FileSystem,
			Network:     r.Network,
			Services:    r.Services,
			Packages:    r.Packages,
			Warnings:    r.Warnings,
		})
		warnings = append(warnings, r.Warnings...)
	}

	p := profile.MergeResults(partials)

	EmitProgress(events, 70, "Applying configuration...")

	// Apply strategy
	if opts.Strategy != "" {
		p.Build.Strategy = opts.Strategy
	}

	// Apply security settings
	if opts.Encrypt {
		p.Security.Encryption.Enabled = true
		Emit(events, EventInfo, "Encryption: enabled (run 'appwrap keygen' to generate keys)")
	}
	if opts.Firewall != "" {
		p.Security.Firewall.Enabled = true
		p.Security.Firewall.DefaultPolicy = opts.Firewall
		p.Security.Firewall.AllowDNS = true
		p.Security.Firewall.AllowLoopback = true
		Emit(events, EventInfo, fmt.Sprintf("Firewall: enabled (default: %s)", opts.Firewall))
	}
	if opts.VPNConfig != "" {
		p.Security.VPN.Enabled = true
		p.Security.VPN.Provider = "wireguard"
		p.Security.VPN.ConfigFile = opts.VPNConfig
		p.Security.VPN.KillSwitch = true
		Emit(events, EventInfo, fmt.Sprintf("VPN: WireGuard enabled (config: %s)", opts.VPNConfig))
	}

	// Send warnings
	for _, w := range warnings {
		Emit(events, EventWarning, w)
	}

	EmitProgress(events, 85, "Validating profile...")

	// Validate
	v := profile.Validate(p)
	for _, w := range v.Warnings {
		Emit(events, EventWarning, fmt.Sprintf("Validation: %s", w))
	}
	for _, e := range v.Errors {
		Emit(events, EventError, fmt.Sprintf("Validation: %s", e))
	}

	// Determine output path
	outputPath := opts.OutputPath
	profileFileName := func() string {
		name := strings.ToLower(p.App.Name)
		name = strings.ReplaceAll(name, " ", "-")
		format := opts.Format
		if format == "" {
			format = "yaml"
		}
		if format == "json" {
			return name + "-profile.json"
		}
		return name + "-profile.yaml"
	}()

	if outputPath == "" {
		outputPath = profileFileName
	} else if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		// If outputPath is a directory, write the profile file inside it
		outputPath = filepath.Join(outputPath, profileFileName)
	}

	// Ensure the output directory exists
	if dir := filepath.Dir(outputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create output directory: %w", err)
		}
	}

	EmitProgress(events, 95, "Writing profile...")

	// Write profile
	format := opts.Format
	if format == "" {
		format = "yaml"
	}
	switch format {
	case "json":
		err = profile.WriteJSON(p, outputPath)
	default:
		err = profile.WriteYAML(p, outputPath)
	}
	if err != nil {
		return nil, fmt.Errorf("write profile: %w", err)
	}

	// Summary info
	totalDLLs := len(p.Dependencies.DLLs)
	appDLLs := 0
	for _, d := range p.Dependencies.DLLs {
		if !d.IsSystem {
			appDLLs++
		}
	}
	Emit(events, EventInfo, fmt.Sprintf("App: %s | Arch: %s | DLLs: %d (%d app, %d system)",
		p.App.Name, p.Binary.Arch, totalDLLs, appDLLs, totalDLLs-appDLLs))

	EmitComplete(events, fmt.Sprintf("Profile written to: %s", outputPath))

	return &ScanResult{
		Profile:    p,
		OutputPath: outputPath,
		Warnings:   warnings,
	}, nil
}
