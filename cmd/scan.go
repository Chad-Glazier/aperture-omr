package cmd

import (
	"ubco-team15/omr/internal/scanner"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Scans an image and sends it to the grading service for processing.",
	Long: `Scans an image and sends it to the grading service for processing.
	The path argument specifies the path to the image to be scanned. 
	The --display flag can be used to display the scanned image in a window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		display, _ := cmd.Flags().GetBool("display")
		img, err := scanner.Scan(args[0])
		if err != nil {
			return err
		}

		defer img.Close()

		if display {
			utils.Display(img, "Scanned Image")
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().BoolP("display", "d", false, "Display the scanned image in a window.")
	rootCmd.AddCommand(scanCmd)
}
