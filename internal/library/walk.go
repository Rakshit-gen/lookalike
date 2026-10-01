// Package library tracks the photos under one folder: which files exist,
// their fingerprints, and which ones changed since the last run.
package library

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var photoExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}

// Walk lists the photos under root as slash-separated paths relative to
// it. Hidden files and folders (names starting with a dot) are skipped;
// they are usually app caches and thumbnails, not photos anyone took.
// HEIC and RAW files are skipped too, since there is no pure Go decoder
// for them.
func Walk(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && photoExt[strings.ToLower(filepath.Ext(path))] {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}
