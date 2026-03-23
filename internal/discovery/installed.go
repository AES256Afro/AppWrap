package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// InstalledApp represents a discovered application.
type InstalledApp struct {
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	Version     string `json:"version"`
	InstallPath string `json:"installPath"`
	ExePath     string `json:"exePath"`
	Source      string `json:"source"` // "registry", "startmenu", "winget"
}

// ScanInstalled discovers all installed applications on the system.
// It queries the Windows registry, Start Menu shortcuts, and optionally winget.
func ScanInstalled() ([]InstalledApp, error) {
	seen := map[string]bool{}
	var apps []InstalledApp

	// 1. Windows Registry (3 hives)
	regKeys := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	for _, rk := range regKeys {
		k, err := registry.OpenKey(rk.root, rk.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}

		for _, name := range names {
			sub, err := registry.OpenKey(rk.root, rk.path+`\`+name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			displayName, _, _ := sub.GetStringValue("DisplayName")
			installLocation, _, _ := sub.GetStringValue("InstallLocation")
			displayIcon, _, _ := sub.GetStringValue("DisplayIcon")
			publisher, _, _ := sub.GetStringValue("Publisher")
			version, _, _ := sub.GetStringValue("DisplayVersion")
			systemComponent, _, sysErr := sub.GetIntegerValue("SystemComponent")
			sub.Close()

			if displayName == "" {
				continue
			}
			// Skip system components
			if sysErr == nil && systemComponent == 1 {
				continue
			}
			// Skip updates
			if strings.HasPrefix(displayName, "Update for") || strings.HasPrefix(displayName, "Security Update") || strings.HasPrefix(displayName, "Hotfix for") {
				continue
			}

			key := strings.ToLower(displayName)
			if seen[key] {
				continue
			}
			seen[key] = true

			exePath := resolveExePath(displayName, installLocation, displayIcon)

			apps = append(apps, InstalledApp{
				Name:        displayName,
				Publisher:   publisher,
				Version:     version,
				InstallPath: installLocation,
				ExePath:     exePath,
				Source:      "registry",
			})
		}
	}

	// 2. Start Menu shortcuts
	startMenuDirs := []string{
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`),
		filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs`),
	}

	for _, dir := range startMenuDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(path), ".lnk") {
				return nil
			}

			name := strings.TrimSuffix(info.Name(), ".lnk")
			key := strings.ToLower(name)
			if seen[key] {
				return nil
			}

			// Try to resolve the .lnk target using PowerShell
			target := resolveLnk(path)
			if target == "" || !strings.HasSuffix(strings.ToLower(target), ".exe") {
				return nil
			}

			seen[key] = true
			apps = append(apps, InstalledApp{
				Name:        name,
				ExePath:     target,
				InstallPath: filepath.Dir(target),
				Source:      "startmenu",
			})
			return nil
		})
	}

	// Sort by name
	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps, nil
}

// resolveExePath tries to find the .exe path from registry info.
func resolveExePath(displayName, installLocation, displayIcon string) string {
	// 1. Try displayIcon — often points directly to the exe.
	//    Formats: "C:\path\app.exe,0"  or  C:\path\app.exe,0  or  "C:\path\app.exe"
	if displayIcon != "" {
		p := displayIcon
		// Strip quotes first
		p = strings.Trim(p, `"`)
		// Strip icon index: find last comma that's followed by a number
		if idx := strings.LastIndex(p, ","); idx > 0 {
			suffix := strings.TrimSpace(p[idx+1:])
			isNum := true
			for _, c := range suffix {
				if c < '0' || c > '9' {
					isNum = false
					break
				}
			}
			if isNum {
				p = p[:idx]
			}
		}
		p = strings.TrimSpace(p)
		if strings.HasSuffix(strings.ToLower(p), ".exe") {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// 2. Try App Paths registry (Windows stores exe paths for many apps here)
	appPathsKeys := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`},
	}
	// Derive possible exe names from display name
	possibleExeNames := deriveExeNames(displayName)
	for _, rk := range appPathsKeys {
		for _, exeName := range possibleExeNames {
			sub, err := registry.OpenKey(rk.root, rk.path+`\`+exeName, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			val, _, err := sub.GetStringValue("")
			sub.Close()
			if err == nil {
				val = strings.Trim(val, `"`)
				if _, statErr := os.Stat(val); statErr == nil {
					return val
				}
			}
		}
	}

	// 3. Check install location — top level AND one level of subdirectories
	if installLocation != "" {
		installLocation = strings.Trim(installLocation, `"`)
		if info, err := os.Stat(installLocation); err == nil && info.IsDir() {
			// Check top level
			if exe := findExeInDir(installLocation); exe != "" {
				return exe
			}
			// Check one level of subdirectories (e.g., Chrome's "Application" folder)
			entries, err := os.ReadDir(installLocation)
			if err == nil {
				for _, e := range entries {
					if e.IsDir() {
						subDir := filepath.Join(installLocation, e.Name())
						if exe := findExeInDir(subDir); exe != "" {
							return exe
						}
					}
				}
			}
		}
	}

	// 4. Check well-known install locations for common apps
	for _, base := range []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		filepath.Join(os.Getenv("LOCALAPPDATA")),
	} {
		if base == "" {
			continue
		}
		for _, exeName := range possibleExeNames {
			// Try: base/DisplayName/exeName  and  base/DisplayName/Application/exeName
			candidates := []string{
				filepath.Join(base, displayName, exeName),
				filepath.Join(base, displayName, "Application", exeName),
			}
			// Also try without spaces: "Google Chrome" -> "Google\Chrome"
			parts := strings.Fields(displayName)
			if len(parts) > 1 {
				candidates = append(candidates,
					filepath.Join(base, parts[0], parts[len(parts)-1], "Application", exeName),
					filepath.Join(base, parts[0], parts[len(parts)-1], exeName),
				)
			}
			for _, cand := range candidates {
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
	}

	return ""
}

// deriveExeNames guesses possible .exe filenames from a display name.
func deriveExeNames(displayName string) []string {
	if displayName == "" {
		return nil
	}
	var names []string
	// Exact name lowercase
	base := strings.ToLower(displayName)
	// Remove version info like "25.01 (x64)"
	if idx := strings.IndexAny(base, "0123456789"); idx > 0 {
		base = strings.TrimSpace(base[:idx])
	}
	base = strings.ReplaceAll(base, " ", "")
	names = append(names, base+".exe")

	// Last word (e.g., "Google Chrome" -> "chrome.exe")
	parts := strings.Fields(strings.ToLower(displayName))
	if len(parts) > 1 {
		names = append(names, parts[len(parts)-1]+".exe")
	}

	// Full name with spaces preserved
	names = append(names, strings.ReplaceAll(strings.ToLower(displayName), " ", "")+".exe")

	// Deduplicate
	seen := map[string]bool{}
	var result []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result
}

// findExeInDir returns the first .exe found in a directory.
func findExeInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".exe") {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// resolveLnk resolves a Windows .lnk shortcut to its target path.
func resolveLnk(lnkPath string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`(New-Object -ComObject WScript.Shell).CreateShortcut('%s').TargetPath`, strings.ReplaceAll(lnkPath, "'", "''")))
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
