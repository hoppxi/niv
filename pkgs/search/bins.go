package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func searchBins(term string) []Result {
	toks := tokensFrom(term)
	paths := strings.Split(os.Getenv("PATH"), ":")
	home := homeDir()
	if home != "" {
		paths = append(paths, filepath.Join(home, ".nix-profile", "bin"))
	}
	paths = append(paths, "/run/current-system/sw/bin")
	seen := map[string]bool{}
	var out []Result

	// default terminal prefix for bins
	defaultTerminal := "alacritty --hold -e"

	for _, p := range paths {
		if p == "" {
			continue
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ok, _ := matchScore(toks, name, p)
			if !ok && term != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(term)) {
				continue
			}
			full := filepath.Join(p, name)
			if !isExecutable(full) {
				continue
			}
			if seen[full] {
				continue
			}
			seen[full] = true
			out = append(out, Result{
				Name:    name,
				GUI:     false,
				Type:    "bin",
				Source:  full,
				Command: fmt.Sprintf("%s %s", defaultTerminal, shellEscape(full)), // requires terminal
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}
