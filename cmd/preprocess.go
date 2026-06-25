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
	Use:   "preprocess <scan-template> <img> [<img>...]",
	Short: "Preprocesses one or more images using a scan template.",
	Long: `Applies perspective correction and binarization to each image using the
anchor points and config defined in the scan template. Pass one image per page
defined in the template. The --output flag prefix writes each preprocessed
binary image to <prefix>_<N>.png.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")

		tmplPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve template path: %w", err)
		}
		imgPaths := args[1:]

		results, err := doPreprocess(tmplPath, imgPaths, output)
		if err != nil {
			return err
		}
		for _, d := range results {
			d.Close()
		}
		return nil
	},
}

func init() {
	preprocessCmd.Flags().StringP("output", "o", "", "Prefix for preprocessed output images (e.g. -o out writes out_0.png, out_1.png, ...).")
	rootCmd.AddCommand(preprocessCmd)
}

// doPreprocess runs the scanning pipeline on each imgPath and optionally saves
// the binary outputs. The returned ScanData slice must be closed by the caller.
func doPreprocess(
	tmplPath string, imgPaths []string, outputPrefix string,
) ([]*scanner.ScanData, error) {
	tmpl, err := loadScanTemplate(tmplPath)
	if err != nil {
		return nil, err
	}
	defer tmpl.Close()

	readers := make([]io.Reader, len(imgPaths))
	closers := make([]io.Closer, len(imgPaths))
	for i, p := range imgPaths {
		resolved, err := utils.Resolve(p)
		if err != nil {
			for j := 0; j < i; j++ {
				closers[j].Close()
			}
			return nil, fmt.Errorf("resolve image path: %w", err)
		}
		f, err := os.Open(resolved)
		if err != nil {
			for j := 0; j < i; j++ {
				closers[j].Close()
			}
			return nil, fmt.Errorf("open image: %w", err)
		}
		readers[i] = f
		closers[i] = f
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	results, err := scanner.Scan(readers, tmpl)
	if err != nil {
		return nil, fmt.Errorf("preprocess: %w", err)
	}

	if outputPrefix != "" {
		for i, data := range results {
			outPath := fmt.Sprintf("%s_%d.png", outputPrefix, i)
			resolved, err := utils.Resolve(outPath)
			if err != nil {
				for _, d := range results {
					d.Close()
				}
				return nil, fmt.Errorf("resolve output path: %w", err)
			}
			if ok := gocv.IMWrite(resolved, data.Binary); !ok {
				for _, d := range results {
					d.Close()
				}
				return nil, fmt.Errorf("failed to write output image to %q", resolved)
			}
			fmt.Printf("wrote preprocessed image to: %q\n", resolved)
		}
	}

	return results, nil
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
