package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func searchTranslateMode(term string) []Result {
	shellEscape := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}

	term = strings.TrimSpace(term)
	if term == "" {
		return []Result{{
			Name:    "Enter text to translate",
			GUI:     false,
			Type:    "translate",
			Source:  "https://translate.google.com",
			Command: "xdg-open https://translate.google.com",
		}}
	}

	// Change target language here (e.g. "es" for Spanish, "fr" for French)
	targetLang := "en"

	// Unofficial Google Translate endpoint
	apiURL := "https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=" + targetLang + "&dt=t&q=" + url.QueryEscape(term)

	resp, err := http.Get(apiURL)
	if err != nil {
		return []Result{{Name: "Error fetching translation: " + err.Error()}}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return []Result{{Name: "Error parsing translation: " + err.Error()}}
	}

	translated := ""
	// JSON structure: [[[ "translated text", "original text", ... ]], ...]
	if arr, ok := data.([]any); ok && len(arr) > 0 {
		if inner, ok := arr[0].([]any); ok {
			for _, segment := range inner {
				if segArr, ok := segment.([]any); ok && len(segArr) > 0 {
					if s, ok := segArr[0].(string); ok {
						translated += s
					}
				}
			}
		}
	}

	results := []Result{}
	if translated != "" {
		u := "https://translate.google.com/?sl=auto&tl=" + targetLang + "&text=" + url.QueryEscape(term)
		results = append(results, Result{
			Name:    translated,
			GUI:     false,
			Type:    "translate",
			Source:  u,
			Command: "xdg-open " + shellEscape(u),
		})
	} else {
		results = append(results, Result{
			Name:    "No translation found",
			GUI:     false,
			Type:    "translate",
			Source:  "https://translate.google.com",
			Command: "xdg-open https://translate.google.com",
		})
	}

	return results
}
