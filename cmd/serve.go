package cmd

import (
	"fmt"
	"net"
	"strconv"

	"github.com/Chad-Glazier/aperture-omr/internal/server"

	"github.com/spf13/cobra"
)

var (
	portNum int
	tls     bool
	apiKey  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the OMR's HTTP server",
	Long: `Starts the OMR's HTTP server.

To set a global API key, use the '--key' flag. All requests will
be checked for an 'OMR-API-Key' header with a matching value. Any 
requests that are missing the key will be rejected. Alternatively,
you can define a global key by setting the 'OMR_GLOBAL_KEY' 
variable in the environment.

To use TLS, you'll need certificates. You can supply your own by
setting them in '` + certDir + `' or by running the 'certify' sub-
command. In either case, the OMR will look for
  - the public key in '` + certDir + `cert.pem', and
  - the private key in '` + certDir + `key.pem'.

Generated certificates will be secure for practical purposes, 
but since they're not made by a recognized Certificate Authority
tools like browsers may give you warnings.
	`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if portNum < 1 || portNum > 65535 {
			return fmt.Errorf(
				"invalid port %d: must be between 1 and 65535",
				portNum,
			)
		}

		testCnx, err := net.Listen("tcp", ":"+strconv.Itoa(portNum))
		if err != nil {
			return fmt.Errorf(
				"port %d is already in use.",
				portNum,
			)
		}
		testCnx.Close()

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			hostname = "localhost"
			port     = strconv.Itoa(portNum)
		)

		if tls {
			server.StartTls(hostname, port, apiKey, certDir)
		} else {
			server.Start(hostname, port, apiKey)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(
		&portNum,
		"port",
		"p",
		3000,
		"port to listen on",
	)
	serveCmd.Flags().BoolVar(
		&tls,
		"tls",
		false,
		"set this if you want to use TLS",
	)
	serveCmd.Flags().StringVar(
		&apiKey,
		"key",
		"",
		"set a global API key",
	)
}
