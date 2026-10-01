package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rakshit-gen/lookalike/internal/group"
	"github.com/Rakshit-gen/lookalike/internal/library"
)

// runForget removes everything lookalike stored about a folder: its
// NuclaDB tenant and its fingerprint cache. The photos are not touched.
func runForget(ctx context.Context, args []string) error {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: lookalike forget FOLDER")
		os.Exit(2)
	}
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	if err := nuclaClient().DeleteTenant(ctx, group.Tenant(&library.Library{Root: abs})); err != nil {
		return err
	}
	if err := library.Forget(dir, abs); err != nil {
		return err
	}
	fmt.Printf("forgot %s\n", abs)
	return nil
}
