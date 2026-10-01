package mover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUndoRestores(t *testing.T) {
	root, dest := t.TempDir(), t.TempDir()
	for _, n := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		os.WriteFile(filepath.Join(root, n), []byte(n), 0o644)
	}
	moves, _ := Plan(root, dest, []string{"a.jpg", "b.jpg", "c.jpg"}, []int64{1, 1, 1})
	logPath := filepath.Join(dest, "undo.tsv")
	if _, err := Apply(moves, logPath); err != nil {
		t.Fatal(err)
	}
	// After the move: someone deleted the moved b, and put a new file at a's old path.
	os.Remove(moves[1].To)
	os.WriteFile(filepath.Join(root, "a.jpg"), []byte("newer"), 0o644)

	r, err := Undo(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if r.Restored != 1 || len(r.Missing) != 1 || len(r.Blocked) != 1 {
		t.Fatalf("result %+v", r)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "c.jpg")); string(b) != "c.jpg" {
		t.Fatal("c not restored")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.jpg")); string(b) != "newer" {
		t.Fatal("undo overwrote a newer file")
	}
}

func TestUndoRejectsOtherFiles(t *testing.T) {
	p := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(p, []byte("hello world\n"), 0o644)
	if _, err := Undo(p); err == nil {
		t.Fatal("a non-log file was accepted")
	}
}
