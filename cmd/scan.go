package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"ubco-team15/omr/internal/scanner"

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

		path, err := resolve(args[0])
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

// Expands a relative or home directory path into an
// OS-specific absolute file path.
func resolve(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()

		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %v", err)
		}

		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to resolve absolute path: %v", err)
	}

	return filepath.Clean(abs), nil
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
