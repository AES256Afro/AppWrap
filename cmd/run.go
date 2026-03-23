package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
)

var (
	runDisplay    string
	runDetach     bool
	runRemove     bool
	runName       string
	runProfile    string
	runAgeKey     string
	runPassphrase string
)

var runCmd = &cobra.Command{
	Use:   "run <image>",
	Short: "Run a containerized application",
	Long: `Start a container from an AppWrap-built image. For GUI apps,
use --display to choose how the app is displayed.

For encrypted containers, provide the Age key:
  appwrap run myapp:latest --profile myapp-profile.yaml --age-key ./age-key.txt

For passphrase-encrypted containers:
  appwrap run myapp:latest --profile myapp-profile.yaml --passphrase "mysecret"`,
	Args: cobra.ExactArgs(1),
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringVar(&runDisplay, "display", "none", "display mode: none, vnc, novnc, rdp")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "run in background")
	runCmd.Flags().BoolVar(&runRemove, "rm", true, "remove container after exit")
	runCmd.Flags().StringVar(&runName, "name", "", "container name")
	runCmd.Flags().StringVar(&runProfile, "profile", "", "profile path (needed for security features)")
	runCmd.Flags().StringVar(&runAgeKey, "age-key", "", "path to Age identity file for encrypted containers")
	runCmd.Flags().StringVar(&runPassphrase, "passphrase", "", "passphrase for encrypted containers")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
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

	err := svc.RunContainer(cmd.Context(), service.RunOpts{
		Image:      args[0],
		Display:    runDisplay,
		Detach:     runDetach,
		Remove:     runRemove,
		Name:       runName,
		Profile:    runProfile,
		AgeKey:     runAgeKey,
		Passphrase: runPassphrase,
	}, events)

	close(events)
	wg.Wait()

	return err
}
