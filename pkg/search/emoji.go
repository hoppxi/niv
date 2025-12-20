package search

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed emoji.json
var emojiJSON []byte

var (
	emojiCache []EmojiItem
	emojiOnce  sync.Once
)

type EmojiItem struct {
	Emoji       string   `json:"emoji"`
	Unicode     string   `json:"unicode"`
	Text        string   `json:"text"`
	Category    string   `json:"category"`
	Subcategory string   `json:"subcategory"`
	Keywords    []string `json:"keywords"`
}

func parseEmojiQuery(rawTerm string) (term, category, subcategory string, copyUnicode bool) {
	toks := strings.Fields(rawTerm)

	term = rawTerm
	category = ""
	subcategory = ""
	copyUnicode = false

	var cleanedToks []string

	for i := 0; i < len(toks); i++ {
		t := strings.ToLower(toks[i])

		switch t {
		case ":c":
			if i+1 < len(toks) {
				category = strings.ToLower(toks[i+1])
				i++
			}
		case ":sc":
			if i+1 < len(toks) {
				subcategory = strings.ToLower(toks[i+1])
				i++
			}
		case ":u":
			copyUnicode = true
		default:
			cleanedToks = append(cleanedToks, toks[i])
		}
	}

	term = strings.ToLower(strings.Join(cleanedToks, " "))
	return
}

func searchEmojiMode(rawTerm string) []Result {
	emojiOnce.Do(func() {
		if err := json.Unmarshal(emojiJSON, &emojiCache); err != nil {
			fmt.Printf("Error parsing emoji.json: %v\n", err)
		}
	})

	term, categoryFilter, subcategoryFilter, copyUnicode := parseEmojiQuery(rawTerm)

	var results []Result
	limit := 20 // LIMIT OUTPUT TO 20

	for _, item := range emojiCache {
		match := true

		if categoryFilter != "" && strings.ToLower(item.Category) != categoryFilter {
			match = false
		}
		if subcategoryFilter != "" && strings.ToLower(item.Subcategory) != subcategoryFilter {
			match = false
		}

		if !match {
			continue
		}

		if term != "" {
			termMatch := false
			if strings.Contains(strings.ToLower(item.Text), term) {
				termMatch = true
			} else {
				for _, k := range item.Keywords {
					if strings.Contains(strings.ToLower(k), term) {
						termMatch = true
						break
					}
				}
			}
			match = termMatch // The final match depends on passing the text search
		}

		if match {
			contentToCopy := item.Emoji
			outputName := fmt.Sprintf("%s  %s", item.Emoji, item.Text)

			if copyUnicode {

				contentToCopy = item.Unicode
				outputName = fmt.Sprintf("U: %s  %s", item.Unicode, item.Text)
			}

			commandToCopy := fmt.Sprintf("echo -n \"%s\" | wl-copy", contentToCopy)

			results = append(results, Result{
				Name:    outputName,
				GUI:     false,
				Type:    "emoji",
				Source:  "internal",
				Command: commandToCopy, // Command to execute upon selection
				Icon:    item.Emoji,    // Icon is always the emoji character for visualization
				Comment: item.Unicode,
			})

			// STOP if we reached the limit
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}
