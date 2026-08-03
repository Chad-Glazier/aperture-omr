//go:build linux || darwin
package sys

import (
	"runtime/debug"
)

/*
#include <malloc.h>
*/
import "C"

// Attempts to release as much memory to the OS as possible. Be warned that
// forcing a GC may pause the program momentarily and it should not be used too
// often.
func Tidy() {
	C.malloc_trim(0)
	debug.FreeOSMemory()
}
