package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/mover"
)

func runMove(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("move", flag.ExitOnError)
	threshold := fs.Float64("min", defaultThreshold, "minimum similarity (0 to 1) for two photos to be grouped")
	rotated := fs.Bool("rotated", false, "also match copies turned a quarter or half turn")
	apply := fs.Bool("apply", false, "actually move the files (without it, only print the plan)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: lookalike move [flags] FOLDER DEST\n\n"+
			"Moves every photo except the best copy in each group from FOLDER to DEST,\n"+
			"keeping subfolders. Nothing is deleted, and an undo log is written to DEST.\n"+
			"Without -apply it only prints what it would do.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(2)
	}
	folder, dest := fs.Arg(0), fs.Arg(1)
	lib, err := openLibrary(ctx, folder)
	if err != nil {
		return err
	}
	groups, err := group.Find(ctx, nuclaClient(), lib, group.Options{Threshold: *threshold, Neighbors: 10, Rotated: *rotated})
	if err != nil {
		return err
	}
	var rels []string
	var sizes []int64
	for _, g := range groups {
		for _, p := range g.Photos[1:] {
			rels = append(rels, p.Path)
			sizes = append(sizes, p.Size)
		}
	}
	if len(rels) == 0 {
		fmt.Println("no look-alike photos found, nothing to move")
		return nil
	}
	moves, err := mover.Plan(lib.Root, dest, rels, sizes)
	if err != nil {
		return err
	}
	var total int64
	for _, m := range moves {
		total += m.Size
	}

	if !*apply {
		for _, m := range moves {
			fmt.Printf("%s\n  -> %s\n", m.From, m.To)
		}
		fmt.Printf("\nwould move %d photos (%s). Run again with -apply to do it.\n", len(moves), humanSize(total))
		return nil
	}

	logPath := filepath.Join(dest, "lookalike-undo-"+time.Now().Format("20060102-150405")+".tsv")
	done, err := mover.Apply(moves, logPath)
	fmt.Printf("moved %d of %d photos to %s\n", done, len(moves), dest)
	fmt.Printf("to put them back: lookalike undo %q\n", logPath)
	if err != nil {
		return err
	}
	// Drop the moved photos from the index right away.
	_, err = openLibrary(ctx, folder)
	return err
}

func runUndo(ctx context.Context, args []string) error {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: lookalike undo LOGFILE")
		os.Exit(2)
	}
	r, err := mover.Undo(args[0])
	fmt.Printf("restored %d photos\n", r.Restored)
	for _, p := range r.Missing {
		fmt.Printf("not found, skipped: %s\n", p)
	}
	for _, p := range r.Blocked {
		fmt.Printf("another file is now at %s, left the moved copy where it is\n", p)
	}
	return err
}
