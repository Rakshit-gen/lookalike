package mover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanKeepsStructureAndAvoidsClashes(t *testing.T) {
	root, dest := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(dest, "trip"), 0o755)
	os.WriteFile(filepath.Join(dest, "trip", "a.jpg"), []byte("already here"), 0o644)

	moves, err := Plan(root, dest, []string{"trip/a.jpg", "b.jpg", "b.jpg"}, []int64{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dest, "trip", "a (2).jpg"),
		filepath.Join(dest, "b.jpg"),
		filepath.Join(dest, "b (2).jpg"),
	}
	for i, m := range moves {
		if m.To != want[i] {
			t.Errorf("move %d to %s, want %s", i, m.To, want[i])
		}
	}
	if moves[0].From != filepath.Join(root, "trip", "a.jpg") || moves[2].Size != 3 {
		t.Errorf("move 0 %+v", moves[0])
	}
}

func TestPlanRefusesDestinationInsideRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := Plan(root, filepath.Join(root, "dupes"), nil, nil); err == nil {
		t.Fatal("destination inside the library was accepted")
	}
	if _, err := Plan(root, root, nil, nil); err == nil {
		t.Fatal("destination equal to the library was accepted")
	}
	if _, err := Plan(root, root+"-dupes", nil, nil); err != nil {
		t.Fatalf("sibling folder rejected: %v", err)
	}
}
