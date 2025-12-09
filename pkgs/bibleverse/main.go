package bibleverse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Verse struct {
	Text string `json:"text"`
	Ref  string `json:"ref"`
}

func GetDailyVerse() (*Verse, error) {
	// Use NET Bible API — "votd" (verse of the day), JSON response
	url := "https://labs.bible.org/api/?passage=votd&type=json"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status from API: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var arr []struct {
		Bookname string `json:"bookname"`
		Chapter  string `json:"chapter"`
		Verse    string `json:"verse"`
		Text     string `json:"text"`
	}

	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, err
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("no verse returned")
	}

	v := arr[0]
	ref := fmt.Sprintf("%s %s:%s", v.Bookname, v.Chapter, v.Verse)

	return &Verse{
		Text: v.Text,
		Ref:  ref,
	}, nil
}

func GetDailyVerseJSON() ([]byte, error) {
	v, err := GetDailyVerse()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}
