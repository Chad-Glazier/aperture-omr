package cmd

import (
	"ubco-team15/omr/internal/scanner"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan path",
	Short: "Scans an image and sends it to the grading service for processing.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scanner.Scan(args[0])
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
