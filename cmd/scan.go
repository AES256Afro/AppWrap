package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
)

var (
	scanOutput   string
	scanFormat   string
	scanStrategy string
	scanEncrypt  bool
	scanFirewall string
	scanVPN      string
)

var scanCmd = &cobra.Command{
	Use:   "scan <exe-path|lnk-path>",
	Short: "Discover dependencies and generate a container profile",
	Long: `Scan a Windows executable to discover all its dependencies including
DLLs, registry keys, COM objects, and runtime requirements. Generates
a YAML profile that can be used with 'appwrap build'.`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "output profile path (default: <app-name>-profile.yaml)")
	scanCmd.Flags().StringVar(&scanFormat, "format", "yaml", "output format: yaml or json")
	scanCmd.Flags().StringVar(&scanStrategy, "strategy", "wine", "build strategy: wine, windows-servercore, windows-nanoserver")
	scanCmd.Flags().BoolVar(&scanEncrypt, "encrypt", false, "enable Age encryption in the profile")
	scanCmd.Flags().StringVar(&scanFirewall, "firewall", "", "firewall default policy: deny or allow (enables firewall)")
	scanCmd.Flags().StringVar(&scanVPN, "vpn", "", "path to WireGuard .conf file (enables VPN)")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	svc := service.New(service.WithConfigDir(findConfigDir()))

	events := make(chan service.Event, 256)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ev := range events {
			switch ev.Kind {
			case service.EventInfo:
				fmt.Println(ev.Message)
			case service.EventProgress:
				fmt.Printf("[%3d%%] %s\n", ev.Percent, ev.Message)
			case service.EventWarning:
				fmt.Printf("Warning: %s\n", ev.Message)
			case service.EventLogLine:
				fmt.Println(ev.Message)
			case service.EventError:
				fmt.Fprintf(os.Stderr, "Error: %s\n", ev.Message)
			case service.EventComplete:
				fmt.Println(ev.Message)
			}
		}
	}()

	result, err := svc.ScanApp(cmd.Context(), service.ScanOpts{
		TargetPath: args[0],
		Strategy:   scanStrategy,
		Format:     scanFormat,
		OutputPath: scanOutput,
		Encrypt:    scanEncrypt,
		Firewall:   scanFirewall,
		VPNConfig:  scanVPN,
		Verbose:    verbose,
	}, events)

	close(events)
	wg.Wait()

	if err != nil {
		return err
	}

	appName := strings.ToLower(result.Profile.App.Name)
	appName = strings.ReplaceAll(appName, " ", "-")
	fmt.Printf("\nNext: appwrap build %s --tag %s:latest\n", result.OutputPath, appName)

	return nil
}
