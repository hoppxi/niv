package search

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func searchFilesMode(arg string) []Result {
	// arg may contain inline options :dir and :max followed by search term
	dir, max, rem := parseFileOptions(arg)
	if dir == "" {
		dir = homeDir()
	} else {
		// expand ~
		if strings.HasPrefix(dir, "~/") {
			dir = filepath.Join(homeDir(), strings.TrimPrefix(dir, "~/"))
		} else if dir == "~" {
			dir = homeDir()
		}
	}
	if max <= 0 {
		max = 20
	}
	term := strings.TrimSpace(rem)

	// walk concurrently but stop when max reached
	found := make(chan Result, max)
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	done := make(chan struct{})

	if !exists(dir) {
		return []Result{
			{
				Name:    fmt.Sprintf("Directory not found: %s", dir),
				GUI:     false,
				Type:    "file",
				Source:  dir,
				Comment: "check directory",
				Command: "",
			},
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
			select {
			case <-done:
				return errors.New("stopped")
			default:
			}
			if err != nil {
				return nil
			}

			base := de.Name()
			if base == ".git" || base == "node_modules" || strings.HasPrefix(base, ".cache") {
				if de.IsDir() {
					return filepath.SkipDir
				}
			}
			if de.IsDir() {
				return nil
			}
			name := filepath.Base(path)
			if term == "" || strings.Contains(strings.ToLower(name), strings.ToLower(term)) || strings.Contains(strings.ToLower(path), strings.ToLower(term)) {
				mu.Lock()
				if count >= max {
					mu.Unlock()
					close(done)
					return errors.New("max reached")
				}
				count++
				mu.Unlock()
				found <- Result{
					Name:    name,
					GUI:     false,
					Type:    "file",
					Source:  path,
					Command: fileOpenCommand(path),
					Comment: "opening" + path,
				}
			}
			return nil
		})
	}()

	go func() {
		wg.Wait()
		close(found)
	}()

	out := []Result{}
	for r := range found {
		out = append(out, r)
		if len(out) >= max {
			break
		}
	}
	// ensure unique by source
	uniq := map[string]Result{}
	for _, r := range out {
		uniq[r.Source] = r
	}
	final := []Result{}
	for _, v := range uniq {
		final = append(final, v)
	}
	// sort by name
	sort.Slice(final, func(i, j int) bool { return strings.ToLower(final[i].Name) < strings.ToLower(final[j].Name) })
	return final
}
