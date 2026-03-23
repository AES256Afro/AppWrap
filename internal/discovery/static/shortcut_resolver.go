package static

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveShortcut extracts the target exe, arguments, and working directory
// from a Windows .lnk shortcut file using PowerShell.
func ResolveShortcut(lnkPath string) (exePath string, args []string, workingDir string, err error) {
	absPath, err := filepath.Abs(lnkPath)
	if err != nil {
		return "", nil, "", fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", nil, "", fmt.Errorf("shortcut not found: %w", err)
	}

	// Use PowerShell to read the .lnk file
	script := fmt.Sprintf(
		`$s = (New-Object -ComObject WScript.Shell).CreateShortcut('%s'); `+
			`Write-Output $s.TargetPath; `+
			`Write-Output '---SEPARATOR---'; `+
			`Write-Output $s.Arguments; `+
			`Write-Output '---SEPARATOR---'; `+
			`Write-Output $s.WorkingDirectory`,
		strings.ReplaceAll(absPath, "'", "''"),
	)

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return "", nil, "", fmt.Errorf("powershell: %w", err)
	}

	parts := strings.Split(string(out), "---SEPARATOR---")
	if len(parts) < 3 {
		return "", nil, "", fmt.Errorf("unexpected powershell output")
	}

	exePath = strings.TrimSpace(parts[0])
	argStr := strings.TrimSpace(parts[1])
	workingDir = strings.TrimSpace(parts[2])

	if argStr != "" {
		args = splitArgs(argStr)
	}

	return exePath, args, workingDir, nil
}

// splitArgs does a basic split of a command-line argument string,
// respecting double-quoted strings.
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
