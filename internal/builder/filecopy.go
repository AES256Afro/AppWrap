package builder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// StageFiles copies all required application files into the Docker build context.
// Files are placed under contextDir/app/ preserving relative paths from the install dir.
func StageFiles(p *profile.AppProfile, contextDir string) error {
	appDir := filepath.Join(contextDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("create app dir: %w", err)
	}

	installDir := filepath.Dir(p.Binary.Path)

	// Copy the main executable
	if err := copyFileToContext(p.Binary.Path, installDir, appDir); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}

	// Copy non-system DLLs that have resolved paths
	for _, dll := range p.Dependencies.DLLs {
		if dll.IsSystem || dll.FullPath == "" {
			continue
		}
		if err := copyFileToContext(dll.FullPath, installDir, appDir); err != nil {
			// Non-fatal: log and continue
			fmt.Printf("  Warning: could not copy DLL %s: %v\n", dll.Name, err)
		}
	}

	// Copy extra files from filesystem requirements
	for _, f := range p.FileSystem.ExtraFiles {
		if err := copyFileToContext(f, installDir, appDir); err != nil {
			fmt.Printf("  Warning: could not copy extra file %s: %v\n", f, err)
		}
	}

	// Copy config files
	for _, f := range p.FileSystem.ConfigPaths {
		if err := copyFileToContext(f, installDir, appDir); err != nil {
			fmt.Printf("  Warning: could not copy config %s: %v\n", f, err)
		}
	}

	return nil
}

// copyFileToContext copies a single file into the build context.
// If the file is under installDir, it preserves the relative path.
// Otherwise, it goes into the app root.
func copyFileToContext(srcPath, installDir, appDir string) error {
	srcPath = filepath.Clean(srcPath)
	installDir = filepath.Clean(installDir)

	var relPath string
	if strings.HasPrefix(strings.ToLower(srcPath), strings.ToLower(installDir)) {
		var err error
		relPath, err = filepath.Rel(installDir, srcPath)
		if err != nil {
			relPath = filepath.Base(srcPath)
		}
	} else {
		relPath = filepath.Base(srcPath)
	}

	destPath := filepath.Join(appDir, relPath)

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	return copyFile(srcPath, destPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
