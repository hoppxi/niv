package search

import (
	"fmt"
	"regexp"
	"strings"
)

type Result struct {
	Name    string `json:"name"`
	GUI     bool   `json:"gui"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Command string `json:"command"`
	Icon    string `json:"icon,omitempty"` // NEW
}

var placeholderRe = regexp.MustCompile(`%[fFuUdDnNickvm]`)

func Search(args []string) {

	term := ""
	if len(args) > 0 {
		term = strings.TrimSpace(strings.Join(args, " "))
	}
	if term == "" {
		printJSON(helpJSON(""))
		return
	}

	toks := strings.Fields(term)
	if len(toks) > 0 && strings.HasPrefix(toks[0], ":") {
		mode := strings.ToLower(toks[0])
		// aliases
		switch mode {
		case ":help":
			printJSON(helpJSON(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":h":
			printJSON(helpJSON(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":bin", ":bins":
			printJSON(searchBins(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":clipboard", ":clip":
			printJSON(searchClipboardMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":google", ":g":
			printJSON(searchGoogleMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":youtube", ":yt":
			printJSON(searchYouTubeMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":acc", ":accs":
			printJSON(searchAccMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":cal", ":calc":
			printJSON(searchCalcMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":cmd", ":sh":
			printJSON(searchCmdMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":translate", ":ts":
			printJSON(searchTranslateMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":url", ":u":
			printJSON(searchURLMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":files", ":file", ":f":
			printJSON(searchFilesMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		// additional convenient filters
		case ":music":
			printJSON(searchFilesMode(":dir ~/Music " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":pictures", ":pics", ":images":
			printJSON(searchFilesMode(":dir ~/Pictures " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":videos":
			printJSON(searchFilesMode(":dir ~/Videos " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":docs", ":documents":
			printJSON(searchFilesMode(":dir ~/Documents " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":configs", ":config":
			printJSON(searchFilesMode(":dir ~/.config " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":notes":
			printJSON(searchFilesMode(":dir ~/Notes " + strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		default:
			// unknown mode -> fallback to help
			printJSON(helpJSON(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		}
	}

	// default search: apps first; if none found, search bins
	out := defaultSearch(term)
	printJSON(out)
}

func runHelp(cmd string) string {
	out, err := runCapture(cmd, "--help")
	if err != nil || out == "" {
		return fmt.Sprintf("Help unavailable for %s", cmd)
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 20 { // truncate long help
		return strings.Join(lines[:5], "\n") + "\n…"
	}
	return out
}

func helpJSON(term string) []Result {
	helpItems := []Result{
		{Name: "Unified Search", GUI: false, Type: "help", Source: "internal", Command: ":search"},
		{Name: ":help or :h", GUI: false, Type: "help", Source: "internal", Command: ":help"},
		{Name: ":bin <term> or :bins <term>", GUI: false, Type: "help", Source: "PATH", Command: ":bin"},
		{Name: ":clipboard <term> or :clip <term>", GUI: false, Type: "help", Source: "cliphist or wl-paste", Command: ":clipboard"},
		{Name: ":google <term> or :g <term>", GUI: false, Type: "help", Source: "Google Suggest API", Command: ":google"},
		{Name: ":youtube <term> or :yt <term>", GUI: false, Type: "help", Source: "YouTube Suggest API", Command: ":youtube"},
		{Name: ":translate <term> or :ts <term>", GUI: false, Type: "help", Source: "Google Translate API", Command: ":translate"},
		{Name: ":url <url> or :u <url>", GUI: false, Type: "help", Source: "system", Command: ":url"},
		{Name: ":files :dir '<dir>' :max <n> <term> or :f <term>", GUI: false, Type: "help", Source: "filesystem", Command: ":files"},
		{Name: ":music", GUI: false, Type: "help", Source: "filesystem", Command: ":music"},
		{Name: ":pictures or :pics or :images", GUI: false, Type: "help", Source: "filesystem", Command: ":pictures"},
		{Name: ":videos", GUI: false, Type: "help", Source: "filesystem", Command: ":videos"},
		{Name: ":docs or :documents", GUI: false, Type: "help", Source: "filesystem", Command: ":docs"},
		{Name: ":configs or :config", GUI: false, Type: "help", Source: "filesystem", Command: ":configs"},
		{Name: ":notes", GUI: false, Type: "help", Source: "filesystem", Command: ":notes"},
		{Name: ":cmd <command> or :sh <command>", GUI: false, Type: "help", Source: "system", Command: ":cmd"},
		{Name: ":acc <term> or :accs <term>", GUI: false, Type: "help", Source: "internal", Command: ":acc"},
		{Name: ":calc <expression> or :cal <expression>", GUI: false, Type: "help", Source: "internal", Command: ":calc"},
	}

	if term == "" {
		return helpItems
	}

	term = strings.ToLower(term)
	var filtered []Result
	for _, item := range helpItems {
		if strings.Contains(strings.ToLower(item.Name), term) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func defaultSearch(term string) []Result {
	apps := SearchDesktopApps(term)
	if len(apps) > 0 {
		return apps
	}
	// fallback to bins
	bins := searchBins(term)
	return bins
}
