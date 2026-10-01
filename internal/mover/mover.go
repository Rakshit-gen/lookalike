// Package mover moves extra copies out of a photo folder and can put them
// back. It never deletes a photo and never overwrites a file.
package mover

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Move is one file to move.
type Move struct {
	From, To string // absolute paths
	Size     int64
}

// Plan lists where each extra copy would go: the same relative path under
// dest. If that name is taken, " (2)", " (3)" and so on are added before
// the extension. Nothing is moved.
func Plan(root, dest string, rels []string, sizes []int64) ([]Move, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return nil, err
	}
	if inside(dest, root) {
		return nil, fmt.Errorf("destination %s is inside %s; the moved copies would be scanned again", dest, root)
	}
	taken := map[string]bool{}
	moves := make([]Move, len(rels))
	for i, rel := range rels {
		to := freeName(filepath.Join(dest, filepath.FromSlash(rel)), taken)
		taken[to] = true
		moves[i] = Move{From: filepath.Join(root, filepath.FromSlash(rel)), To: to, Size: sizes[i]}
	}
	return moves, nil
}

func inside(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// freeName returns p, or p with a " (n)" suffix, such that nothing exists
// there on disk and no earlier move in the plan claimed it.
func freeName(p string, taken map[string]bool) string {
	ext := filepath.Ext(p)
	base := strings.TrimSuffix(p, ext)
	for n := 1; ; n++ {
		c := p
		if n > 1 {
			c = fmt.Sprintf("%s (%d)%s", base, n, ext)
		}
		if _, err := os.Lstat(c); errors.Is(err, fs.ErrNotExist) && !taken[c] {
			return c
		}
	}
}
