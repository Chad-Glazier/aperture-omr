package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"ubco-team15/omr/config"
)

var healthcheckCmd = &cobra.Command{
	Use:    "healthcheck",
	Short:  "Checks whether the OMR HTTP service is healthy",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := http.Client{Timeout: 2 * time.Second}
		response, err := client.Get("http://127.0.0.1:" + config.PORT + "/health")
		if err != nil {
			return err
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("health endpoint returned %s", response.Status)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
}
