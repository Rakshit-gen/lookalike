package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// Entry is one photo in the library.
type Entry struct {
	Path    string    `json:"path"` // relative to the library root, slash separated
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mtime"`
	Width   int       `json:"w"`
	Height  int       `json:"h"`
	Vector  []float32 `json:"v"`
}

// ID is the NuclaDB vector id for the entry: a hash of its path, since
// NuclaDB ids are decimal uint64s.
func (e *Entry) ID() string { return PathID(e.Path) }

func PathID(rel string) string {
	h := fnv.New64a()
	h.Write([]byte(rel))
	return strconv.FormatUint(h.Sum64(), 10)
}

type Library struct {
	Root    string            `json:"root"` // absolute
	Entries map[string]*Entry `json:"entries"`
}

// Key names a library by its absolute root path. It is used for both the
// cache file and the NuclaDB tenant, so two folders never share either.
func Key(root string) string {
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:8])
}

func cachePath(dir, root string) string {
	return filepath.Join(dir, Key(root)+".json")
}

// Open loads the cached state for root, or starts an empty library.
func Open(dir, root string) (*Library, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	l := &Library{Root: abs, Entries: map[string]*Entry{}}
	b, err := os.ReadFile(cachePath(dir, abs))
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, l); err != nil {
		return nil, fmt.Errorf("reading %s: %w", cachePath(dir, abs), err)
	}
	return l, nil
}

// Save writes the cache through a temp file and a rename, so an
// interrupted run never leaves a half-written file.
func (l *Library) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".lookalike-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), cachePath(dir, l.Root))
}

// Sorted returns the entries ordered by path.
func (l *Library) Sorted() []*Entry {
	out := make([]*Entry, 0, len(l.Entries))
	for _, e := range l.Entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Forget deletes the cache file for root, if there is one.
func Forget(dir, root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := os.Remove(cachePath(dir, abs)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
