package security

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// GenerateSecurityEntrypoint creates a master entrypoint script that chains
// all security features in the correct order:
//   1. WireGuard VPN (must be up before firewall rules apply)
//   2. Firewall (iptables rules)
//   3. Decrypt (Age decryption to tmpfs)
//   4. Launch the application
func GenerateSecurityEntrypoint(contextDir string, cfg profile.SecurityConfig) error {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# AppWrap Security Entrypoint\n")
	script.WriteString("# Chains: VPN → Firewall → Decrypt → App\n\n")

	// Step 1: VPN
	if cfg.VPN.Enabled {
		script.WriteString("# --- Step 1: Start VPN ---\n")
		script.WriteString("source /usr/local/bin/vpn-setup.sh\n")
		script.WriteString("echo ''\n\n")
	}

	// Step 2: Firewall
	if cfg.Firewall.Enabled {
		script.WriteString("# --- Step 2: Configure Firewall ---\n")
		script.WriteString("source /usr/local/bin/firewall-setup.sh\n")
		script.WriteString("echo ''\n\n")
	}

	// Step 3: Decrypt
	if cfg.Encryption.Enabled {
		script.WriteString("# --- Step 3: Decrypt Application ---\n")
		script.WriteString("ENCRYPTED_DIR=\"/home/appuser/app\"\n")
		script.WriteString("DECRYPT_DIR=\"/run/appwrap\"\n\n")
		script.WriteString("mkdir -p \"$DECRYPT_DIR\"\n")
		script.WriteString("mount -t tmpfs -o size=512m,mode=0700 tmpfs \"$DECRYPT_DIR\"\n\n")
		script.WriteString("echo '[appwrap] Decrypting application files to tmpfs...'\n")
		script.WriteString("find \"$ENCRYPTED_DIR\" -name '*.age' | while read -r enc_file; do\n")
		script.WriteString("    rel_path=\"${enc_file#$ENCRYPTED_DIR/}\"\n")
		script.WriteString("    out_path=\"$DECRYPT_DIR/${rel_path%.age}\"\n")
		script.WriteString("    mkdir -p \"$(dirname \"$out_path\")\"\n")

		if cfg.Encryption.Passphrase {
			script.WriteString("    echo \"$AGE_PASSPHRASE\" | age -d -o \"$out_path\" \"$enc_file\"\n")
		} else {
			script.WriteString("    age -d -i /run/secrets/age-key -o \"$out_path\" \"$enc_file\"\n")
		}

		script.WriteString("done\n")
		script.WriteString("chmod -R +x \"$DECRYPT_DIR\" 2>/dev/null || true\n")
		script.WriteString("echo '[appwrap] Decryption complete.'\n")
		script.WriteString("echo ''\n\n")

		// App path is now in the decrypted tmpfs
		script.WriteString("APP_DIR=\"$DECRYPT_DIR\"\n")
	} else {
		script.WriteString("APP_DIR=\"/home/appuser/app\"\n")
	}

	// Step 4: Launch the application
	script.WriteString("# --- Step 4: Launch Application ---\n")
	script.WriteString("echo '[appwrap] Starting application...'\n")
	script.WriteString("exec \"$@\"\n")

	scriptPath := filepath.Join(contextDir, "security-entrypoint.sh")
	return os.WriteFile(scriptPath, []byte(script.String()), 0755)
}

// HasSecurityFeatures returns true if any security feature is enabled.
func HasSecurityFeatures(cfg profile.SecurityConfig) bool {
	return cfg.Encryption.Enabled || cfg.Firewall.Enabled || cfg.VPN.Enabled
}

// CollectDockerRunArgs gathers all extra docker run args needed for enabled security features.
func CollectDockerRunArgs(cfg profile.SecurityConfig) []string {
	seen := make(map[string]bool)
	var args []string

	addUnique := func(newArgs []string) {
		for _, a := range newArgs {
			if !seen[a] {
				seen[a] = true
				args = append(args, a)
			}
		}
	}

	if cfg.Encryption.Enabled {
		addUnique(DockerRunArgs(cfg.Encryption))
	}
	if cfg.Firewall.Enabled {
		addUnique(FirewallDockerRunArgs())
	}
	if cfg.VPN.Enabled {
		addUnique(WireGuardDockerRunArgs())
	}

	return args
}

// SecurityDockerfileSnippet returns the combined Dockerfile snippet for all security features.
func SecurityDockerfileSnippet(cfg profile.SecurityConfig) string {
	var snippet strings.Builder

	if cfg.VPN.Enabled {
		snippet.WriteString(WireGuardDockerfileSnippet())
	}
	if cfg.Firewall.Enabled {
		snippet.WriteString(FirewallDockerfileSnippet())
	}
	if cfg.Encryption.Enabled {
		snippet.WriteString(DockerfileSnippet())
	}

	// Master security entrypoint
	snippet.WriteString(`
# --- Security Entrypoint ---
USER root
COPY security-entrypoint.sh /usr/local/bin/security-entrypoint.sh
RUN chmod +x /usr/local/bin/security-entrypoint.sh
USER appuser
`)

	return snippet.String()
}
