package logs

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-logs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 40 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

func TestWriteAndTail(t *testing.T) {
	dir := tempDir(t)
	l := Open(dir)
	l.now = func() time.Time { return time.Date(2026, 10, 16, 18, 0, 0, 0, time.UTC) }
	if lines, err := Tail(dir, 10); err != nil || len(lines) != 0 {
		t.Fatalf("empty log: %v %v", lines, err)
	}
	l.Write("info", "started")
	l.Write("warn", "\x1b[31mcolour\x1b[0m and\nline breaks  ")
	l.Close()

	lines, err := Tail(dir, 10)
	if err != nil || len(lines) != 2 {
		t.Fatalf("lines = %q err %v", lines, err)
	}
	if lines[0] != "2026-10-16 18:00:00 INFO  started" || lines[1] != "2026-10-16 18:00:00 WARN  colour and line breaks" {
		t.Fatalf("lines = %q", lines)
	}
	if last, _ := Tail(dir, 1); len(last) != 1 || !strings.HasSuffix(last[0], "line breaks") {
		t.Fatalf("tail 1 = %q", last)
	}
	// A nil log is a valid no-op sink.
	var none *Log
	none.Write("info", "ignored")
}

func TestRotationKeepsTheNewestLines(t *testing.T) {
	dir := tempDir(t)
	l := Open(dir)
	defer l.Close()
	chunk := strings.Repeat("x", 1000)
	for i := 0; i < 5000; i++ { // ~5 MB: rotates once
		l.Write("info", fmt.Sprintf("%05d %s", i, chunk))
	}
	if _, err := os.Stat(l.path() + ".1"); err != nil {
		t.Fatal("no rotated file")
	}
	fi, _ := os.Stat(l.path())
	if fi.Size() > maxSize {
		t.Fatalf("log is %d bytes", fi.Size())
	}
	lines, err := Tail(dir, 3)
	if err != nil || len(lines) != 3 || !strings.Contains(lines[2], "04999") || !strings.Contains(lines[0], "04997") {
		t.Fatalf("tail = %q err %v", lines, err)
	}
	// Asking for more than the current file holds reads the rotated one.
	many, _ := Tail(dir, 4900)
	if len(many) != 4900 || !strings.Contains(many[len(many)-1], "04999") {
		t.Fatalf("got %d lines", len(many))
	}
}
