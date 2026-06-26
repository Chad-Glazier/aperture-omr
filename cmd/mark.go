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
	Use:   "mark <mark-template> <img> [<img>...]",
	Short: "Marks preprocessed binary images using a mark template.",
	Long: `Reads bubble fill ratios from one preprocessed binary image per page and
scores all questions defined in the mark template. Images must already have been
corrected and binarized (e.g. by the preprocess command), one per page in order.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tmplPath, err := utils.Resolve(args[0])
		if err != nil {
			return fmt.Errorf("resolve template path: %w", err)
		}

		imgs := make([]gocv.Mat, len(args)-1)
		for i, p := range args[1:] {
			resolved, err := utils.Resolve(p)
			if err != nil {
				for j := 0; j < i; j++ {
					imgs[j].Close()
				}
				return fmt.Errorf("resolve image path: %w", err)
			}
			imgs[i] = gocv.IMRead(resolved, gocv.IMReadGrayScale)
			if imgs[i].Empty() {
				for j := 0; j < i; j++ {
					imgs[j].Close()
				}
				return fmt.Errorf("could not read image: %s", p)
			}
		}
		defer func() {
			for i := range imgs {
				imgs[i].Close()
			}
		}()

		return doMark(imgs, tmplPath)
	},
}

func init() {
	rootCmd.AddCommand(markCmd)
}

// doMark evaluates imgs against the mark template at tmplPath and prints the report.
func doMark(imgs []gocv.Mat, tmplPath string) error {
	tmpl, err := loadMarkTemplate(tmplPath)
	if err != nil {
		return err
	}
	result, err := marker.Evaluate(imgs, tmpl)
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
