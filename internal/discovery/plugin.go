package discovery

import "context"

// Plugin is the interface all discovery methods implement.
type Plugin interface {
	// Name returns a human-readable name for this plugin.
	Name() string

	// Supports returns true if this plugin can analyze the given target.
	Supports(target Target) bool

	// Discover runs the analysis and returns partial results.
	Discover(ctx context.Context, target Target, opts DiscoverOpts) (*Result, error)

	// Priority determines merge order. Higher priority results override lower ones.
	Priority() int
}

// Target represents the application to analyze.
type Target struct {
	ExePath    string   // Resolved absolute path to .exe
	Args       []string // Default arguments
	Shortcut   string   // Original .lnk path (if resolved from shortcut)
	AppName    string   // Friendly name (from version info or shortcut)
	InstallDir string   // Inferred install directory
}

// DiscoverOpts controls discovery behavior.
type DiscoverOpts struct {
	Verbose        bool
	RuntimeTrace   bool          // Enable runtime tracing (Phase 2)
	TraceDuration  int           // Seconds to trace (Phase 2)
	SkipSystemDLLs []string      // Additional DLLs to skip
}
