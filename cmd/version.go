package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Chad-Glazier/aperture-omr/internal/sys"
	"github.com/gen2brain/go-fitz"
	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

const version = "0.0.1"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of the service and its major dependencies",
	Long:  `Prints the version of the service and its major dependencies.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(
			`
  Aperture OMR          %s
  ─┬────────────────────────────
   ├──go                %s
   ├──opencv            %s
   └──mupdf             %s

`,
			sys.FgBrightCyan(version),
			strings.Split(strings.TrimPrefix(runtime.Version(), "go"), "-")[0],
			gocv.OpenCVVersion(),
			fitz.FzVersion,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
