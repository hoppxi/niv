package search

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

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
				Comment: "Install cliphist and wl-clipboard; entries will appear here.",
				Command: "",
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
			Comment: "Click to copy",
		})
	}

	return out
}
