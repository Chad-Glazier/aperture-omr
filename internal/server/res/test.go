package res

import "testing"

// Creates temporary resources for testing purposes. In order for the resources
// to clean up properly, it's essential that the [ServerResources.Close] method
// is called.
func TestingResources(t *testing.T) ServerResources {
	if !testing.Testing() {
		panic("res: called TestingResources while outside of a test")
	}

	s, err := NewLocalResources(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
