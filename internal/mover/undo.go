package mover

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// UndoResult says what Undo did.
type UndoResult struct {
	Restored int
	Missing  []string // logged moves whose file isn't at the destination (never moved, or moved again since)
	Blocked  []string // original paths now taken by another file, left alone
}

// Undo moves every file in the log back where it came from, newest move
// first. It never overwrites: if something now sits at the original path,
// that file is left where it is and reported in Blocked.
func Undo(logPath string) (UndoResult, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return UndoResult{}, err
	}
	defer f.Close()
	var moves []Move
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		if sc.Text() == "" {
			continue
		}
		fromQ, toQ, ok := strings.Cut(sc.Text(), "\t")
		from, err1 := strconv.Unquote(fromQ)
		to, err2 := strconv.Unquote(toQ)
		if !ok || err1 != nil || err2 != nil {
			return UndoResult{}, fmt.Errorf("%s line %d: not a lookalike undo log", logPath, line)
		}
		moves = append(moves, Move{From: from, To: to})
	}
	if err := sc.Err(); err != nil {
		return UndoResult{}, err
	}

	var r UndoResult
	for i := len(moves) - 1; i >= 0; i-- {
		m := moves[i]
		if _, err := os.Lstat(m.To); errors.Is(err, fs.ErrNotExist) {
			r.Missing = append(r.Missing, m.To)
			continue
		}
		if _, err := os.Lstat(m.From); err == nil {
			r.Blocked = append(r.Blocked, m.From)
			continue
		}
		if err := moveFile(m.To, m.From); err != nil {
			return r, err
		}
		r.Restored++
	}
	return r, nil
}
