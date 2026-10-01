package library

import (
	"testing"
	"time"
)

func TestSaveOpenRoundTrip(t *testing.T) {
	cache, root := t.TempDir(), t.TempDir()
	l, err := Open(cache, root)
	if err != nil || len(l.Entries) != 0 {
		t.Fatalf("fresh open: %+v %v", l, err)
	}
	l.Entries["a.jpg"] = &Entry{Path: "a.jpg", Size: 10, ModTime: time.Unix(100, 0).UTC(), Width: 4, Height: 3, Vector: []float32{1, 2}}
	if err := l.Save(cache); err != nil {
		t.Fatal(err)
	}
	got, err := Open(cache, root)
	if err != nil {
		t.Fatal(err)
	}
	e := got.Entries["a.jpg"]
	if e == nil || e.Size != 10 || e.Width != 4 || len(e.Vector) != 2 || !e.ModTime.Equal(time.Unix(100, 0)) {
		t.Fatalf("got %+v", e)
	}
}

func TestIDsAreStableAndDistinct(t *testing.T) {
	if PathID("a.jpg") != PathID("a.jpg") || PathID("a.jpg") == PathID("b.jpg") {
		t.Fatal("ids not stable or not distinct")
	}
	if Key("/photos") == Key("/photos2") {
		t.Fatal("library keys collide")
	}
}

func TestForget(t *testing.T) {
	cache, root := t.TempDir(), t.TempDir()
	l, _ := Open(cache, root)
	l.Entries["a.jpg"] = &Entry{Path: "a.jpg"}
	l.Save(cache)
	if err := Forget(cache, root); err != nil {
		t.Fatal(err)
	}
	if l, _ := Open(cache, root); len(l.Entries) != 0 {
		t.Fatal("cache survived Forget")
	}
	if err := Forget(cache, root); err != nil {
		t.Fatalf("second Forget: %v", err)
	}
}
