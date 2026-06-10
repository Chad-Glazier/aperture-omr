package scanner

import (
	"strings"
	"testing"
)

func assertError(t *testing.T, err error, expectError bool, errContains string) {
	t.Helper() // Tells Go test runner to report failures at the caller's line number

	if expectError {
		if err == nil {
			t.Fatalf("expected an error containing %q, but got nil", errContains)
		}
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("expected error to contain %q, but got %q", errContains, err.Error())
		}
		return
	}

	if err != nil {
		t.Fatalf("did not expect an error, but got: %v", err)
	}
}
