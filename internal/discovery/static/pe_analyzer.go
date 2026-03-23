package static

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	peparser "github.com/saferwall/pe"
	"github.com/theencryptedafro/appwrap/internal/discovery"
	"github.com/theencryptedafro/appwrap/internal/profile"
	"github.com/theencryptedafro/appwrap/internal/winapi"
)

// PEAnalyzer discovers dependencies by parsing PE import tables.
type PEAnalyzer struct {
	systemDLLs map[string]bool
}

func NewPEAnalyzer(systemDLLs []string) *PEAnalyzer {
	m := make(map[string]bool, len(systemDLLs))
	for _, dll := range systemDLLs {
		m[strings.ToLower(dll)] = true
	}
	return &PEAnalyzer{systemDLLs: m}
}

func (p *PEAnalyzer) Name() string  { return "pe-static" }
func (p *PEAnalyzer) Priority() int { return 10 }

func (p *PEAnalyzer) Supports(target discovery.Target) bool {
	ext := strings.ToLower(filepath.Ext(target.ExePath))
	return ext == ".exe" || ext == ".dll"
}

func (p *PEAnalyzer) Discover(ctx context.Context, target discovery.Target, opts discovery.DiscoverOpts) (*discovery.Result, error) {
	result := &discovery.Result{
		PluginName:  p.Name(),
		Priority:    p.Priority(),
		Environment: make(map[string]string),
	}

	pef, err := peparser.New(target.ExePath, &peparser.Options{})
	if err != nil {
		return nil, fmt.Errorf("open PE file: %w", err)
	}
	if err := pef.Parse(); err != nil {
		return nil, fmt.Errorf("parse PE file: %w", err)
	}

	arch := getMachineArch(pef)
	subsystem := getSubsystem(pef)

	result.Binary = &profile.BinaryInfo{
		Path:       target.ExePath,
		Args:       target.Args,
		WorkingDir: filepath.Dir(target.ExePath),
		Arch:       arch,
		Subsystem:  subsystem,
	}

	result.App = &profile.AppInfo{
		Name: target.AppName,
	}
	if result.App.Name == "" {
		result.App.Name = strings.TrimSuffix(filepath.Base(target.ExePath), filepath.Ext(target.ExePath))
	}

	// Walk the import table recursively
	visited := make(map[string]bool)
	appDir := filepath.Dir(target.ExePath)
	p.walkImports(ctx, pef, appDir, arch, visited, result, opts)

	return result, nil
}

func (p *PEAnalyzer) walkImports(ctx context.Context, pef *peparser.File, appDir, arch string, visited map[string]bool, result *discovery.Result, opts discovery.DiscoverOpts) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	for _, imp := range pef.Imports {
		p.processDLL(ctx, imp.Name, appDir, arch, visited, result, false, opts)
	}

	for _, imp := range pef.DelayImports {
		p.processDLL(ctx, imp.Name, appDir, arch, visited, result, true, opts)
	}
}

func (p *PEAnalyzer) processDLL(ctx context.Context, dllName, appDir, arch string, visited map[string]bool, result *discovery.Result, isDelayLoad bool, opts discovery.DiscoverOpts) {
	lower := strings.ToLower(dllName)
	if visited[lower] {
		return
	}
	visited[lower] = true

	// API set DLLs and ext-ms- DLLs are always system-provided
	isSystem := p.systemDLLs[lower] ||
		strings.HasPrefix(lower, "api-ms-win-") ||
		strings.HasPrefix(lower, "ext-ms-")
	dep := profile.DLLDependency{
		Name:        dllName,
		IsSystem:    isSystem,
		IsDelayLoad: isDelayLoad,
		Source:      "static",
	}

	if !isSystem {
		fullPath := winapi.ResolveDLLPath(dllName, appDir, arch)
		if fullPath != "" {
			dep.FullPath = fullPath
			dep.Version = winapi.GetFileVersion(fullPath)

			// Recursively analyze non-system DLLs
			childPE, err := peparser.New(fullPath, &peparser.Options{})
			if err == nil {
				if err := childPE.Parse(); err == nil {
					p.walkImports(ctx, childPE, appDir, arch, visited, result, opts)
				}
			}
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("could not resolve DLL: %s", dllName))
		}
	}

	result.DLLs = append(result.DLLs, dep)

	if vcRedist := detectVCRedist(lower); vcRedist != "" {
		found := false
		for _, v := range result.VCRedist {
			if v == vcRedist {
				found = true
				break
			}
		}
		if !found {
			result.VCRedist = append(result.VCRedist, vcRedist)
		}
	}
}

func getMachineArch(pef *peparser.File) string {
	switch pef.NtHeader.FileHeader.Machine {
	case peparser.ImageFileMachineI386:
		return "x86"
	case peparser.ImageFileMachineAMD64:
		return "x64"
	case peparser.ImageFileMachineARM64:
		return "arm64"
	default:
		return "x64"
	}
}

func getSubsystem(pef *peparser.File) string {
	if pef.NtHeader.OptionalHeader == nil {
		return "console"
	}
	var sub peparser.ImageOptionalHeaderSubsystemType
	switch oh := pef.NtHeader.OptionalHeader.(type) {
	case *peparser.ImageOptionalHeader32:
		sub = oh.Subsystem
	case *peparser.ImageOptionalHeader64:
		sub = oh.Subsystem
	}
	switch sub {
	case peparser.ImageSubsystemWindowsGUI:
		return "gui"
	default:
		return "console"
	}
}

func detectVCRedist(dllName string) string {
	patterns := map[string]string{
		"vcruntime140":   "vc2015-2022",
		"msvcp140":       "vc2015-2022",
		"concrt140":      "vc2015-2022",
		"vcomp140":       "vc2015-2022",
		"vcruntime140_1": "vc2015-2022",
		"msvcp120":       "vc2013",
		"msvcr120":       "vc2013",
		"msvcp110":       "vc2012",
		"msvcr110":       "vc2012",
		"msvcp100":       "vc2010",
		"msvcr100":       "vc2010",
	}
	base := strings.TrimSuffix(dllName, ".dll")
	if v, ok := patterns[base]; ok {
		return v
	}
	return ""
}

var _ discovery.Plugin = (*PEAnalyzer)(nil)

// GetFileInfo reads basic PE information without full dependency walking.
func GetFileInfo(exePath string) (arch, subsystem string, imports []string, err error) {
	pef, err := peparser.New(exePath, &peparser.Options{})
	if err != nil {
		return "", "", nil, fmt.Errorf("open PE: %w", err)
	}
	if err := pef.Parse(); err != nil {
		return "", "", nil, fmt.Errorf("parse PE: %w", err)
	}

	arch = getMachineArch(pef)
	subsystem = getSubsystem(pef)

	for _, imp := range pef.Imports {
		imports = append(imports, imp.Name)
	}

	return arch, subsystem, imports, nil
}
