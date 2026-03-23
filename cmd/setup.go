package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

type depCheck struct {
	name     string
	binary   string
	args     []string
	required bool
	install  string
}

var setupDeps = []depCheck{
	{name: "WSL2", binary: "wsl", args: []string{"--status"}, required: true, install: "wsl --install --no-distribution"},
	{name: "Docker Desktop", binary: "docker", args: []string{"info"}, required: true, install: "winget install -e --id Docker.DockerDesktop"},
	{name: "Age Encryption", binary: "age", args: []string{"--version"}, required: false, install: "winget install -e --id FiloSottile.age"},
	{name: "WireGuard", binary: "wg", args: []string{"--version"}, required: false, install: "winget install -e --id WireGuard.WireGuard"},
}

var setupInstall bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Check and install all dependencies",
	Long: `Verify that all required and optional dependencies are installed.
Use --install to automatically install missing dependencies via winget.

Required: WSL2, Docker Desktop
Optional: Age (encryption), WireGuard (VPN)`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().BoolVar(&setupInstall, "install", false, "automatically install missing dependencies")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("AppWrap Environment Check")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	// Check winget availability
	_, wingetErr := exec.LookPath("winget")
	hasWinget := wingetErr == nil

	allGood := true
	var missing []depCheck

	for _, dep := range setupDeps {
		reqTag := "optional"
		if dep.required {
			reqTag = "required"
		}

		// Check if binary exists
		_, err := exec.LookPath(dep.binary)
		if err != nil {
			fmt.Printf("  ✗ %-20s [%s]  NOT FOUND\n", dep.name, reqTag)
			if dep.required {
				allGood = false
			}
			missing = append(missing, dep)
			continue
		}

		// Try running check command
		if len(dep.args) > 0 {
			checkCmd := exec.Command(dep.binary, dep.args...)
			out, err := checkCmd.CombinedOutput()
			if err != nil {
				if dep.binary == "docker" {
					fmt.Printf("  ✗ %-20s [%s]  INSTALLED BUT NOT RUNNING\n", dep.name, reqTag)
					fmt.Printf("    → Start Docker Desktop and try again\n")
				} else {
					fmt.Printf("  ? %-20s [%s]  FOUND (check failed)\n", dep.name, reqTag)
				}
				if dep.required {
					allGood = false
				}
				missing = append(missing, dep)
				continue
			}
			// Show version (first line)
			version := strings.TrimSpace(strings.Split(string(out), "\n")[0])
			if len(version) > 60 {
				version = version[:60] + "..."
			}
			fmt.Printf("  ✓ %-20s [%s]  %s\n", dep.name, reqTag, version)
		} else {
			fmt.Printf("  ✓ %-20s [%s]  found\n", dep.name, reqTag)
		}
	}

	fmt.Println()

	if allGood && len(missing) == 0 {
		fmt.Println("All dependencies are installed. AppWrap is ready!")
		return nil
	}

	if allGood && len(missing) > 0 {
		fmt.Println("Required dependencies OK. Some optional tools are missing.")
	} else {
		fmt.Println("Some required dependencies are missing.")
	}

	if len(missing) > 0 && !setupInstall {
		fmt.Println()
		fmt.Println("To install missing dependencies automatically:")
		fmt.Println("  appwrap setup --install")
		fmt.Println()
		fmt.Println("Or install manually:")
		for _, dep := range missing {
			fmt.Printf("  %s\n", dep.install)
		}
		return nil
	}

	if setupInstall && len(missing) > 0 {
		if !hasWinget {
			fmt.Println()
			fmt.Println("winget is not available. Install dependencies manually:")
			for _, dep := range missing {
				fmt.Printf("  %s\n", dep.install)
			}
			return fmt.Errorf("winget not found")
		}

		fmt.Println()
		for _, dep := range missing {
			fmt.Printf("Installing %s...\n", dep.name)

			parts := strings.Fields(dep.install)
			installCmd := exec.Command(parts[0], parts[1:]...)
			installCmd.Stdout = cmd.OutOrStdout()
			installCmd.Stderr = cmd.ErrOrStderr()

			if err := installCmd.Run(); err != nil {
				fmt.Printf("  ✗ Failed to install %s: %v\n", dep.name, err)
				if dep.required {
					return fmt.Errorf("failed to install required dependency: %s", dep.name)
				}
			} else {
				fmt.Printf("  ✓ %s installed\n", dep.name)
			}
			fmt.Println()
		}

		fmt.Println("Done! You may need to restart your terminal or start Docker Desktop.")
	}

	return nil
}
