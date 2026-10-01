package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/photo"
	"github.com/Rakshit-gen/lookalike/internal/variant"
)

// runEval measures lookalike on copies with known edits. FOLDER must hold
// photos that are all different from each other; every edited copy should
// land in its original's group, and no two originals should share a group.
func runEval(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	rotated := fs.Bool("rotated", false, "search rotated fingerprints too, as scan -rotated does")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: lookalike eval [flags] FOLDER\n\n"+
			"FOLDER must hold photos that are all different from each other. eval writes\n"+
			"edited copies of each to a temporary folder, indexes them in a temporary\n"+
			"NuclaDB tenant, and reports how many copies were grouped with their original\n"+
			"at a range of thresholds. Nothing in FOLDER is changed.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	src := fs.Arg(0)
	paths, err := library.Walk(src)
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "lookalike-eval-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	lib := filepath.Join(work, "photos")

	skipped := 0
	for i, rel := range paths {
		fmt.Fprintf(os.Stderr, "\rmaking copies %d/%d", i+1, len(paths))
		img, ok, err := decodeUpright(filepath.Join(src, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		if !ok {
			skipped++
			continue
		}
		name := strings.TrimSuffix(path.Base(rel), path.Ext(rel)) + fmt.Sprintf("-%d", i)
		if err := writeJPEG(filepath.Join(lib, name, "original.jpg"), img, 95); err != nil {
			return err
		}
		for _, k := range variant.Kinds {
			if err := writeJPEG(filepath.Join(lib, name, k.Name+".jpg"), k.Make(img), k.Quality); err != nil {
				return err
			}
		}
	}
	fmt.Fprintln(os.Stderr)
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "skipped %d photos with an EXIF rotation tag\n", skipped)
	}
	originals := len(paths) - skipped
	if originals < 2 {
		return fmt.Errorf("need at least 2 usable photos, found %d", originals)
	}

	l, err := library.Open(filepath.Join(work, "cache"), lib)
	if err != nil {
		return err
	}
	ch, err := l.Refresh(nil)
	if err != nil {
		return err
	}
	if len(ch.Failed) > 0 {
		return fmt.Errorf("%d generated copies failed to load", len(ch.Failed))
	}
	db := nuclaClient()
	defer db.DeleteTenant(context.WithoutCancel(ctx), group.Tenant(l))
	if err := group.Push(ctx, db, l, ch); err != nil {
		return err
	}

	fmt.Printf("%d originals, %d copies each (%s)\n\n", originals, len(variant.Kinds), kindNames())
	fmt.Printf("%-9s", "min")
	for _, k := range variant.Kinds {
		fmt.Printf(" %12s", k.Name)
	}
	fmt.Printf(" %14s\n", "wrong merges")
	for _, th := range []float64{0.80, 0.85, 0.90, 0.93, 0.95, 0.97} {
		groups, err := group.Find(ctx, db, l, group.Options{Threshold: th, Neighbors: 10, Rotated: *rotated})
		if err != nil {
			return err
		}
		found, wrong := score(groups)
		fmt.Printf("%-9.2f", th)
		for _, k := range variant.Kinds {
			fmt.Printf(" %11.0f%%", 100*float64(found[k.Name])/float64(originals))
		}
		fmt.Printf(" %14d\n", wrong)
	}
	fmt.Println("\nEach percentage is the share of copies grouped with their own original.")
	fmt.Println("Wrong merges counts pairs of different originals that ended up in one group.")
	return nil
}

// score counts, per edit kind, the copies that share a group with their
// original, and the pairs of different originals put in the same group.
// Copies live at NAME/KIND.jpg, so the folder says which original they
// belong to.
func score(groups []group.Group) (found map[string]int, wrong int) {
	found = map[string]int{}
	for _, g := range groups {
		bySource := map[string][]string{}
		for _, p := range g.Photos {
			dir, file := path.Split(p.Path)
			bySource[dir] = append(bySource[dir], strings.TrimSuffix(file, ".jpg"))
		}
		n := len(bySource)
		wrong += n * (n - 1) / 2
		for _, kinds := range bySource {
			hasOriginal := false
			for _, k := range kinds {
				hasOriginal = hasOriginal || k == "original"
			}
			if !hasOriginal {
				continue
			}
			for _, k := range kinds {
				if k != "original" {
					found[k]++
				}
			}
		}
	}
	return found, wrong
}

// decodeUpright decodes a photo, reporting ok=false for JPEGs with an EXIF
// rotation tag: their copies would be written without the tag, so they
// would not be fair test cases.
func decodeUpright(p string) (image.Image, bool, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	if photo.Orientation(f) != 1 {
		return nil, false, nil
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, false, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", p, err)
	}
	return img, true, nil
}

func writeJPEG(p string, img image.Image, quality int) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: quality}); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func kindNames() string {
	names := make([]string, len(variant.Kinds))
	for i, k := range variant.Kinds {
		names[i] = k.Name
	}
	return strings.Join(names, ", ")
}
