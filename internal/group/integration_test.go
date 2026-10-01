package group

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/nucla"
	"github.com/Rakshit-gen/lookalike/internal/variant"
)

// TestAgainstRealNuclaDB runs real files through fingerprinting, a live
// server's HNSW search and grouping. It is skipped unless NUCLADB_TEST_URL
// is set, for example:
//
//	nucladbd -data-dir=/tmp/n &
//	NUCLADB_TEST_URL=http://localhost:8080 go test ./internal/group
func TestAgainstRealNuclaDB(t *testing.T) {
	url := os.Getenv("NUCLADB_TEST_URL")
	if url == "" {
		t.Skip("NUCLADB_TEST_URL not set")
	}
	ctx := context.Background()
	root := t.TempDir()
	write := func(name string, img image.Image) {
		f, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
	}
	stripes, blob := scene(true), scene(false)
	write("stripes.jpg", stripes)
	write("stripes-small.jpg", variant.Half(stripes))
	write("blob.jpg", blob)

	l, err := library.Open(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := l.Refresh(nil)
	if err != nil {
		t.Fatal(err)
	}
	db := nucla.New(url)
	defer db.DeleteTenant(ctx, Tenant(l))
	if err := Push(ctx, db, l, ch); err != nil {
		t.Fatal(err)
	}
	groups, err := Find(ctx, db, l, Options{Threshold: 0.9, Neighbors: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Photos) != 2 || groups[0].Photos[0].Path != "stripes.jpg" {
		t.Fatalf("groups %+v", groups)
	}
}

// scene draws a 400x300 test picture: diagonal stripes or a bright blob,
// different enough that they should never group.
func scene(stripes bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := range 300 {
		for x := range 400 {
			v := uint8(40)
			if stripes && (x+y)/25%2 == 0 {
				v = 220
			}
			if !stripes && (x-120)*(x-120)+(y-200)*(y-200) < 70*70 {
				v = 230
			}
			img.Set(x, y, color.RGBA{v, v, uint8(x / 2), 255})
		}
	}
	return img
}
