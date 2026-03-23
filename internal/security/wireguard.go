package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// PrepareWireGuard generates the WireGuard config and startup script for the container.
func PrepareWireGuard(contextDir string, cfg profile.VPNConfig) error {
	// Generate wg0.conf either from a file or inline config
	wgConf, err := buildWGConfig(cfg)
	if err != nil {
		return fmt.Errorf("build wireguard config: %w", err)
	}

	// Write wg0.conf into the build context
	wgDir := filepath.Join(contextDir, "wireguard")
	if err := os.MkdirAll(wgDir, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(wgDir, "wg0.conf"), []byte(wgConf), 0600); err != nil {
		return err
	}

	// Generate the VPN startup script
	return writeVPNScript(contextDir, cfg)
}

func buildWGConfig(cfg profile.VPNConfig) (string, error) {
	// If a config file is provided, read it
	if cfg.ConfigFile != "" {
		data, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			return "", fmt.Errorf("read wireguard config: %w", err)
		}
		return string(data), nil
	}

	// Build from inline config
	if cfg.Interface == nil || cfg.Peer == nil {
		return "", fmt.Errorf("VPN enabled but no configFile or inline interface/peer provided")
	}

	var conf strings.Builder
	conf.WriteString("[Interface]\n")
	if cfg.Interface.PrivateKey != "" {
		conf.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.Interface.PrivateKey))
	}
	if cfg.Interface.Address != "" {
		conf.WriteString(fmt.Sprintf("Address = %s\n", cfg.Interface.Address))
	}
	if cfg.Interface.DNS != "" {
		conf.WriteString(fmt.Sprintf("DNS = %s\n", cfg.Interface.DNS))
	}
	if cfg.Interface.ListenPort > 0 {
		conf.WriteString(fmt.Sprintf("ListenPort = %d\n", cfg.Interface.ListenPort))
	}

	conf.WriteString("\n[Peer]\n")
	if cfg.Peer.PublicKey != "" {
		conf.WriteString(fmt.Sprintf("PublicKey = %s\n", cfg.Peer.PublicKey))
	}
	if cfg.Peer.Endpoint != "" {
		conf.WriteString(fmt.Sprintf("Endpoint = %s\n", cfg.Peer.Endpoint))
	}
	if cfg.Peer.AllowedIPs != "" {
		conf.WriteString(fmt.Sprintf("AllowedIPs = %s\n", cfg.Peer.AllowedIPs))
	}
	if cfg.Peer.PersistentKeepalive > 0 {
		conf.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", cfg.Peer.PersistentKeepalive))
	}
	if cfg.Peer.PresharedKey != "" {
		conf.WriteString(fmt.Sprintf("PresharedKey = %s\n", cfg.Peer.PresharedKey))
	}

	return conf.String(), nil
}

func writeVPNScript(contextDir string, cfg profile.VPNConfig) error {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("echo '[appwrap] Starting WireGuard VPN...'\n\n")

	// Create TUN device if not present
	script.WriteString("# Ensure TUN device exists\n")
	script.WriteString("mkdir -p /dev/net\n")
	script.WriteString("if [ ! -c /dev/net/tun ]; then\n")
	script.WriteString("    mknod /dev/net/tun c 10 200\n")
	script.WriteString("    chmod 600 /dev/net/tun\n")
	script.WriteString("fi\n\n")

	// Copy WireGuard config to expected location
	script.WriteString("# Set up WireGuard config\n")
	script.WriteString("cp /home/appuser/wireguard/wg0.conf /etc/wireguard/wg0.conf\n")
	script.WriteString("chmod 600 /etc/wireguard/wg0.conf\n\n")

	// Bring up the interface
	script.WriteString("# Bring up WireGuard interface\n")
	script.WriteString("wg-quick up wg0\n\n")

	// Verify connection
	script.WriteString("echo '[appwrap] WireGuard interface:'\n")
	script.WriteString("wg show wg0\n")
	script.WriteString("echo ''\n")
	script.WriteString("echo '[appwrap] VPN IP:'\n")
	script.WriteString("ip addr show wg0 | grep inet\n\n")

	// Kill switch: if VPN drops, block all traffic
	if cfg.KillSwitch {
		script.WriteString("# Kill switch — block all traffic except WireGuard\n")
		script.WriteString("WG_ENDPOINT=$(grep 'Endpoint' /etc/wireguard/wg0.conf | awk '{print $3}' | cut -d: -f1)\n")
		script.WriteString("WG_PORT=$(grep 'Endpoint' /etc/wireguard/wg0.conf | awk '{print $3}' | cut -d: -f2)\n")
		script.WriteString("\n")
		script.WriteString("# Save current default gateway for WireGuard endpoint\n")
		script.WriteString("DEFAULT_GW=$(ip route | grep default | awk '{print $3}')\n")
		script.WriteString("\n")
		script.WriteString("iptables -F OUTPUT\n")
		script.WriteString("iptables -P OUTPUT DROP\n")
		script.WriteString("# Allow loopback\n")
		script.WriteString("iptables -A OUTPUT -o lo -j ACCEPT\n")
		script.WriteString("# Allow traffic through WireGuard interface\n")
		script.WriteString("iptables -A OUTPUT -o wg0 -j ACCEPT\n")
		script.WriteString("# Allow WireGuard handshake to endpoint\n")
		script.WriteString("iptables -A OUTPUT -d $WG_ENDPOINT -p udp --dport $WG_PORT -j ACCEPT\n")
		script.WriteString("# Allow established connections\n")
		script.WriteString("iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT\n\n")
		script.WriteString("echo '[appwrap] Kill switch active — traffic only flows through VPN.'\n\n")
	}

	script.WriteString("echo '[appwrap] VPN ready.'\n")

	scriptPath := filepath.Join(contextDir, "vpn-setup.sh")
	return os.WriteFile(scriptPath, []byte(script.String()), 0755)
}

// WireGuardDockerfileSnippet returns Dockerfile lines for WireGuard support.
func WireGuardDockerfileSnippet() string {
	return `
# --- WireGuard VPN Support ---
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
        wireguard-tools \
        iproute2 \
        iptables \
        && \
    apt-get clean && rm -rf /var/lib/apt/lists/* && \
    mkdir -p /etc/wireguard
COPY wireguard/ /home/appuser/wireguard/
COPY vpn-setup.sh /usr/local/bin/vpn-setup.sh
RUN chmod +x /usr/local/bin/vpn-setup.sh && \
    chmod 600 /home/appuser/wireguard/wg0.conf
USER appuser
`
}

// WireGuardDockerRunArgs returns extra docker run arguments for WireGuard.
func WireGuardDockerRunArgs() []string {
	return []string{
		"--cap-add=NET_ADMIN",
		"--device=/dev/net/tun",
		"--sysctl=net.ipv4.conf.all.src_valid_mark=1",
	}
}
