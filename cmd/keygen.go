package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
)

var keygenOutput string

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate an Age keypair for encrypting container contents",
	Long: `Generate an Age encryption keypair. The public key (recipient) is used
in the profile to encrypt files at build time. The private key (identity)
is needed at runtime to decrypt and start the container.

Example:
  appwrap keygen --output ./keys
  # Then add to your profile:
  #   security:
  #     encryption:
  #       enabled: true
  #       recipient: age1...  (from keygen output)
  #       keyFile: ./keys/appwrap-age-key.txt`,
	RunE: runKeygen,
}

func init() {
	keygenCmd.Flags().StringVarP(&keygenOutput, "output", "o", ".", "directory to save the keypair")
	rootCmd.AddCommand(keygenCmd)
}

func runKeygen(cmd *cobra.Command, args []string) error {
	svc := service.New(service.WithConfigDir(findConfigDir()))

	result, err := svc.GenerateKeys(cmd.Context(), service.KeygenOpts{
		OutputDir: keygenOutput,
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nAdd this to your profile under security.encryption:\n")
	fmt.Printf("  recipient: %s\n", result.Recipient)
	fmt.Printf("  keyFile: %s\n", result.KeyFile)

	return nil
}
