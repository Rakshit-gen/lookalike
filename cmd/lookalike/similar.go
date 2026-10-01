package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/photo"
)

func runSimilar(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("similar", flag.ExitOnError)
	n := fs.Int("n", 5, "how many photos to show")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: lookalike similar [flags] FOLDER PHOTO\n\n"+
			"Shows the photos in FOLDER that look most like PHOTO. PHOTO can be outside FOLDER.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(2)
	}
	lib, err := openLibrary(ctx, fs.Arg(0))
	if err != nil {
		return err
	}
	info, err := photo.Load(fs.Arg(1))
	if err != nil {
		return err
	}
	ms, err := nuclaClient().Search(ctx, group.Tenant(lib), info.Vector, *n)
	if err != nil {
		return err
	}
	for _, m := range ms {
		fmt.Printf("%.3f  %s\n", 1-m.Score, m.Metadata["path"])
	}
	return nil
}
