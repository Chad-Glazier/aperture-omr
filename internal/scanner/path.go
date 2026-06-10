package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Expands a relative or home directory path into an OS-specific absolute file path.
func Resolve(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()

		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %v", err)
		}

		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to resolve absolute path: %v", err)
	}

	return filepath.Clean(abs), nil
}
