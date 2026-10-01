// Command lookalike finds photos that are near copies of each other.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/Rakshit-gen/lookalike/internal/nucla"
)

const usage = `lookalike finds photos that are near copies of each other.

Usage:
  lookalike scan FOLDER          index the photos in FOLDER and list look-alike groups
  lookalike similar FOLDER PHOTO list the photos in FOLDER that look like PHOTO
  lookalike forget FOLDER        drop the index and cache for FOLDER (photos are not touched)

Environment:
  NUCLADB_URL      NuclaDB REST address (default http://localhost:8080)
  NUCLADB_API_KEY  key for a NuclaDB server started with -api-keys
  LOOKALIKE_CACHE  where fingerprints are kept (default: your user cache dir)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "scan":
		err = runScan(ctx, args)
	case "similar":
		err = runSimilar(ctx, args)
	case "forget":
		err = runForget(ctx, args)
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "lookalike: unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "lookalike:", err)
		os.Exit(1)
	}
}

func nuclaClient() *nucla.Client {
	url := os.Getenv("NUCLADB_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	c := nucla.New(url)
	c.APIKey = os.Getenv("NUCLADB_API_KEY")
	return c
}

func cacheDir() (string, error) {
	if d := os.Getenv("LOOKALIKE_CACHE"); d != "" {
		return d, nil
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "lookalike"), nil
}
