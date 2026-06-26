package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"ubco-team15/omr/internal/scanner"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var preprocessCmd = &cobra.Command{
	Use:   "preprocess <img> <scan-template>",
	Short: "Preprocesses an image using a scan template.",
	Long: `Applies perspective correction and binarization to an image using the
anchor points and config defined in the scan template. The --output flag writes
the preprocessed binary image to a file, which can then be passed to the mark
command.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")

		imgPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve image path: %w", err)
		}
		tmplPath, err := utils.Resolve(args[1])
		if err != nil {
			return fmt.Errorf("resolve template path: %w", err)
		}

		data, err := doPreprocess(imgPath, tmplPath, output)
		if err != nil {
			return err
		}
		data.Close()
		return nil
	},
}

func init() {
	preprocessCmd.Flags().StringP("output", "o", "", "Write the binary preprocessed image to a file.")
	rootCmd.AddCommand(preprocessCmd)
}

// doPreprocess runs the scanning pipeline on imgPath and optionally saves the
// binary output. The returned ScanData must be closed by the caller.
func doPreprocess(
	imgPath, tmplPath, output string,
) (*scanner.ScanData, error) {
	tmpl, err := loadScanTemplate(tmplPath)
	if err != nil {
		return nil, err
	}
	defer tmpl.Close()

	imgFile, err := os.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer imgFile.Close()

	data, err := scanner.Scan(imgFile, tmpl)
	if err != nil {
		return nil, fmt.Errorf("preprocess: %w", err)
	}

	if output != "" {
		outPath, err := utils.Resolve(output)
		if err != nil {
			data.Close()
			return nil, fmt.Errorf("resolve output path: %w", err)
		}
		if ok := gocv.IMWrite(outPath, data.Binary); !ok {
			data.Close()
			return nil, fmt.Errorf("failed to write output image to %q", outPath)
		}
		fmt.Printf("wrote preprocessed image to: %q\n", outPath)
	}

	return data, nil
}

// loadScanTemplate is shared by preprocess and scan commands.
func loadScanTemplate(tmplPath string) (*scanner.Template, error) {
	f, err := os.Open(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("open scan template: %w", err)
	}
	defer f.Close()

	tmplDir := filepath.Dir(tmplPath)
	tmpl, err := scanner.LoadTemplate(f,
		func(anchorPath string) (io.ReadCloser, error) {
			if !filepath.IsAbs(anchorPath) &&
				!strings.HasPrefix(anchorPath, "~") {
				anchorPath = filepath.Join(tmplDir, anchorPath)
			}
			anchorPath, err := utils.Resolve(anchorPath)
			if err != nil {
				return nil, err
			}
			return os.Open(anchorPath)
		})
	if err != nil {
		return nil, fmt.Errorf("load scan template: %w", err)
	}
	return tmpl, nil
}
