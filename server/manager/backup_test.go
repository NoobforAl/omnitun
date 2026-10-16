package manager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/NoobforAl/omnitun/server/store"
)

func TestBackupRestoreRoundTrip(t *testing.T) {
	src := newManager(t)
	if _, err := src.AddUser(NewUser{Name: "alice", QuotaGB: 3}); err != nil {
		t.Fatal(err)
	}
	// Files that must not travel.
	os.WriteFile(filepath.Join(src.Store.Dir, "cache.db"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(src.Store.Dir, "state.json.lock"), nil, 0o600)

	var buf bytes.Buffer
	if err := src.Backup(&buf); err != nil {
		t.Fatal(err)
	}
	names := tarNames(t, buf.Bytes())
	for _, want := range []string{"state.json", "tls.crt", "tls.key"} {
		if !names[want] {
			t.Errorf("backup lacks %s (has %v)", want, names)
		}
	}
	if names["cache.db"] || names["state.json.lock"] {
		t.Errorf("backup holds files it should not: %v", names)
	}

	dst := New(store.Open(tempDir(t)))
	if err := dst.Restore(bytes.NewReader(buf.Bytes()), false); err != nil {
		t.Fatal(err)
	}
	want, _ := src.Links("alice")
	got, err := dst.Links("alice")
	if err != nil || got.Subscription != want.Subscription || got.Links[0].URL != want.Links[0].URL {
		t.Fatalf("restored links differ: %v\n%+v\n%+v", err, got, want)
	}
	if _, err := os.Stat(filepath.Join(dst.Store.Dir, "tls.key")); err != nil {
		t.Fatal("certificate key not restored")
	}

	// An existing server is not overwritten unless asked.
	if err := dst.Restore(bytes.NewReader(buf.Bytes()), false); !errors.Is(err, ErrConflict) {
		t.Fatalf("restore over a server: %v", err)
	}
	if err := dst.Restore(bytes.NewReader(buf.Bytes()), true); err != nil {
		t.Fatalf("forced restore: %v", err)
	}
}

func tarNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err != nil {
			return out
		}
		out[h.Name] = true
	}
}

// makeArchive builds a .tar.gz from name -> content (a name ending in "/"
// becomes a directory entry).
func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		h := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if name[len(name)-1] == '/' {
			h.Typeflag, h.Size = tar.TypeDir, 0
		}
		tw.WriteHeader(h)
		tw.Write([]byte(content))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestRestoreRejectsBadArchives(t *testing.T) {
	good := makeArchive(t, map[string]string{"state.json": "{}"})
	_ = good
	for name, data := range map[string][]byte{
		"not gzip":          []byte("hello"),
		"path traversal":    makeArchive(t, map[string]string{"state.json": "{}", "../evil.json": "{}"}),
		"nested path":       makeArchive(t, map[string]string{"state.json": "{}", "sub/x.json": "{}"}),
		"unexpected file":   makeArchive(t, map[string]string{"state.json": "{}", "run.sh": "rm -rf /"}),
		"directory entry":   makeArchive(t, map[string]string{"state.json": "{}", "dir/": ""}),
		"no state":          makeArchive(t, map[string]string{"tls.crt": "x"}),
		"state is not json": makeArchive(t, map[string]string{"state.json": "not json"}),
	} {
		dir := tempDir(t)
		dst := New(store.Open(filepath.Join(dir, "data")))
		if err := dst.Restore(bytes.NewReader(data), false); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want invalid input", name, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "evil.json")); err == nil {
			t.Errorf("%s: wrote outside the data directory", name)
		}
	}
}
