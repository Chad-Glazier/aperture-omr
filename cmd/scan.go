package cmd

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"ubco-team15/omr/internal/marker"
	"ubco-team15/omr/internal/scanner"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var scanCmd = &cobra.Command{
	Use:   "scan <img> <template>",
	Short: "Scans an image and sends it to the grading service for processing.",
	Long: `Scans an image and sends it to the grading service for processing.
	The img argument specifies the path to the image to be scanned.
	The template argument specifies the path to the JSON template to use.
	The --display flag can be used to display the scanned image in a window.
	The --output flag can be used to write the scanned image to a file.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		display, _ := cmd.Flags().GetBool("display")
		output, _ := cmd.Flags().GetString("output")

		path, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		tmpl, err := utils.Resolve(args[1])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		f, err := os.Open(tmpl)
		if err != nil {
			return fmt.Errorf("open template: %w", err)
		}
		defer f.Close()

		tmplDir := filepath.Dir(tmpl)
		template, err := scanner.LoadTemplate(f, func(anchorPath string) (io.Reader, error) {
			if !filepath.IsAbs(anchorPath) && !strings.HasPrefix(anchorPath, "~") {
				anchorPath = filepath.Join(tmplDir, anchorPath)
			}
			anchorPath, err := utils.Resolve(anchorPath)
			if err != nil {
				return nil, err
			}
			return os.Open(anchorPath)
		})
		if err != nil {
			return fmt.Errorf("load template: %w", err)
		}
		defer template.Close()

		imgFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open image: %w", err)
		}
		defer imgFile.Close()

		data, err := scanner.Scan(imgFile, template)
		if err != nil {
			return fmt.Errorf("scanner: %w", err)
		}
		defer data.Close()

		if output != "" {
			outPath, err := utils.Resolve(output)
			if err != nil {
				return fmt.Errorf("resolve output path: %w", err)
			}
			if ok := gocv.IMWrite(outPath, data.Color); !ok {
				return fmt.Errorf("failed to write output image to %q", outPath)
			}
			fmt.Printf("successfully wrote output to: %q\n", outPath)
		}

		if len(template.Questions) == 0 {
			return nil
		}

		threshold := template.Config.FillThreshold
		if threshold == 0 {
			threshold = 0.5
		}
		inset := template.Config.BubbleInset
		if inset == 0 {
			inset = 0.75
		}
		result, err := marker.Evaluate(data.Binary, template.Questions, threshold, inset)
		if err != nil {
			return fmt.Errorf("extract: %w", err)
		}

		printResults(result)

		if display {
			preview := gocv.NewMat()
			data.Color.CopyTo(&preview)

			red := color.RGBA{R: 255}
			for _, q := range template.Questions {
				r := q.BubbleWidth / 2
				if q.BubbleHeight < q.BubbleWidth {
					r = q.BubbleHeight / 2
				}
				for _, b := range q.Options {
					center := image.Pt(b.X+q.BubbleWidth/2, b.Y+q.BubbleHeight/2)
					gocv.Circle(&preview, center, r, red, 2)
				}
			}

			Display(preview, "Scanned Image")
			preview.Close()
		}

		return nil
	},
}

func init() {
	scanCmd.Flags().BoolP(
		"display", "d", false, "Display the scanned image in a window.")
	scanCmd.Flags().StringP(
		"output", "o", "", "Write the scanned image to a file at the given path.")
	rootCmd.AddCommand(scanCmd)
}

func printResults(r *marker.Result) {
	fmt.Println("\n======================================================")
	fmt.Println("             OMR BUBBLE EXTRACTION REPORT             ")
	fmt.Println("======================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	fmt.Fprintln(w, "QUESTION\tSELECTED\tCONFIDENCE\tMANUAL REVIEW FLAG")
	fmt.Fprintln(w, "--------\t--------\t----------\t------------------")

	for _, ans := range r.Answers {
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%t\n",
			ans.QuestionID,
			strings.Join(ans.Selected, ", "),
			ans.Confidence,
			ans.Flag,
		)
	}

	w.Flush()
	fmt.Println("======================================================")
}

// Provides a display window to see img output,
// although it does NOT work in docker containers :'(
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
