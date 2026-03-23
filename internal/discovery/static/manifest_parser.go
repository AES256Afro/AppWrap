package static

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/discovery"
	"github.com/theencryptedafro/appwrap/internal/profile"
)

// ManifestParser reads SxS application manifests to discover
// required assemblies like VC++ runtimes, common controls, etc.
type ManifestParser struct{}

func NewManifestParser() *ManifestParser { return &ManifestParser{} }

func (m *ManifestParser) Name() string     { return "manifest-parser" }
func (m *ManifestParser) Priority() int     { return 5 }

func (m *ManifestParser) Supports(target discovery.Target) bool {
	ext := strings.ToLower(filepath.Ext(target.ExePath))
	return ext == ".exe"
}

func (m *ManifestParser) Discover(ctx context.Context, target discovery.Target, opts discovery.DiscoverOpts) (*discovery.Result, error) {
	result := &discovery.Result{
		PluginName: m.Name(),
		Priority:   m.Priority(),
	}

	// Look for external manifest: app.exe.manifest
	manifestPath := target.ExePath + ".manifest"
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		// No external manifest — that's fine, the PE analyzer handles embedded ones
		return result, nil
	}

	if err := m.parseManifest(data, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("manifest parse error: %v", err))
	}

	return result, nil
}

type assembly struct {
	XMLName             xml.Name             `xml:"assembly"`
	DependentAssemblies []dependentAssembly  `xml:"dependency>dependentAssembly"`
}

type dependentAssembly struct {
	AssemblyIdentity assemblyIdentity `xml:"assemblyIdentity"`
}

type assemblyIdentity struct {
	Type            string `xml:"type,attr"`
	Name            string `xml:"name,attr"`
	Version         string `xml:"version,attr"`
	ProcessorArch   string `xml:"processorArchitecture,attr"`
	PublicKeyToken  string `xml:"publicKeyToken,attr"`
}

func (m *ManifestParser) parseManifest(data []byte, result *discovery.Result) error {
	var asm assembly
	if err := xml.Unmarshal(data, &asm); err != nil {
		return fmt.Errorf("unmarshal manifest: %w", err)
	}

	for _, dep := range asm.DependentAssemblies {
		id := dep.AssemblyIdentity

		// Detect VC++ runtime assemblies
		if vcRedist := detectVCRedistFromAssembly(id.Name, id.Version); vcRedist != "" {
			result.VCRedist = append(result.VCRedist, vcRedist)
		}

		// Detect common controls v6 (implies ComCtl32 dependency)
		if strings.Contains(strings.ToLower(id.Name), "microsoft.windows.common-controls") {
			result.DLLs = append(result.DLLs, profile.DLLDependency{
				Name:     "comctl32.dll",
				IsSystem: true,
				Source:   "manifest",
			})
		}
	}

	return nil
}

func detectVCRedistFromAssembly(name, version string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "microsoft.vc140") || strings.Contains(lower, "microsoft.vc150") || strings.Contains(lower, "microsoft.vc160"):
		return "vc2015-2022"
	case strings.Contains(lower, "microsoft.vc120"):
		return "vc2013"
	case strings.Contains(lower, "microsoft.vc110"):
		return "vc2012"
	case strings.Contains(lower, "microsoft.vc100"):
		return "vc2010"
	case strings.Contains(lower, "microsoft.vc90"):
		return "vc2008"
	case strings.Contains(lower, "microsoft.vc80"):
		return "vc2005"
	}
	return ""
}

var _ discovery.Plugin = (*ManifestParser)(nil)
