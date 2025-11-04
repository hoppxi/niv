package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Output struct {
	Desc string `json:"desc"`
	Plc  string `json:"plc"`
}

func fetchBibleVerse() (Output, error) {
	resp, err := http.Get("https://labs.bible.org/api/?passage=random&type=json")
	if err != nil {
		return Output{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Output{}, err
	}

	var verses []struct {
		Book string `json:"bookname"`
		Chapter string `json:"chapter"`
		Verse string `json:"verse"`
		Text string `json:"text"`
	}

	err = json.Unmarshal(body, &verses)
	if err != nil {
		return Output{}, err
	}

	if len(verses) == 0 {
		return Output{}, fmt.Errorf("no verse found")
	}

	place := fmt.Sprintf("%s %s:%s", verses[0].Book, verses[0].Chapter, verses[0].Verse)
	return Output{Desc: verses[0].Text, Plc: place}, nil
}

func fetchQuote() (Output, error) {
	resp, err := http.Get("https://api.quotable.io/random")
	if err != nil {
		return Output{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Output{}, err
	}

	var quote struct {
		Content string `json:"content"`
		Author  string `json:"author"`
	}

	err = json.Unmarshal(body, &quote)
	if err != nil {
		return Output{}, err
	}

	return Output{Desc: quote.Content, Plc: quote.Author}, nil
}

func main() {
	opt := flag.String("o", "quote", "output type: verse or quote")
	flag.Parse()

	var result Output
	var err error

	if *opt == "verse" {
		result, err = fetchBibleVerse()
	} else {
		result, err = fetchQuote()
	}

	if err != nil {
		fmt.Println(`{"desc": "error fetching content", "plc": ""}`)
		os.Exit(1)
	}

	jsonOutput, _ := json.Marshal(result)
	fmt.Println(string(jsonOutput))
}
