package engine

import "os"

func osTempDir() string { return os.TempDir() }

func ensureDir(dir string) error { return os.MkdirAll(dir, 0o700) }
