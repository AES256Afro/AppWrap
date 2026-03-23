package discovery

import (
	"context"
	"fmt"
	"sort"
)

// Engine orchestrates all discovery plugins.
type Engine struct {
	plugins []Plugin
}

func NewEngine() *Engine {
	return &Engine{}
}

// Register adds a discovery plugin.
func (e *Engine) Register(p Plugin) {
	e.plugins = append(e.plugins, p)
}

// Run executes all compatible plugins against the target, sorted by priority.
// Returns all results for merging.
func (e *Engine) Run(ctx context.Context, target Target, opts DiscoverOpts) ([]*Result, error) {
	// Sort plugins by priority (lowest first, so higher priority runs last and overrides)
	sorted := make([]Plugin, len(e.plugins))
	copy(sorted, e.plugins)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})

	var results []*Result
	for _, plugin := range sorted {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		if !plugin.Supports(target) {
			continue
		}

		if opts.Verbose {
			fmt.Printf("[discovery] Running plugin: %s\n", plugin.Name())
		}

		result, err := plugin.Discover(ctx, target, opts)
		if err != nil {
			fmt.Printf("[discovery] Warning: plugin %s failed: %v\n", plugin.Name(), err)
			continue
		}

		results = append(results, result)

		if opts.Verbose {
			fmt.Printf("[discovery] Plugin %s found %d DLLs, %d registry entries\n",
				plugin.Name(), len(result.DLLs), len(result.Registry))
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no plugins could analyze target: %s", target.ExePath)
	}

	return results, nil
}
