package discovery

import "github.com/theencryptedafro/appwrap/internal/profile"

// Result holds the partial output from a single discovery plugin.
// Multiple Results are merged into a final AppProfile.
type Result struct {
	PluginName string
	Priority   int

	// Binary info discovered by this plugin
	Binary *profile.BinaryInfo

	// App info (name, version, publisher)
	App *profile.AppInfo

	// Dependencies found
	DLLs     []profile.DLLDependency
	COM      []profile.COMDependency
	VCRedist []string
	DotNet   *profile.DotNetReq
	DirectX  *profile.DirectXReq
	Fonts    []string

	// Registry keys observed
	Registry []profile.RegistryEntry

	// Environment variables
	Environment map[string]string

	// File system requirements
	FileSystem *profile.FileSystemReqs

	// Network requirements
	Network *profile.NetworkReqs

	// Service dependencies
	Services []profile.ServiceDep

	// Package manager references
	Packages []profile.PackageRef

	// Errors encountered (non-fatal)
	Warnings []string
}
