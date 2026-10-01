package group

import (
	"context"
	"sort"

	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/nucla"
)

// Group is a set of photos that look alike. Photos[0] is the one to keep:
// the most pixels, then the biggest file, then the oldest, then by path.
type Group struct {
	Photos []*library.Entry `json:"photos"`
	// Closest is the highest similarity between any two members, Loosest
	// the lowest similarity of an edge that joined the group.
	Closest float64 `json:"closest"`
	Loosest float64 `json:"loosest"`
}

// Options control how alike photos must be to share a group.
type Options struct {
	Threshold float64 // minimum cosine similarity between two photos
	Neighbors int     // how many nearest photos to look at for each one
}

// Find asks NuclaDB for each photo's nearest neighbours and joins every
// pair at or above the threshold.
//
// ponytail: single linkage. A burst of 20 slowly changing shots chains
// into one group even when the first and last differ a lot. That is
// usually what people want for bursts; if not, raise the threshold.
func Find(ctx context.Context, db *nucla.Client, l *library.Library, opt Options) ([]Group, error) {
	entries := l.Sorted()
	byID := make(map[string]int, len(entries))
	for i, e := range entries {
		byID[e.ID()] = i
	}
	parent := make([]int, len(entries))
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	closest := map[int]float64{}
	loosest := map[int]float64{}
	type edge struct {
		a, b int
		sim  float64
	}
	var edges []edge
	for i, e := range entries {
		ms, err := db.Search(ctx, Tenant(l), e.Vector, opt.Neighbors+1)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			j, ok := byID[m.ID]
			if !ok || j == i {
				continue
			}
			if sim := 1 - float64(m.Score); sim >= opt.Threshold && i < j {
				edges = append(edges, edge{i, j, sim})
			} else if sim >= opt.Threshold && i > j {
				edges = append(edges, edge{j, i, sim})
			}
		}
	}
	for _, ed := range edges {
		ra, rb := root(ed.a), root(ed.b)
		if ra != rb {
			parent[ra] = rb
		}
	}
	for _, ed := range edges {
		r := root(ed.a)
		if ed.sim > closest[r] {
			closest[r] = ed.sim
		}
		if l, ok := loosest[r]; !ok || ed.sim < l {
			loosest[r] = ed.sim
		}
	}

	members := map[int][]*library.Entry{}
	for i, e := range entries {
		r := root(i)
		members[r] = append(members[r], e)
	}
	var out []Group
	for r, ps := range members {
		if len(ps) < 2 {
			continue
		}
		sort.Slice(ps, func(i, j int) bool { return better(ps[i], ps[j]) })
		out = append(out, Group{Photos: ps, Closest: closest[r], Loosest: loosest[r]})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Photos) != len(out[j].Photos) {
			return len(out[i].Photos) > len(out[j].Photos)
		}
		return out[i].Photos[0].Path < out[j].Photos[0].Path
	})
	return out, nil
}

// better reports whether a is a better copy to keep than b.
func better(a, b *library.Entry) bool {
	if pa, pb := a.Width*a.Height, b.Width*b.Height; pa != pb {
		return pa > pb
	}
	if a.Size != b.Size {
		return a.Size > b.Size
	}
	if !a.ModTime.Equal(b.ModTime) {
		return a.ModTime.Before(b.ModTime)
	}
	return a.Path < b.Path
}
