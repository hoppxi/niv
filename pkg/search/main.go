package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Result struct {
	Name    string `json:"name"`
	GUI     bool   `json:"gui"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Command string `json:"command"`
	Icon    string `json:"icon,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type ExtConfig struct {
	Name         string `mapstructure:"name"`
	Trigger      string `mapstructure:"trigger"`
	TriggerShort string `mapstructure:"trigger_short"`
	Path         string `mapstructure:"path"`
	FromFolder   string `mapstructure:"from_folder"`
	ExcludeType  string `mapstructure:"exclude_type"`
	OnSelect     string `mapstructure:"on_select"`
	HelpText     string `mapstructure:"help_text"`
	Limit        int    `mapstructure:"limit"`
}

var placeholderRe = regexp.MustCompile(`%[fFuUdDnNickvm]`)

func Search(args []string, v *viper.Viper, configKey string) {

	term := ""
	if len(args) > 0 {
		term = strings.TrimSpace(strings.Join(args, " "))
	}

	var extensions []ExtConfig
	v.UnmarshalKey(configKey, &extensions)

	if term == "" {
		printJSON(helpJSON("", extensions))
		return
	}

	toks := strings.Fields(term)
	if len(toks) > 0 && strings.HasPrefix(toks[0], ":") {
		mode := strings.ToLower(toks[0])
		query := strings.TrimSpace(strings.TrimPrefix(term, toks[0]))

		// aliases
		switch mode {
		case ":help", ":h":
			printJSON(helpJSON(query, extensions))
			return
		case ":bin", ":bins":
			printJSON(searchBins(query))
			return
		case ":clipboard", ":clip":
			printJSON(searchClipboardMode(query))
			return
		case ":google", ":g":
			printJSON(searchGoogleMode(query))
			return
		case ":youtube", ":yt":
			printJSON(searchYouTubeMode(query))
			return
		case ":cal", ":calc":
			printJSON(searchCalcMode(query))
			return
		case ":emoji", ":e":
			printJSON(searchEmojiMode(query))
			return
		case ":cmd", ":sh":
			printJSON(searchCmdMode(query))
			return
		case ":translate", ":ts":
			printJSON(searchTranslateMode(query))
			return
		case ":url", ":u":
			printJSON(searchURLMode(query))
			return
		case ":files", ":file", ":f":
			printJSON(searchFilesMode(query))
			return
		// additional convenient filters
		case ":music":
			printJSON(searchFilesMode(":dir ~/Music " + query))
			return
		case ":pictures", ":pics", ":images":
			printJSON(searchFilesMode(":dir ~/Pictures " + query))
			return
		case ":videos":
			printJSON(searchFilesMode(":dir ~/Videos " + query))
			return
		case ":docs", ":documents":
			printJSON(searchFilesMode(":dir ~/Documents " + query))
			return
		case ":configs", ":config":
			printJSON(searchFilesMode(":dir ~/.config " + query))
			return
		case ":notes":
			printJSON(searchFilesMode(":dir ~/Notes " + query))
			return
		}

		for _, ext := range extensions {
			if mode == ext.Trigger || (ext.TriggerShort != "" && mode == ext.TriggerShort) {
				var results []Result
				if ext.FromFolder != "" {
					results = handleFolderExtension(ext, query)
				} else if ext.Path != "" {
					results = handleScriptExtension(ext, query)
				}

				// Apply Limit if set
				if ext.Limit > 0 && len(results) > ext.Limit {
					results = results[:ext.Limit]
				}

				printJSON(results)
				return
			}
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

func helpJSON(term string, extensions []ExtConfig) []Result {
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
		{Name: ":calc <expression> or :cal <expression>", GUI: false, Type: "help", Source: "internal", Command: ":calc"},
		{Name: ":emoji <term> [:c <category>] [:sc <subcategory>] [:u]", GUI: false, Type: "help", Source: "internal", Command: ":emoji"},
	}

	for _, ext := range extensions {
		triggerDisplay := ext.Trigger
		if ext.TriggerShort != "" {
			triggerDisplay = fmt.Sprintf("%s (%s)", ext.Trigger, ext.TriggerShort)
		}

		helpItems = append(helpItems, Result{
			Name:    triggerDisplay,
			GUI:     false,
			Type:    "help",
			Source:  ext.Name,
			Command: ext.HelpText,
		})
	}

	if term == "" {
		return helpItems
	}

	if term == "" {
		return helpItems
	}

	term = strings.ToLower(term)
	var filtered []Result
	for _, item := range helpItems {
		if strings.Contains(strings.ToLower(item.Name), term) ||
			strings.Contains(strings.ToLower(item.Source), term) ||
			strings.Contains(strings.ToLower(item.Command), term) {
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
