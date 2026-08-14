package cmd

import (
	"fmt"
	"strings"
	"runtime"

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
		s := "\n"
		s += "  Aperture OMR       \u001B[96m" +
			version + "\u001B[0m\n"
		s += "  ─┬──────────\n"
		s += fmt.Sprintf(
			"   ├──go            %s\n", 
			strings.Split(strings.TrimPrefix(runtime.Version(), "go"), "-")[0],
		)
		s += fmt.Sprintf("   ├──opencv        %s\n", gocv.OpenCVVersion())
		s += fmt.Sprintf("   └──mupdf         %s\n", fitz.FzVersion)
		s += "\n"

		fmt.Print(s)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
