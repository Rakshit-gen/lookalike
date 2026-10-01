package mover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyMovesAndLogs(t *testing.T) {
	root, dest := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "trip"), 0o755)
	os.WriteFile(filepath.Join(root, "trip", "a copy.jpg"), []byte("A"), 0o644)
	os.WriteFile(filepath.Join(root, "b.jpg"), []byte("B"), 0o644)

	moves, err := Plan(root, dest, []string{"trip/a copy.jpg", "b.jpg"}, []int64{1, 1})
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dest, "undo.tsv")
	n, err := Apply(moves, logPath)
	if err != nil || n != 2 {
		t.Fatalf("applied %d: %v", n, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "trip", "a copy.jpg")); string(b) != "A" {
		t.Fatal("file not at destination")
	}
	if _, err := os.Stat(filepath.Join(root, "b.jpg")); !os.IsNotExist(err) {
		t.Fatal("source still there")
	}
	log, _ := os.ReadFile(logPath)
	if strings.Count(string(log), "\n") != 2 || !strings.Contains(string(log), `"`+filepath.Join(root, "trip", "a copy.jpg")+`"`) {
		t.Fatalf("log %q", log)
	}
}

func TestApplyNeverOverwrites(t *testing.T) {
	root, dest := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(root, "a.jpg"), []byte("new"), 0o644)
	moves, _ := Plan(root, dest, []string{"a.jpg"}, []int64{3})
	// Something appears at the destination after planning.
	os.WriteFile(moves[0].To, []byte("precious"), 0o644)

	n, err := Apply(moves, filepath.Join(dest, "undo.tsv"))
	if err == nil || n != 0 {
		t.Fatalf("applied %d, err %v", n, err)
	}
	if b, _ := os.ReadFile(moves[0].To); string(b) != "precious" {
		t.Fatal("existing file was overwritten")
	}
	if b, _ := os.ReadFile(moves[0].From); string(b) != "new" {
		t.Fatal("source was lost")
	}
}

func TestCopyFileKeepsContentAndTime(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	os.WriteFile(from, []byte("pixels"), 0o600)
	if err := copyFile(from, to); err != nil {
		t.Fatal(err)
	}
	a, _ := os.Stat(from)
	b, _ := os.Stat(to)
	if got, _ := os.ReadFile(to); string(got) != "pixels" || !a.ModTime().Equal(b.ModTime()) {
		t.Fatalf("copy differs: %q %v %v", got, a.ModTime(), b.ModTime())
	}
	if err := copyFile(from, to); err == nil {
		t.Fatal("copy over an existing file was allowed")
	}
}
