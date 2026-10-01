package mover

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// Apply carries out the moves and records each one in the undo log at
// logPath. A move is written to the log and synced before the file is
// moved, so after a crash the log may name a move that didn't happen, but
// never misses one that did. Undo copes with the first case.
//
// It stops at the first error and returns how many moves completed.
func Apply(moves []Move, logPath string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return 0, err
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer log.Close()
	for i, m := range moves {
		if _, err := fmt.Fprintf(log, "%s\t%s\n", strconv.Quote(m.From), strconv.Quote(m.To)); err != nil {
			return i, err
		}
		if err := log.Sync(); err != nil {
			return i, err
		}
		if err := moveFile(m.From, m.To); err != nil {
			return i, err
		}
	}
	return len(moves), nil
}

// moveFile renames from to to, refusing to replace an existing file. When
// the two are on different disks, where rename can't work, it copies,
// syncs, and only then removes the original.
//
// ponytail: the exists check and the rename are two steps, so a file
// created at "to" in between would be replaced. Fine for one person
// running a CLI on their own folders.
func moveFile(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("%s already exists, not overwriting it", to)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	err := os.Rename(from, to)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyFile(from, to); err != nil {
		os.Remove(to)
		return err
	}
	return os.Remove(from)
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, st.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Chtimes(to, st.ModTime(), st.ModTime())
}
