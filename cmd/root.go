package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "omr",
	Short: "An Optical Mark Recognition (OMR) service exposed via HTTP",
	Long:  `An Optical Mark Recognition (OMR) service exposed via HTTP.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// The following flags are actually used in the config package, not here.
	// They are only included here so that Cobra's generated help messages
	// recognize them.
	rootCmd.PersistentFlags().Bool("test", false, "run in test mode")
	rootCmd.PersistentFlags().Bool("development", false, "run in development mode")
	rootCmd.PersistentFlags().Bool("production", false, "run in production mode")
}
