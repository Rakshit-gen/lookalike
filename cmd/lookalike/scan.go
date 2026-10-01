package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/library"
)

// defaultThreshold is the similarity two photos need to be grouped.
const defaultThreshold = 0.9

// openLibrary loads the cached library for folder, brings it up to date
// with the disk, and pushes any changes to NuclaDB.
func openLibrary(ctx context.Context, folder string) (*library.Library, error) {
	st, err := os.Stat(folder)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", folder)
	}
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}
	lib, err := library.Open(dir, folder)
	if err != nil {
		return nil, err
	}
	ch, err := lib.Refresh(func(done, total int) {
		if done%50 == 0 || done == total {
			fmt.Fprintf(os.Stderr, "\rfingerprinting %d/%d", done, total)
		}
	})
	if err != nil {
		return nil, err
	}
	if len(ch.Fingerprinted) > 0 {
		fmt.Fprintln(os.Stderr)
	}
	if len(ch.Failed) > 0 {
		paths := make([]string, 0, len(ch.Failed))
		for p := range ch.Failed {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		fmt.Fprintf(os.Stderr, "skipped %d files that could not be read:\n", len(paths))
		for _, p := range paths {
			fmt.Fprintf(os.Stderr, "  %v\n", ch.Failed[p])
		}
	}
	if err := group.Push(ctx, nuclaClient(), lib, ch); err != nil {
		return nil, fmt.Errorf("indexing: %w", err)
	}
	// Save only after NuclaDB has the vectors, so a failed push is retried
	// next run instead of being skipped as unchanged.
	if err := lib.Save(dir); err != nil {
		return nil, err
	}
	return lib, nil
}

func runScan(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	threshold := fs.Float64("min", defaultThreshold, "minimum similarity (0 to 1) for two photos to be grouped")
	rotated := fs.Bool("rotated", false, "also match copies turned a quarter or half turn (4x slower)")
	asJSON := fs.Bool("json", false, "print the groups as JSON")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: lookalike scan [flags] FOLDER\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	lib, err := openLibrary(ctx, fs.Arg(0))
	if err != nil {
		return err
	}
	groups, err := group.Find(ctx, nuclaClient(), lib, group.Options{Threshold: *threshold, Neighbors: 10, Rotated: *rotated})
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(lib.Root, groups)
	}
	extra := 0
	for _, g := range groups {
		fmt.Printf("\n%d photos, similarity %.2f to %.2f\n", len(g.Photos), g.Loosest, g.Closest)
		for i, p := range g.Photos {
			mark := "  "
			if i == 0 {
				mark = "* "
			}
			fmt.Printf("  %s%s  (%dx%d, %s)\n", mark, p.Path, p.Width, p.Height, humanSize(p.Size))
		}
		extra += len(g.Photos) - 1
	}
	fmt.Printf("\n%d photos indexed, %d groups, %d photos that look like another one\n", len(lib.Entries), len(groups), extra)
	if len(groups) > 0 {
		fmt.Println("* marks the copy lookalike would keep")
	}
	return nil
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// printJSON writes groups with absolute paths, which is what a script
// acting on the output needs.
func printJSON(root string, groups []group.Group) error {
	type photoJSON struct {
		Path   string `json:"path"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Bytes  int64  `json:"bytes"`
		Keep   bool   `json:"keep"`
	}
	type groupJSON struct {
		Closest float64     `json:"closest"`
		Loosest float64     `json:"loosest"`
		Photos  []photoJSON `json:"photos"`
	}
	out := make([]groupJSON, 0, len(groups))
	for _, g := range groups {
		gj := groupJSON{Closest: g.Closest, Loosest: g.Loosest}
		for i, p := range g.Photos {
			gj.Photos = append(gj.Photos, photoJSON{
				Path: filepath.Join(root, filepath.FromSlash(p.Path)), Width: p.Width, Height: p.Height, Bytes: p.Size, Keep: i == 0,
			})
		}
		out = append(out, gj)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
