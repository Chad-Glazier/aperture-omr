package main

//
// This entrypoint just prints the version of the project and its dependencies.
//

import (
	"fmt"
	"runtime"
	"ubc/team15/config"

	"gocv.io/x/gocv"
)

func main() {
	s := "\n"
	s += "  OMR version           \u001B[96m" + config.VERSION + "\u001B[0m\n"
	s += "  ─┬─\n"
	s += fmt.Sprintf("   ├──runtime           %s\n", runtime.Version())
	s += fmt.Sprintf("   ├──gocv version      %s\n", gocv.Version())
	s += fmt.Sprintf("   └──opencv version    %s\n", gocv.OpenCVVersion())
	s += "\n"

	fmt.Print(s)
}
