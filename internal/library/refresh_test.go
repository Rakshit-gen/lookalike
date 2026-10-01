package library

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePNG(t *testing.T, path string, shade uint8) {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.SetGray(x, y, color.Gray{uint8(x*3) + shade})
		}
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

func TestRefreshOnlyDecodesWhatChanged(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "a.png"), 0)
	writePNG(t, filepath.Join(root, "sub/b.png"), 10)
	os.WriteFile(filepath.Join(root, "broken.png"), []byte("nope"), 0o644)

	l, _ := Open(t.TempDir(), root)
	ch, err := l.Refresh(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.Fingerprinted) != 2 || len(ch.Failed) != 1 || ch.Failed["broken.png"] == nil {
		t.Fatalf("first refresh %+v", ch)
	}
	if e := l.Entries["sub/b.png"]; e == nil || e.Width != 40 || len(e.Vector) == 0 {
		t.Fatalf("entry %+v", e)
	}

	ch, _ = l.Refresh(nil)
	if len(ch.Fingerprinted) != 0 || ch.Unchanged != 2 {
		t.Fatalf("second refresh %+v", ch)
	}

	os.Remove(filepath.Join(root, "sub/b.png"))
	writePNG(t, filepath.Join(root, "a.png"), 50)
	later := time.Now().Add(time.Minute)
	os.Chtimes(filepath.Join(root, "a.png"), later, later)
	ch, _ = l.Refresh(nil)
	if len(ch.Fingerprinted) != 1 || ch.Fingerprinted[0].Path != "a.png" {
		t.Fatalf("modified file not refreshed: %+v", ch)
	}
	if len(ch.Removed) != 1 || ch.Removed[0].Path != "sub/b.png" || l.Entries["sub/b.png"] != nil {
		t.Fatalf("removed file still tracked: %+v", ch)
	}
}
