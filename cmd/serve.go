package cmd

import (
	"ubco-team15/omr/internal/httpserver"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the HTTP server for the service",
	Long: `Starts an HTTP server for the OMR service. The server will configure
itself based on the value of the environment variable "OMR_MODE", 
which should be set to one of

  - "OMR_MODE=TEST", which requires no external services and stores 
    nothing in persistent memory, instead mocking database and 
    network connections.

  - "OMR_MODE=DEVELOPMENT", which assumes that the development 
    services are running. Access to these services must be configured
    by environment variables.

  - "OMR_MODE=PRODUCTION", which assumes that the production services
    are running. Access to these services must be configured by
    environment variables.

Alternatively, you can use a flag to indicate the mode.
`,
	Run: func(cmd *cobra.Command, args []string) {
		httpserver.Start("localhost", port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
