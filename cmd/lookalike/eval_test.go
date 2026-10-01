package main

import (
	"testing"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/library"
)

func grp(paths ...string) group.Group {
	var g group.Group
	for _, p := range paths {
		g.Photos = append(g.Photos, &library.Entry{Path: p})
	}
	return g
}

func TestScore(t *testing.T) {
	found, wrong := score([]group.Group{
		grp("a-0/original.jpg", "a-0/half-size.jpg", "a-0/brighter.jpg"),
		// Two copies of b grouped without their original don't count.
		grp("b-1/half-size.jpg", "b-1/brighter.jpg"),
		// c and d merged: one wrong pair, but c's copy is still with c.
		grp("c-2/original.jpg", "c-2/half-size.jpg", "d-3/original.jpg"),
		// Three different originals: three wrong pairs.
		grp("e-4/original.jpg", "f-5/original.jpg", "g-6/original.jpg"),
	})
	if found["half-size"] != 2 || found["brighter"] != 1 {
		t.Fatalf("found %v", found)
	}
	if wrong != 4 {
		t.Fatalf("wrong %d", wrong)
	}
}
