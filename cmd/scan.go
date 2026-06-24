package cmd

import (
	"fmt"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var scanCmd = &cobra.Command{
	Use:   "scan <img> <scan-template> <mark-template>",
	Short: "Preprocesses an image and marks it in one step.",
	Long: `Combines the preprocess and mark commands: applies perspective correction
and binarization using the scan template, then scores all bubbles using the mark
template. Use --output to save the binary preprocessed image, and --display to
show the colour-corrected image before marking.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		display, _ := cmd.Flags().GetBool("display")
		output, _ := cmd.Flags().GetString("output")

		imgPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve image path: %w", err)
		}
		scanTmplPath, err := utils.Resolve(args[1])
		if err != nil {
			return fmt.Errorf("resolve scan template path: %w", err)
		}
		markTmplPath, err := utils.Resolve(args[2])
		if err != nil {
			return fmt.Errorf("resolve mark template path: %w", err)
		}

		data, err := doPreprocess(imgPath, scanTmplPath, display, output)
		if err != nil {
			return err
		}
		defer data.Close()

		return doMark(data.Binary, markTmplPath)
	},
}

func init() {
	scanCmd.Flags().BoolP("display", "d", false, "Display the colour-corrected image in a window.")
	scanCmd.Flags().StringP("output", "o", "", "Write the binary preprocessed image to a file.")
	rootCmd.AddCommand(scanCmd)
}

// Display shows img in a window and waits for Esc. Does not work in Docker.
func Display(img gocv.Mat, title string) {
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
