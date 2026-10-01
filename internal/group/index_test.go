package group

import (
	"context"
	"testing"

	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/nucla"
	"github.com/Rakshit-gen/lookalike/internal/nucla/nuclafake"
)

func TestPushUpsertsAndDeletes(t *testing.T) {
	fake := nuclafake.New()
	defer fake.Close()
	db := nucla.New(fake.URL)
	ctx := context.Background()
	l := &library.Library{Root: "/photos", Entries: map[string]*library.Entry{}}
	a := &library.Entry{Path: "a.jpg", Vector: []float32{1, 0}}
	b := &library.Entry{Path: "b.jpg", Vector: []float32{0, 1}}

	if err := Push(ctx, db, l, library.Changes{Fingerprinted: []*library.Entry{a, b}}); err != nil {
		t.Fatal(err)
	}
	if n := fake.Count(Tenant(l)); n != 2 {
		t.Fatalf("count %d", n)
	}
	if err := Push(ctx, db, l, library.Changes{Removed: []*library.Entry{b}}); err != nil {
		t.Fatal(err)
	}
	if n := fake.Count(Tenant(l)); n != 1 {
		t.Fatalf("count after delete %d", n)
	}
}
