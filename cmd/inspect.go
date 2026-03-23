package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <exe-path>",
	Short: "Quick PE analysis — print architecture, subsystem, and imports",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	svc := service.New(service.WithConfigDir(findConfigDir()))

	result, err := svc.InspectBinary(cmd.Context(), service.InspectOpts{
		TargetPath: args[0],
	})
	if err != nil {
		return err
	}

	fmt.Printf("File:      %s\n", result.FileName)
	fmt.Printf("Path:      %s\n", result.FullPath)
	fmt.Printf("Arch:      %s\n", result.Arch)
	fmt.Printf("Subsystem: %s\n", result.Subsystem)
	fmt.Printf("Imports:   %d DLLs\n\n", len(result.Imports))

	for _, imp := range result.Imports {
		marker := " "
		if imp.IsSystem {
			marker = "S"
		}
		fmt.Printf("  [%s] %s\n", marker, imp.Name)
	}

	fmt.Printf("\n[S] = system DLL\n")

	return nil
}
