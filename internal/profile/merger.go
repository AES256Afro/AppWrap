package profile

import (
	"time"
	"runtime"
)

// MergeResults combines multiple discovery results into a single AppProfile.
// Results should be sorted by priority (lowest first) so higher-priority
// results override earlier ones.
type PartialResult struct {
	PluginName  string
	Binary      *BinaryInfo
	App         *AppInfo
	DLLs        []DLLDependency
	COM         []COMDependency
	VCRedist    []string
	DotNet      *DotNetReq
	DirectX     *DirectXReq
	Fonts       []string
	Registry    []RegistryEntry
	Environment map[string]string
	FileSystem  *FileSystemReqs
	Network     *NetworkReqs
	Services    []ServiceDep
	Packages    []PackageRef
	Warnings    []string
}

func MergeResults(results []PartialResult) *AppProfile {
	p := &AppProfile{
		SchemaVersion: CurrentSchemaVersion,
		Environment:   make(map[string]string),
		Metadata: Metadata{
			CreatedAt: time.Now(),
			CreatedBy: "appwrap",
			HostOS:    runtime.GOOS + "/" + runtime.GOARCH,
		},
	}

	dllSeen := make(map[string]int) // dll name -> index in p.Dependencies.DLLs
	comSeen := make(map[string]bool)
	vcSeen := make(map[string]bool)
	scanMethods := make(map[string]bool)

	for _, r := range results {
		scanMethods[r.PluginName] = true

		// Binary info: later results override
		if r.Binary != nil {
			p.Binary = *r.Binary
		}

		// App info: later results override
		if r.App != nil {
			if r.App.Name != "" {
				p.App.Name = r.App.Name
			}
			if r.App.Version != "" {
				p.App.Version = r.App.Version
			}
			if r.App.Publisher != "" {
				p.App.Publisher = r.App.Publisher
			}
		}

		// DLLs: deduplicate by name, later results update existing entries
		for _, dll := range r.DLLs {
			key := toLower(dll.Name)
			if idx, exists := dllSeen[key]; exists {
				// Update with richer info from later plugins
				existing := &p.Dependencies.DLLs[idx]
				if dll.FullPath != "" {
					existing.FullPath = dll.FullPath
				}
				if dll.Version != "" {
					existing.Version = dll.Version
				}
				if dll.Source != "" {
					existing.Source = dll.Source
				}
			} else {
				dllSeen[key] = len(p.Dependencies.DLLs)
				p.Dependencies.DLLs = append(p.Dependencies.DLLs, dll)
			}
		}

		// COM objects
		for _, com := range r.COM {
			if !comSeen[com.CLSID] {
				comSeen[com.CLSID] = true
				p.Dependencies.COM = append(p.Dependencies.COM, com)
			}
		}

		// VC++ runtimes
		for _, vc := range r.VCRedist {
			if !vcSeen[vc] {
				vcSeen[vc] = true
				p.Dependencies.VCRedist = append(p.Dependencies.VCRedist, vc)
			}
		}

		// DotNet: later overrides
		if r.DotNet != nil {
			p.Dependencies.DotNet = r.DotNet
		}

		// DirectX: later overrides
		if r.DirectX != nil {
			p.Dependencies.DirectX = r.DirectX
		}

		// Fonts: append unique
		for _, f := range r.Fonts {
			if !contains(p.Dependencies.Fonts, f) {
				p.Dependencies.Fonts = append(p.Dependencies.Fonts, f)
			}
		}

		// Registry: append all
		p.Registry = append(p.Registry, r.Registry...)

		// Environment: merge maps
		for k, v := range r.Environment {
			p.Environment[k] = v
		}

		// FileSystem: merge
		if r.FileSystem != nil {
			p.FileSystem.ConfigPaths = appendUnique(p.FileSystem.ConfigPaths, r.FileSystem.ConfigPaths...)
			p.FileSystem.DataDirs = appendUnique(p.FileSystem.DataDirs, r.FileSystem.DataDirs...)
			p.FileSystem.TempDirs = appendUnique(p.FileSystem.TempDirs, r.FileSystem.TempDirs...)
			p.FileSystem.ExtraFiles = appendUnique(p.FileSystem.ExtraFiles, r.FileSystem.ExtraFiles...)
		}

		// Network: later overrides
		if r.Network != nil {
			p.Network = *r.Network
		}

		// Services: append
		p.Services = append(p.Services, r.Services...)

		// Packages: append
		p.Packages = append(p.Packages, r.Packages...)
	}

	// Set scan methods
	for m := range scanMethods {
		p.Metadata.ScanMethods = append(p.Metadata.ScanMethods, m)
	}

	// Determine confidence
	p.Metadata.Confidence = "low"
	if len(scanMethods) >= 2 {
		p.Metadata.Confidence = "medium"
	}
	if scanMethods["etw-runtime"] {
		p.Metadata.Confidence = "high"
	}

	// Set default build config
	if p.Build.Strategy == "" {
		p.Build.Strategy = "wine"
	}

	return p
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func appendUnique(dest []string, items ...string) []string {
	seen := make(map[string]bool, len(dest))
	for _, d := range dest {
		seen[d] = true
	}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			dest = append(dest, item)
		}
	}
	return dest
}
