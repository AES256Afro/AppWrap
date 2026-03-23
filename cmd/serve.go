package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theencryptedafro/appwrap/internal/service"
	"github.com/theencryptedafro/appwrap/internal/web"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Launch the web UI",
	Long:  "Start AppWrap's web interface. Opens a local HTTP server with a browser-based GUI.",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "host to bind to")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	svc := service.New(service.WithConfigDir(findConfigDir()))
	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	fmt.Printf("AppWrap Web UI: http://%s\n", addr)
	return web.Serve(svc, addr)
}
