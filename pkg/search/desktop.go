package search

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func SearchDesktopApps(term string) []Result {
	toks := tokensFrom(term)
	paths := collectDesktopFiles()
	var out []Result
	seen := map[string]bool{}
	for _, p := range paths {
		info := parseDesktopFile(p)
		if info == nil {
			continue
		}
		if info["Type"] != "Application" {
			continue
		}
		if strings.EqualFold(info["NoDisplay"], "true") {
			continue
		}
		name := info["Name"]
		if name == "" {
			name = filepath.Base(p)
		}
		comment := info["Comment"]

		execCmd := sanitizeExecField(info["Exec"])
		ok, score := matchScore(toks, name, execCmd, comment, p)
		if !ok {
			continue
		}
		launch := execCmd
		if launch == "" {
			launch = name
		}
		key := strings.ToLower(launch + "|" + name)
		if seen[key] {
			continue
		}
		seen[key] = true

		iconPath := info["Icon"]
		out = append(out, Result{
			Name:    name,
			GUI:     true,
			Type:    "app",
			Comment: comment,
			Source:  fmt.Sprintf("%s|score:%d", p, score),
			Command: launch,
			Icon:    iconPath,
		})
	}
	// sort by score desc then name
	sort.Slice(out, func(i, j int) bool {
		si := extractScore(out[i].Source)
		sj := extractScore(out[j].Source)
		if si != sj {
			return si > sj
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func extractScore(src string) int {
	parts := strings.SplitSeq(src, "|")
	for p := range parts {
		if after, ok := strings.CutPrefix(p, "score:"); ok {
			n, _ := strconv.Atoi(after)
			return n
		}
	}
	return 0
}

func collectDesktopFiles() []string {
	var dirs []string
	xdg := os.Getenv("XDG_DATA_DIRS")
	if xdg == "" {
		xdg = "/usr/local/share:/usr/share"
	}
	parts := strings.Split(xdg, ":")
	home := homeDir()
	parts = append([]string{filepath.Join(home, ".local", "share")}, parts...)
	parts = append(parts, filepath.Join(home, ".nix-profile", "share"))
	if exists("/run/current-system/sw/share") {
		parts = append(parts, "/run/current-system/sw/share")
	}
	seen := map[string]bool{}
	for _, p := range parts {
		if p == "" {
			continue
		}
		if !seen[p] {
			seen[p] = true
			dirs = append(dirs, p)
		}
	}

	outCh := make(chan string, 512)
	var wg sync.WaitGroup
	for _, d := range dirs {
		appdir := filepath.Join(d, "applications")
		wg.Add(1)
		go func(ad string) {
			defer wg.Done()
			filepath.WalkDir(ad, func(path string, de fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if de.IsDir() {
					return nil
				}
				if strings.HasSuffix(path, ".desktop") {
					outCh <- path
				}
				return nil
			})
		}(appdir)
	}

	// include nix store scan
	if exists("/nix/store") {
		entries, _ := os.ReadDir("/nix/store")
		for _, e := range entries {
			appdir := filepath.Join("/nix/store", e.Name(), "share", "applications")
			wg.Add(1)
			go func(ad string) {
				defer wg.Done()
				filepath.WalkDir(ad, func(path string, de fs.DirEntry, err error) error {
					if err != nil {
						return nil
					}
					if de.IsDir() {
						return nil
					}
					if strings.HasSuffix(path, ".desktop") {
						outCh <- path
					}
					return nil
				})
			}(appdir)
		}
	}

	go func() {
		wg.Wait()
		close(outCh)
	}()

	uniq := map[string]bool{}
	var out []string
	for p := range outCh {
		if !uniq[p] {
			uniq[p] = true
			out = append(out, p)
		}
	}
	return out
}

func parseDesktopFile(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	in := false
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if line == "[Desktop Entry]" {
			in = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = false
			continue
		}
		if !in {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out
}
