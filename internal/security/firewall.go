package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theencryptedafro/appwrap/internal/profile"
)

// GenerateFirewallScript creates an iptables setup script from the firewall config.
// This script runs at container startup before the application.
func GenerateFirewallScript(contextDir string, cfg profile.FirewallConfig) error {
	var rules strings.Builder

	rules.WriteString("#!/bin/bash\n")
	rules.WriteString("set -e\n\n")
	rules.WriteString("echo '[appwrap] Configuring firewall...'\n\n")

	// Flush existing rules
	rules.WriteString("# Flush existing rules\n")
	rules.WriteString("iptables -F OUTPUT\n")
	rules.WriteString("iptables -F INPUT\n\n")

	// Set default policy
	defaultPolicy := strings.ToUpper(cfg.DefaultPolicy)
	if defaultPolicy == "" || defaultPolicy == "DENY" {
		defaultPolicy = "DROP"
	} else {
		defaultPolicy = "ACCEPT"
	}

	rules.WriteString(fmt.Sprintf("# Default policy: %s\n", defaultPolicy))
	rules.WriteString(fmt.Sprintf("iptables -P OUTPUT %s\n", defaultPolicy))
	rules.WriteString(fmt.Sprintf("iptables -P INPUT %s\n\n", defaultPolicy))

	// Always allow established/related connections (so responses come back)
	rules.WriteString("# Allow established connections\n")
	rules.WriteString("iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT\n")
	rules.WriteString("iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT\n\n")

	// Loopback
	if cfg.AllowLoopback {
		rules.WriteString("# Allow loopback (localhost)\n")
		rules.WriteString("iptables -A INPUT -i lo -j ACCEPT\n")
		rules.WriteString("iptables -A OUTPUT -o lo -j ACCEPT\n\n")
	}

	// DNS
	if cfg.AllowDNS {
		rules.WriteString("# Allow DNS (UDP/TCP port 53)\n")
		rules.WriteString("iptables -A OUTPUT -p udp --dport 53 -j ACCEPT\n")
		rules.WriteString("iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT\n\n")
	}

	// Allow rules
	for _, rule := range cfg.AllowRules {
		comment := ""
		if rule.Comment != "" {
			comment = fmt.Sprintf(" # %s", rule.Comment)
		}
		for _, r := range buildIPTablesArgs(rule, "ACCEPT") {
			rules.WriteString(fmt.Sprintf("iptables %s%s\n", r, comment))
		}
	}
	if len(cfg.AllowRules) > 0 {
		rules.WriteString("\n")
	}

	// Deny rules (explicit blocks, useful when default is ACCEPT)
	for _, rule := range cfg.DenyRules {
		comment := ""
		if rule.Comment != "" {
			comment = fmt.Sprintf(" # %s", rule.Comment)
		}
		for _, r := range buildIPTablesArgs(rule, "DROP") {
			rules.WriteString(fmt.Sprintf("iptables %s%s\n", r, comment))
		}
	}
	if len(cfg.DenyRules) > 0 {
		rules.WriteString("\n")
	}

	// Log dropped packets (useful for debugging)
	if defaultPolicy == "DROP" {
		rules.WriteString("# Log dropped outbound packets\n")
		rules.WriteString("iptables -A OUTPUT -j LOG --log-prefix '[appwrap-blocked] ' --log-level 4\n\n")
	}

	rules.WriteString("echo '[appwrap] Firewall configured.'\n")
	rules.WriteString("iptables -L -n --line-numbers\n")

	scriptPath := filepath.Join(contextDir, "firewall-setup.sh")
	return os.WriteFile(scriptPath, []byte(rules.String()), 0755)
}

func buildIPTablesArgs(rule profile.FirewallRule, target string) []string {
	var results []string

	protocols := []string{"tcp", "udp"}
	if rule.Protocol != "" && rule.Protocol != "both" {
		protocols = []string{rule.Protocol}
	}

	for _, proto := range protocols {
		args := fmt.Sprintf("-A OUTPUT -p %s", proto)

		if rule.IP != "" {
			args += fmt.Sprintf(" -d %s", rule.IP)
		}

		if rule.Port > 0 {
			args += fmt.Sprintf(" --dport %d", rule.Port)
		} else if rule.PortRange != "" {
			args += fmt.Sprintf(" --dport %s", rule.PortRange)
		}

		args += fmt.Sprintf(" -j %s", target)
		results = append(results, args)
	}

	return results
}

// FirewallDockerfileSnippet returns Dockerfile lines for iptables support.
func FirewallDockerfileSnippet() string {
	return `
# --- Firewall (iptables) Support ---
USER root
RUN apt-get update && apt-get install -y --no-install-recommends iptables && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
COPY firewall-setup.sh /usr/local/bin/firewall-setup.sh
RUN chmod +x /usr/local/bin/firewall-setup.sh
USER appuser
`
}

// FirewallDockerRunArgs returns extra docker run arguments for firewall support.
func FirewallDockerRunArgs() []string {
	return []string{"--cap-add=NET_ADMIN"}
}
