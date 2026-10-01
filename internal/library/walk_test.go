package library

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func touch(t *testing.T, root string, rel ...string) {
	t.Helper()
	for _, r := range rel {
		p := filepath.Join(root, r)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWalkFindsPhotosOnly(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "a.jpg", "trip/B.JPEG", "trip/c.png", "anim.gif",
		"notes.txt", "raw.heic", ".thumbnails/x.jpg", "trip/.hidden.jpg")
	got, err := Walk(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{"a.jpg", "anim.gif", "trip/B.JPEG", "trip/c.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
