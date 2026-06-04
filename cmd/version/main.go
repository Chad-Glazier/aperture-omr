package main

import (
	"fmt"
	"runtime"
	"ubc/team15/internal/term"

	"gocv.io/x/gocv"
)

const VERSION = "0.0.1"

func main() {
	s := "\n"
	s += fmt.Sprintf("  OMR version           %s\n", term.FgBrightCyan(VERSION))
	s += "  ─┬─\n"
	s += fmt.Sprintf("   ├──runtime           %s\n", term.FgCyan(runtime.Version()))
	s += fmt.Sprintf("   ├──gocv version      %s\n", term.FgCyan(gocv.Version()))
	s += fmt.Sprintf("   └──opencv version    %s\n", term.FgCyan(gocv.OpenCVVersion()))
	s += "\n"

	fmt.Printf(s)
}
