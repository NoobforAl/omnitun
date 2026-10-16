package manager

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NoobforAl/omnitun/server/store"
)

// maxBackupFile bounds a single file when restoring (the state is a few KB).
const maxBackupFile = 16 << 20

// backupFile reports whether a data-directory file belongs in a backup: the
// state, traffic counters, and certificates/keys. Locks, temp files and
// sing-box's cache are left out.
func backupFile(name string) bool {
	if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json", ".crt", ".key", ".pem":
		return true
	}
	return false
}

// Backup writes the server's state, keys and certificates as a .tar.gz. It
// contains every secret of the server; keep it private.
func (m *Manager) Backup(w io.Writer) error {
	if !m.Store.Exists() {
		return store.ErrNotInitialized
	}
	entries, err := os.ReadDir(m.Store.Dir)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if e.IsDir() || !backupFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.Store.Dir, e.Name()))
		if err != nil {
			return err
		}
		hdr := &tar.Header{Name: e.Name(), Mode: 0o600, Size: int64(len(data)), ModTime: m.Now()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

// Restore unpacks a backup into the data directory. It refuses to replace an
// existing server unless force is set. Only the expected flat files are
// accepted, so a crafted archive cannot write elsewhere.
func (m *Manager) Restore(r io.Reader, force bool) error {
	if m.Store.Exists() && !force {
		return fmt.Errorf("%w: %s already has a server; use --force to replace it", ErrConflict, m.Store.Dir)
	}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return invalid("not a backup file: %v", err)
	}
	files := map[string][]byte{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return invalid("damaged backup: %v", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			return invalid("backup contains %q, which is not a plain file", hdr.Name)
		}
		if !backupFile(hdr.Name) {
			return invalid("backup contains an unexpected file %q", hdr.Name)
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxBackupFile+1))
		if err != nil {
			return invalid("damaged backup: %v", err)
		}
		if len(data) > maxBackupFile {
			return invalid("file %q in the backup is too large", hdr.Name)
		}
		files[hdr.Name] = data
	}
	if files["state.json"] == nil {
		return invalid("the backup has no state.json")
	}
	var probe store.State
	if err := json.Unmarshal(files["state.json"], &probe); err != nil || probe.Server.PublicHost == "" {
		return invalid("the backup's state.json is not a server state")
	}
	if err := os.MkdirAll(m.Store.Dir, 0o700); err != nil {
		return err
	}
	// State last: a half-restored directory is never mistaken for a server.
	for name, data := range files {
		if name == "state.json" {
			continue
		}
		if err := os.WriteFile(filepath.Join(m.Store.Dir, name), data, 0o600); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(m.Store.Dir, "state.json"), files["state.json"], 0o600); err != nil {
		return err
	}
	// The restored state must load (and migrate) cleanly.
	if _, err := m.Store.Load(); err != nil {
		return invalid("the restored state is not valid: %v", err)
	}
	return nil
}
