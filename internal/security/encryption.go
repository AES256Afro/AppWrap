package security

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// findAgeBinary searches common installation paths for the age CLI.
func findAgeBinary(name string) string {
	// Check PATH first
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// Check common WinGet install locations
	home, _ := os.UserHomeDir()
	searchDirs := []string{
		filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"),
	}

	// Search WinGet packages directory for age
	packagesDir := filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Packages")
	entries, _ := os.ReadDir(packagesDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "age") {
			searchDirs = append(searchDirs,
				filepath.Join(packagesDir, e.Name(), "age"))
		}
	}

	// Also check Program Files
	searchDirs = append(searchDirs,
		`C:\Program Files\age`,
		`C:\Program Files (x86)\age`,
	)

	exe := name
	if !strings.HasSuffix(exe, ".exe") {
		exe += ".exe"
	}

	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, exe)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// EncryptAppFiles encrypts all files in the app directory using Age.
// The encrypted files replace the originals, with an .age extension.
// Returns the path to the generated decryption entrypoint script.
func EncryptAppFiles(contextDir string, cfg profile.EncryptionConfig) error {
	appDir := filepath.Join(contextDir, "app")

	// Verify age is available on the build host
	ageBin := findAgeBinary("age")
	if ageBin == "" {
		return fmt.Errorf("age CLI not found — install it: winget install FiloSottile.age")
	}

	// Create encrypted output dir
	encDir := filepath.Join(contextDir, "app-encrypted")
	if err := os.MkdirAll(encDir, 0755); err != nil {
		return fmt.Errorf("create encrypted dir: %w", err)
	}

	// Walk all files and encrypt each one
	err := filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(appDir, path)
		if err != nil {
			return err
		}

		encPath := filepath.Join(encDir, relPath+".age")
		if err := os.MkdirAll(filepath.Dir(encPath), 0755); err != nil {
			return err
		}

		return encryptFile(ageBin, path, encPath, cfg)
	})
	if err != nil {
		return fmt.Errorf("encrypt files: %w", err)
	}

	// Remove the unencrypted app dir and rename encrypted dir
	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("remove unencrypted app dir: %w", err)
	}
	if err := os.Rename(encDir, appDir); err != nil {
		return fmt.Errorf("rename encrypted dir: %w", err)
	}

	// Write the decryption entrypoint script
	return writeDecryptScript(contextDir, cfg)
}

func encryptFile(ageBin, src, dst string, cfg profile.EncryptionConfig) error {
	var args []string

	if cfg.Passphrase {
		args = []string{"-p", "-o", dst, src}
	} else if cfg.Recipient != "" {
		args = []string{"-r", cfg.Recipient, "-o", dst, src}
	} else if cfg.KeyFile != "" {
		// For key-based encryption, extract the recipient from the key file
		recipient, err := extractRecipientFromKeyFile(cfg.KeyFile)
		if err != nil {
			return fmt.Errorf("extract recipient from key file: %w", err)
		}
		args = []string{"-r", recipient, "-o", dst, src}
	} else {
		return fmt.Errorf("encryption enabled but no recipient, keyFile, or passphrase specified")
	}

	cmd := exec.Command(ageBin, args...)
	cmd.Stderr = os.Stderr
	if cfg.Passphrase {
		// For passphrase mode during build, we need stdin
		cmd.Stdin = os.Stdin
	}
	return cmd.Run()
}

func extractRecipientFromKeyFile(keyFile string) (string, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// Public key line in age key files: "# public key: age1..."
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimPrefix(line, "# public key: "), nil
		}
	}
	return "", fmt.Errorf("no public key found in %s", keyFile)
}

