package cmd

import (
	"fmt"
	"ubco-team15/omr/internal/scanner"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Scans an image and sends it to the grading service for processing.",
	Long: `Scans an image and sends it to the grading service for processing.
	The path argument specifies the path to the image to be scanned. 
	The --display flag can be used to display the scanned image in a window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		displayOutput, _ := cmd.Flags().GetBool("display")

		path, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		data, err := scanner.Scan(path)
		if err != nil {
			return err
		}

		defer data.Close()

		if displayOutput {
			display(data.Color, "Scanned Image")
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().BoolP("display", "d", false, "Display the scanned image in a window.")
	rootCmd.AddCommand(scanCmd)
}

// Provides a display window to see img output,
// although it does NOT work in docker containers :'(
func display(img gocv.Mat, title string) {
	window := gocv.NewWindow(title)
	defer window.Close()

	window.ResizeWindow(1000, 1414)
	window.IMShow(img)

	fmt.Printf("Press [Esc] to continue...\n")

	for {
		if window.WaitKey(0)&0xFF == 27 {
			break
		}
	}
}
