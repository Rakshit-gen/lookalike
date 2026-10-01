package main

import (
	"strings"
	"testing"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/library"
)

func TestReportEscapesPathsAndMarksKeeper(t *testing.T) {
	g := group.Group{Photos: []*library.Entry{
		{Path: "trip/best.jpg", Width: 4000, Height: 3000, Size: 5 << 20},
		{Path: `trip/<script>alert(1)</script> #2.jpg`, Width: 800, Height: 600, Size: 90 << 10},
	}, Closest: 0.99, Loosest: 0.97}
	var b strings.Builder
	err := reportPage.Execute(&b, map[string]any{"Root": "/home/me/Photos", "Groups": []group.Group{g}, "Threshold": 0.9})
	if err != nil {
		t.Fatal(err)
	}
	html := b.String()
	if strings.Contains(html, "<script>alert") {
		t.Fatal("file name not escaped")
	}
	if !strings.Contains(html, `src="file:///home/me/Photos/trip/best.jpg"`) {
		t.Fatal("missing file URL for keeper")
	}
	// '#' must be encoded in the URL or the browser treats it as a fragment.
	if !strings.Contains(html, "%23") {
		t.Fatal("# not encoded in file URL")
	}
	if strings.Count(html, `class="keep"`) != 1 {
		t.Fatal("keeper not marked exactly once")
	}
}
