package group

import (
	"context"
	"testing"
	"time"

	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/nucla"
	"github.com/Rakshit-gen/lookalike/internal/nucla/nuclafake"
	"github.com/Rakshit-gen/lookalike/internal/photo"
)

func TestFindGroupsAndPicksKeeper(t *testing.T) {
	fake := nuclafake.New()
	defer fake.Close()
	db := nucla.New(fake.URL)
	ctx := context.Background()

	l := &library.Library{Root: "/photos", Entries: map[string]*library.Entry{}}
	add := func(path string, w, h int, size int64, v ...float32) *library.Entry {
		e := &library.Entry{Path: path, Width: w, Height: h, Size: size, ModTime: time.Unix(0, 0), Vector: v}
		l.Entries[path] = e
		return e
	}
	add("beach.jpg", 4000, 3000, 5_000_000, 1, 0, 0)
	add("beach-whatsapp.jpg", 1600, 1200, 200_000, 0.99, 0.05, 0)
	add("beach-edit.jpg", 4000, 3000, 4_000_000, 0.97, 0.1, 0.05)
	add("dog.jpg", 3000, 4000, 3_000_000, 0, 1, 0)
	add("dog-copy.png", 3000, 4000, 9_000_000, 0, 1, 0.01)
	add("receipt.jpg", 1000, 2000, 300_000, 0, 0, 1)

	var all []*library.Entry
	for _, e := range l.Entries {
		all = append(all, e)
	}
	if err := Push(ctx, db, l, library.Changes{Fingerprinted: all}); err != nil {
		t.Fatal(err)
	}

	groups, err := Find(ctx, db, l, Options{Threshold: 0.95, Neighbors: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d: %+v", len(groups), groups)
	}
	beach := groups[0]
	if len(beach.Photos) != 3 || beach.Photos[0].Path != "beach.jpg" || beach.Photos[2].Path != "beach-whatsapp.jpg" {
		t.Fatalf("beach group order: %v %v %v", beach.Photos[0].Path, beach.Photos[1].Path, beach.Photos[2].Path)
	}
	if beach.Loosest < 0.95 || beach.Closest < beach.Loosest {
		t.Fatalf("closest %.3f loosest %.3f", beach.Closest, beach.Loosest)
	}
	dog := groups[1]
	// Same pixel count, so the bigger file (the PNG) is kept.
	if dog.Photos[0].Path != "dog-copy.png" {
		t.Fatalf("dog keeper %s", dog.Photos[0].Path)
	}
}

func TestFindRotatedCatchesTurnedCopies(t *testing.T) {
	fake := nuclafake.New()
	defer fake.Close()
	db := nucla.New(fake.URL)
	ctx := context.Background()

	upright := make([]float32, photo.Dim)
	for i := range upright {
		upright[i] = float32((i*37)%11) - 5
	}
	l := &library.Library{Root: "/photos", Entries: map[string]*library.Entry{
		"a.jpg":         {Path: "a.jpg", Vector: upright},
		"a-rotated.jpg": {Path: "a-rotated.jpg", Vector: photo.Transform(upright, 8)},
	}}
	var all []*library.Entry
	for _, e := range l.Entries {
		all = append(all, e)
	}
	if err := Push(ctx, db, l, library.Changes{Fingerprinted: all}); err != nil {
		t.Fatal(err)
	}
	plain, _ := Find(ctx, db, l, Options{Threshold: 0.9, Neighbors: 5})
	rotated, _ := Find(ctx, db, l, Options{Threshold: 0.9, Neighbors: 5, Rotated: true})
	if len(plain) != 0 || len(rotated) != 1 {
		t.Fatalf("plain %d groups, rotated %d groups", len(plain), len(rotated))
	}
}
