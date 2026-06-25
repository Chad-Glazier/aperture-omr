package cmd

import (
	"fmt"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var scanCmd = &cobra.Command{
	Use:   "scan <scan-template> <mark-template> <img> [<img>...]",
	Short: "Preprocesses images and marks them in one step.",
	Long: `Combines the preprocess and mark commands: applies perspective correction
and binarization to each page image using the scan template, then scores all
bubbles using the mark template. Pass one image per page in order.
Use --output to save binary preprocessed images, and --display to show the
colour-corrected first page before marking.`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		display, _ := cmd.Flags().GetBool("display")
		output, _ := cmd.Flags().GetString("output")

		scanTmplPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve scan template path: %w", err)
		}
		markTmplPath, err := utils.Resolve(args[1])
		if err != nil {
			return fmt.Errorf("resolve mark template path: %w", err)
		}
		imgPaths := args[2:]

		pages, err := doPreprocess(scanTmplPath, imgPaths, output)
		if err != nil {
			return err
		}
		defer func() {
			for _, p := range pages {
				p.Close()
			}
		}()

		binaries := make([]gocv.Mat, len(pages))
		for i, p := range pages {
			binaries[i] = p.Binary
		}

		result := doMark(binaries, markTmplPath)

		if display {
			Display(pages[0].Color, "Marked Result")
		}

		return result
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
