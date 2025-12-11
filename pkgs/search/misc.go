package search

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

var embeddedAcc = []map[string]string{
	{"name": "Panel", "id": "panel"},
	{"name": "Power menu", "id": "power"},
	{"name": "Media Control", "id": "media"},
	{"name": "System monitor", "id": "system-monitor"},
	{"name": "Wallpapers", "id": "wallpaper"},
	{"name": "Notification", "id": "notification"},
}

func searchAccMode(term string) []Result {
	term = strings.TrimSpace(term)
	out := []Result{}
	for _, a := range embeddedAcc {
		name := a["name"]
		id := a["id"]
		if term != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(term)) &&
			!strings.Contains(strings.ToLower(id), strings.ToLower(term)) {
			continue
		}
		out = append(out, Result{
			Name:    name,
			GUI:     false,
			Type:    "acc",
			Source:  id,
			Command: fmt.Sprintf("~/.config/eww/bin/niv-ws open %s", shellEscape(id)),
		})
	}
	return out
}

func searchClipboardMode(term string) []Result {
	term = strings.TrimSpace(term)
	var entries []string
	var ids []string

	// Only use cliphist if available
	if path, _ := exec.LookPath("cliphist"); path != "" {
		out, err := runCapture(path, "list")
		if err == nil && out != "" {
			sc := bufio.NewScanner(strings.NewReader(out))
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" {
					continue
				}
				// cliphist list returns: ID \t content
				parts := strings.SplitN(line, "\t", 2)
				if len(parts) == 2 {
					ids = append(ids, parts[0])
					entries = append(entries, parts[1])
				}
			}
		}
	}

	if len(entries) == 0 {
		return []Result{
			{
				Name:    "No clipboard history found (cliphist not available or returned no entries).",
				GUI:     false,
				Type:    "clipboard",
				Source:  "system",
				Command: "Install cliphist and wl-clipboard; entries will appear here.",
			},
		}
	}

	toks := tokensFrom(term)
	out := []Result{}

	for i, e := range entries {
		preview := strings.TrimSpace(e)
		if preview == "" {
			continue
		}

		ok, _ := matchScore(toks, preview)
		if !ok {
			continue
		}

		// Use cliphist decode with wl-copy
		cmd := ""
		if len(ids) > i {
			cmd = fmt.Sprintf("cliphist decode %s | wl-copy", ids[i])
		}

		out = append(out, Result{
			Name:    truncateString(preview, 160),
			GUI:     false,
			Type:    "clipboard",
			Source:  fmt.Sprintf("clipboard:%s", ids[i]),
			Command: cmd,
		})
	}

	return out
}
