package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func searchURLMode(link string) []Result {
	link = strings.TrimSpace(link)
	if link == "" {
		return []Result{{
			Name:    "No URL provided",
			GUI:     false,
			Type:    "url",
			Source:  "",
			Command: "",
			Comment: "Enter URL to open",
		}}
	}

	if _, err := url.ParseRequestURI(link); err != nil {
		return []Result{{
			Name:    "Invalid URL",
			GUI:     false,
			Type:    "url",
			Source:  link,
			Command: "",
			Comment: "Enter a valid URL to open",
		}}
	}

	cmd := fmt.Sprintf("xdg-open %s", shellEscape(link))
	return []Result{{
		Name:    link,
		GUI:     false,
		Type:    "url",
		Source:  link,
		Command: cmd,
		Comment: "Click to open url",
	}}
}

func searchGoogleMode(term string) []Result {
	shellEscape := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}

	term = strings.TrimSpace(term)
	if term == "" {
		return []Result{{
			Name:    "Open Google",
			GUI:     false,
			Type:    "google",
			Source:  "https://google.com",
			Command: "xdg-open https://google.com",
			Comment: "Click to open Google",
		}}
	}

	apiURL := "https://suggestqueries.google.com/complete/search?client=firefox&q=" + url.QueryEscape(term)
	resp, err := http.Get(apiURL)
	if err != nil {
		return []Result{{Name: "Error fetching suggestions: " + err.Error()}}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return []Result{{Name: "Error parsing suggestions: " + err.Error()}}
	}

	results := []Result{}
	if len(data) > 1 {
		if arr, ok := data[1].([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					u := "https://www.google.com/search?q=" + url.QueryEscape(s)
					results = append(results, Result{
						Name:    s,
						GUI:     false,
						Type:    "google",
						Source:  u,
						Command: "xdg-open " + shellEscape(u),
						Comment: "",
					})
				}
			}
		}
	}

	if len(results) == 0 {
		u := "https://www.google.com/search?q=" + url.QueryEscape(term)
		results = append(results, Result{
			Name:    term,
			GUI:     false,
			Type:    "google",
			Source:  u,
			Command: "xdg-open " + shellEscape(u),
		})
	}

	return results
}

func searchYouTubeMode(term string) []Result {
	term = strings.TrimSpace(term)
	if term == "" {
		return []Result{{
			Name:    "Open YouTube",
			GUI:     false,
			Type:    "youtube",
			Source:  "https://www.youtube.com",
			Command: "xdg-open https://www.youtube.com",
			Comment: "Click to open YouTube",
		}}
	}

	endpoint := "https://suggestqueries.google.com/complete/search?client=youtube&ds=yt&q=" + url.QueryEscape(term)
	resp, err := http.Get(endpoint)
	if err == nil {
		defer resp.Body.Close()
		if body, err := io.ReadAll(resp.Body); err == nil {
			raw := strings.TrimSpace(string(body))

			// remove wrapper: window.google.ac.h([...])
			if strings.HasPrefix(raw, "window.google.ac.h(") && strings.HasSuffix(raw, ")") {
				raw = strings.TrimPrefix(raw, "window.google.ac.h(")
				raw = strings.TrimSuffix(raw, ")")
			}

			var data []any
			if err := json.Unmarshal([]byte(raw), &data); err == nil {
				results := []Result{}
				if len(data) >= 2 {
					if arr, ok := data[1].([]any); ok {
						for _, v := range arr {
							if entry, ok := v.([]any); ok && len(entry) > 0 {
								if s, ok := entry[0].(string); ok {
									u := "https://www.youtube.com/results?search_query=" + url.QueryEscape(s)
									results = append(results, Result{
										Name:    s,
										GUI:     false,
										Type:    "youtube",
										Source:  u,
										Command: "xdg-open '" + strings.ReplaceAll(u, "'", "'\\''") + "'",
									})
								}
							}
						}
					}
				}
				if len(results) > 0 {
					return results
				}
			}
		}
	}

	// fallback if no suggestions or error occurred
	u := "https://www.youtube.com/results?search_query=" + url.QueryEscape(term)
	return []Result{{
		Name:    "Search YouTube for " + term,
		GUI:     false,
		Type:    "youtube",
		Source:  u,
		Command: "xdg-open '" + strings.ReplaceAll(u, "'", "'\\''") + "'",
	}}
}
