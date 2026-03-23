package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/discovery/static"
)

// InspectOpts configures an inspect operation.
type InspectOpts struct {
	TargetPath string
}

// InspectResult holds PE analysis results.
type InspectResult struct {
	FileName  string
	FullPath  string
	Arch      string
	Subsystem string
	Imports   []ImportInfo
}

// ImportInfo describes a single DLL import.
type ImportInfo struct {
	Name     string
	IsSystem bool
}

// InspectBinary performs quick PE analysis.
func (s *AppService) InspectBinary(ctx context.Context, opts InspectOpts) (*InspectResult, error) {
	absPath, err := filepath.Abs(opts.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("resolving target path: %w", err)
	}

	arch, subsystem, imports, err := static.GetFileInfo(absPath)
	if err != nil {
		return nil, fmt.Errorf("inspect failed: %w", err)
	}

	result := &InspectResult{
		FileName:  filepath.Base(absPath),
		FullPath:  absPath,
		Arch:      arch,
		Subsystem: subsystem,
	}

	for _, imp := range imports {
		result.Imports = append(result.Imports, ImportInfo{
			Name:     imp,
			IsSystem: isKnownSystemDLL(imp),
		})
	}

	return result, nil
}

func isKnownSystemDLL(dllName string) bool {
	lower := strings.ToLower(dllName)
	systemDLLs := map[string]bool{
		"ntdll.dll": true, "kernel32.dll": true, "kernelbase.dll": true,
		"advapi32.dll": true, "user32.dll": true, "gdi32.dll": true,
		"shell32.dll": true, "ole32.dll": true, "oleaut32.dll": true,
		"combase.dll": true, "msvcrt.dll": true, "ucrtbase.dll": true,
		"ws2_32.dll": true, "rpcrt4.dll": true, "sechost.dll": true,
		"bcrypt.dll": true, "crypt32.dll": true, "shlwapi.dll": true,
		"version.dll": true, "winhttp.dll": true, "wininet.dll": true,
		"comdlg32.dll": true, "imm32.dll": true, "setupapi.dll": true,
	}
	if systemDLLs[lower] {
		return true
	}
	if strings.HasPrefix(lower, "api-ms-win-") || strings.HasPrefix(lower, "ext-ms-") {
		return true
	}
	return false
}