// writeDecryptScript generates the shell script that runs inside the container
// to decrypt app files from /app (encrypted) to /run/app (tmpfs) at startup.
func writeDecryptScript(contextDir string, cfg profile.EncryptionConfig) error {
	script := `#!/bin/bash
set -e

ENCRYPTED_DIR="/home/appuser/app"
DECRYPT_DIR="/run/appwrap"

# Mount tmpfs for decrypted files (RAM-only, never touches disk)
mkdir -p "$DECRYPT_DIR"
mount -t tmpfs -o size=512m,mode=0700 tmpfs "$DECRYPT_DIR"

echo "[appwrap] Decrypting application files to tmpfs..."

# Decrypt all .age files
find "$ENCRYPTED_DIR" -name '*.age' | while read -r enc_file; do
    rel_path="${enc_file#$ENCRYPTED_DIR/}"
    out_path="$DECRYPT_DIR/${rel_path%.age}"
    mkdir -p "$(dirname "$out_path")"
`

	if cfg.Passphrase {
		// Passphrase provided via AGE_PASSPHRASE env var
		script += `    echo "$AGE_PASSPHRASE" | age -d -o "$out_path" "$enc_file"
`
	} else {
		// Key file mounted into container at /run/secrets/age-key
		script += `    age -d -i /run/secrets/age-key -o "$out_path" "$enc_file"
`
	}

	script += `done

echo "[appwrap] Decryption complete. Files in tmpfs."

# Make decrypted files executable
chmod -R +x "$DECRYPT_DIR" 2>/dev/null || true

# Execute the actual application from the decrypted tmpfs
exec "$@"
`

	scriptPath := filepath.Join(contextDir, "decrypt-entrypoint.sh")
	return os.WriteFile(scriptPath, []byte(script), 0755)
}

// GenerateKeyPair creates a new Age keypair and saves it to the specified directory.
// Returns the public key (recipient) string.
func GenerateKeyPair(outputDir string) (recipient string, err error) {
	keygenBin := findAgeBinary("age-keygen")
	if keygenBin == "" {
		return "", fmt.Errorf("age-keygen not found — install it: winget install FiloSottile.age")
	}

	keyFile := filepath.Join(outputDir, "appwrap-age-key.txt")

	cmd := exec.Command(keygenBin, "-o", keyFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("age-keygen failed: %w\n%s", err, out)
	}

	// Extract public key from output or file
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			recipient = strings.TrimPrefix(line, "# public key: ")
			break
		}
	}

	if recipient == "" {
		// Try parsing from keygen stdout
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "Public key: ") {
				recipient = strings.TrimPrefix(line, "Public key: ")
				break
			}
		}
	}

	if recipient == "" {
		return "", fmt.Errorf("could not extract public key from age-keygen output")
	}

	fmt.Printf("Age keypair generated:\n  Key file: %s\n  Public key: %s\n", keyFile, recipient)
	fmt.Printf("\nKeep the key file safe — it's needed to start the container.\n")
	fmt.Printf("Pass it at runtime: docker run -v %s:/run/secrets/age-key:ro ...\n", keyFile)

	// Also write just the public key for convenience
	pubFile := filepath.Join(outputDir, "appwrap-age-recipient.txt")
	os.WriteFile(pubFile, []byte(recipient+"\n"), 0644)

	return recipient, nil
}

// DockerfileSnippet returns the Dockerfile lines needed for Age decryption support.
func DockerfileSnippet() string {
	return `
# --- Age Encryption Support ---
USER root
RUN apt-get update && apt-get install -y --no-install-recommends age && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
COPY decrypt-entrypoint.sh /usr/local/bin/decrypt-entrypoint.sh
RUN chmod +x /usr/local/bin/decrypt-entrypoint.sh
USER appuser
`
}

// DecryptEntrypoint returns the ENTRYPOINT override for encrypted containers.
func DecryptEntrypoint() string {
	return `ENTRYPOINT ["/usr/local/bin/decrypt-entrypoint.sh"]`
}

// DockerRunArgs returns extra docker run arguments needed for encryption.
func DockerRunArgs(cfg profile.EncryptionConfig) []string {
	var args []string
	// Need SYS_ADMIN for tmpfs mount inside container
	args = append(args, "--cap-add=SYS_ADMIN")

	if !cfg.Passphrase && cfg.KeyFile != "" {
		// Mount the age key file as a read-only secret
		absKey, _ := filepath.Abs(cfg.KeyFile)
		args = append(args, "-v", absKey+":/run/secrets/age-key:ro")
	}
	return args
}

// Suppress unused import warning
var _ = io.EOF
