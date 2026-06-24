package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"ubco-team15/omr/internal/marker"
	"ubco-team15/omr/internal/utils"

	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var markCmd = &cobra.Command{
	Use:   "mark <img> <mark-template>",
	Short: "Marks a preprocessed binary image using a mark template.",
	Long: `Reads bubble fill ratios from a preprocessed binary image and scores
each question defined in the mark template. The image must already have been
corrected and binarized (e.g. by the preprocess command).`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		imgPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve image path: %w", err)
		}
		tmplPath, err := utils.Resolve(args[1])
		if err != nil {
			return fmt.Errorf("resolve template path: %w", err)
		}

		img := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
		defer img.Close()
		if img.Empty() {
			return fmt.Errorf("could not read image: %s", imgPath)
		}

		return doMark(img, tmplPath)
	},
}

func init() {
	rootCmd.AddCommand(markCmd)
}

// doMark evaluates img against the mark template at tmplPath
// and prints the report.
func doMark(img gocv.Mat, tmplPath string) error {
	tmpl, err := loadMarkTemplate(tmplPath)
	if err != nil {
		return err
	}
	result, err := marker.Evaluate(img, tmpl)
	if err != nil {
		return fmt.Errorf("mark: %w", err)
	}
	printResults(result)
	return nil
}

// loadMarkTemplate is shared by mark and scan commands.
func loadMarkTemplate(tmplPath string) (*marker.Template, error) {
	f, err := os.Open(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("open mark template: %w", err)
	}
	defer f.Close()

	tmpl, err := marker.LoadTemplate(f)
	if err != nil {
		return nil, fmt.Errorf("load mark template: %w", err)
	}
	return tmpl, nil
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
