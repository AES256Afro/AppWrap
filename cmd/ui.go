package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
	"github.com/theencryptedafro/appwrap/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive terminal UI",
	Long:  "Start AppWrap's interactive terminal interface (TUI) for scanning, building, and managing containers.",
	RunE:  runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runUI(cmd *cobra.Command, args []string) error {
	svc := service.New(service.WithConfigDir(findConfigDir()))
	return tui.Run(svc)
}
