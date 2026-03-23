package util

import (
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath converts Windows paths to forward slashes and resolves symlinks.
func NormalizePath(path string) string {
	path = filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return path
}

// ToLinuxPath converts a Windows path to a Linux-style path for use inside containers.
func ToLinuxPath(winPath string) string {
	// Remove drive letter
	if len(winPath) >= 2 && winPath[1] == ':' {
		winPath = winPath[2:]
	}
	return strings.ReplaceAll(winPath, `\`, "/")
}

// InferInstallDir tries to determine the installation directory from an exe path.
// Walks up from the exe looking for common install dir patterns.
func InferInstallDir(exePath string) string {
	dir := filepath.Dir(exePath)

	// Common patterns: exe is in root of install dir, or in a bin/ subfolder
	base := strings.ToLower(filepath.Base(dir))
	if base == "bin" || base == "app" || base == "x64" || base == "x86" {
		return filepath.Dir(dir)
	}

	return dir
}

// LoadSystemDLLs reads the known system DLLs list from a YAML config file.
func LoadSystemDLLs(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultSystemDLLs(), nil
	}

	var dlls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimSpace(line)
		if line != "" {
			dlls = append(dlls, line)
		}
	}

	if len(dlls) == 0 {
		return defaultSystemDLLs(), nil
	}
	return dlls, nil
}

func defaultSystemDLLs() []string {
	return []string{
		"ntdll.dll", "kernel32.dll", "kernelbase.dll", "advapi32.dll",
		"user32.dll", "gdi32.dll", "shell32.dll", "ole32.dll", "oleaut32.dll",
		"combase.dll", "msvcrt.dll", "ucrtbase.dll", "ws2_32.dll",
		"rpcrt4.dll", "sechost.dll", "bcrypt.dll", "crypt32.dll",
		"shlwapi.dll", "version.dll", "winhttp.dll", "wininet.dll",
		"setupapi.dll", "cfgmgr32.dll", "imm32.dll", "comdlg32.dll",
	}
}
