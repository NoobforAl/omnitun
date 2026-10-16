package engine

import (
	"os"
	"testing"
	"time"
)

// tempDir is t.TempDir with a cleanup that retries: sing-box's cache file or
// a virus scanner can hold a file open for a moment after Stop on Windows,
// which fails t.TempDir's own cleanup.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-engine-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 60 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}
