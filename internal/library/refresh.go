package library

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Rakshit-gen/lookalike/internal/photo"
)

// Changes is what Refresh found.
type Changes struct {
	Fingerprinted []*Entry         // new or modified photos
	Removed       []*Entry         // in the cache but no longer on disk
	Failed        map[string]error // photos that couldn't be decoded, by path
	Unchanged     int
}

// Refresh brings the library in line with the folder: photos that are new
// or whose size or modification time changed are decoded and
// fingerprinted, using every CPU, and photos that are gone are dropped.
// progress, if not nil, is called after each photo is decoded.
func (l *Library) Refresh(progress func(done, total int)) (Changes, error) {
	paths, err := Walk(l.Root)
	if err != nil {
		return Changes{}, err
	}
	ch := Changes{Failed: map[string]error{}}
	seen := make(map[string]bool, len(paths))
	var todo []*Entry
	for _, rel := range paths {
		seen[rel] = true
		st, err := os.Stat(filepath.Join(l.Root, filepath.FromSlash(rel)))
		if err != nil {
			ch.Failed[rel] = err
			continue
		}
		if e, ok := l.Entries[rel]; ok && e.Size == st.Size() && e.ModTime.Equal(st.ModTime()) {
			ch.Unchanged++
			continue
		}
		todo = append(todo, &Entry{Path: rel, Size: st.Size(), ModTime: st.ModTime()})
	}
	for rel, e := range l.Entries {
		if !seen[rel] {
			ch.Removed = append(ch.Removed, e)
			delete(l.Entries, rel)
		}
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		done int
		jobs = make(chan *Entry)
	)
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				info, err := photo.Load(filepath.Join(l.Root, filepath.FromSlash(e.Path)))
				mu.Lock()
				if err != nil {
					ch.Failed[e.Path] = err
					delete(l.Entries, e.Path) // a file that broke shouldn't keep its old fingerprint
				} else {
					e.Width, e.Height, e.Vector = info.Width, info.Height, info.Vector
					l.Entries[e.Path] = e
					ch.Fingerprinted = append(ch.Fingerprinted, e)
				}
				done++
				if progress != nil {
					progress(done, len(todo))
				}
				mu.Unlock()
			}
		}()
	}
	for _, e := range todo {
		jobs <- e
	}
	close(jobs)
	wg.Wait()
	return ch, nil
}
