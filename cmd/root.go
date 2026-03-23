package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "appwrap",
	Short: "Containerize Windows applications into Docker containers",
	Long: `AppWrap discovers all dependencies of a Windows application,
generates a container profile, builds a Docker image, and runs it
with optional display forwarding (VNC, noVNC, RDP).

Workflow:
  1. appwrap scan <exe>       — Discover dependencies and generate a profile
  2. appwrap build <profile>  — Build a Docker image from the profile
  3. appwrap run <image>      — Run the containerized application`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.appwrap/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not find home directory:", err)
			return
		}
		viper.AddConfigPath(home + "/.appwrap")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	viper.ReadInConfig() // Silently ignore if no config file found
}
