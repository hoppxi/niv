package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Result struct {
	Name    string `json:"name"`
	GUI     bool   `json:"gui"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Command string `json:"command"`
	Icon    string `json:"icon,omitempty"` // NEW
}


func main() {
	args := os.Args[1:]
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
		case ":wallpapers", ":wallpaper", ":wp":
			printJSON(searchWallpapersMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":workspaces", ":workspace", ":ws":
			printJSON(searchWorkspacesMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
			return
		case ":bookmarks", ":bookmark", ":bm":
			printJSON(searchBookmarksMode(strings.TrimSpace(strings.TrimPrefix(term, toks[0]))))
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


func printJSON(arr []Result) {
	enc, _ := json.MarshalIndent(arr, "", "  ")
	fmt.Println(string(enc))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func homeDir() string {
	h := os.Getenv("HOME")
	if h == "" {
		if runtime.GOOS == "windows" {
			return os.Getenv("USERPROFILE")
		}
		return "/"
	}
	return h
}

func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func runCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func tokensFrom(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// split on whitespace, also trim quotes
	f := regexp.MustCompile(`\s+`).Split(s, -1)
	out := []string{}
	for _, p := range f {
		p = strings.Trim(p, `"'`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// matchScore returns (matched,score) where higher score is better
// simple algorithm: require all tokens to appear (AND); score is -sum(positions) so earlier positions -> higher score
func matchScore(tokens []string, fields ...string) (bool, int) {
	if len(tokens) == 0 {
		return true, 0
	}
	combined := strings.ToLower(strings.Join(fields, " "))
	score := 0
	for _, t := range tokens {
		if t == "" {
			continue
		}
		idx := strings.Index(combined, strings.ToLower(t))
		if idx == -1 {
			return false, 0
		}
		score += (100000 - idx) // earlier -> higher contribution
	}
	return true, score
}

// safely get the --help output for a command
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



// help
func helpJSON(term string) []Result {
	helpItems := []Result{
		{Name: "Unified Search", GUI: false, Type: "help", Source: "internal", Command: ":search"},
		{Name: ":help or :h", GUI: false, Type: "help", Source: "internal", Command: ":help"},
		{Name: ":bin <term> or :bins <term>", GUI: false, Type: "help", Source: "PATH", Command: ":bin"},
		{Name: ":clipboard <term> or :clip <term>", GUI: false, Type: "help", Source: "cliphist or wl-paste", Command: ":clipboard"},
		{Name: ":wallpapers <term> or :wp <term>", GUI: false, Type: "help", Source: "~/.config/eww/config/wallpapers.yuck", Command: ":wallpapers"},
		{Name: ":workspaces <term> or :ws <term>", GUI: false, Type: "help", Source: "~/.config/eww/config/workspaces.yuck", Command: ":workspaces"},
		{Name: ":bookmarks <term> or :bm <term>", GUI: false, Type: "help", Source: "~/.config/eww/config/bookmarks.yuck", Command: ":bookmarks"},
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



// collect desktop files
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

var placeholderRe = regexp.MustCompile(`%[fFuUdDnNickvm]`)

func sanitizeExecField(execLine string) string {
	if execLine == "" {
		return ""
	}
	execLine = placeholderRe.ReplaceAllString(execLine, "")
	execLine = strings.TrimSpace(execLine)
	parts := shellSplit(execLine)
	if len(parts) == 0 {
		return execLine
	}
	return parts[0]
}

// quick shell split
func shellSplit(s string) []string {
	var out []string
	cur := ""
	inq := rune(0)
	esc := false
	for _, r := range s {
		switch {
		case esc:
			cur += string(r)
			esc = false
		case r == '\\':
			esc = true
		case r == '\'' || r == '"':
			if inq == 0 {
				inq = r
			} else if inq == r {
				inq = 0
			} else {
				cur += string(r)
			}
		case r == ' ' && inq == 0:
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		default:
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return !mode.IsDir() && mode&0111 != 0
}

// search desktop apps for a token; return top matches
func searchDesktopApps(term string) []Result {
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
		execCmd := sanitizeExecField(info["Exec"])
		ok, score := matchScore(toks, name, execCmd, info["Comment"], p)
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
		if after, ok :=strings.CutPrefix(p, "score:"); ok  {
			n, _ := strconv.Atoi(after)
			return n
		}
	}
	return 0
}


// searchBins
func searchBins(term string) []Result {
	toks := tokensFrom(term)
	paths := strings.Split(os.Getenv("PATH"), ":")
	home := homeDir()
	if home != "" {
		paths = append(paths, filepath.Join(home, ".nix-profile", "bin"))
	}
	paths = append(paths, "/run/current-system/sw/bin")
	seen := map[string]bool{}
	var out []Result

	// default terminal prefix for bins
	defaultTerminal := "alacritty -e"

	for _, p := range paths {
		if p == "" {
			continue
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ok, _ := matchScore(toks, name, p)
			if !ok && term != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(term)) {
				continue
			}
			full := filepath.Join(p, name)
			if !isExecutable(full) {
				continue
			}
			if seen[full] {
				continue
			}
			seen[full] = true
			out = append(out, Result{
				Name:    name,
				GUI:     false,
				Type:    "bin",
				Source:  full,
				Command: fmt.Sprintf("%s %s", defaultTerminal, shellEscape(full)), // requires terminal
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// default search will search for apps first, when none found fall back to bins
func defaultSearch(term string) []Result {
	apps := searchDesktopApps(term)
	if len(apps) > 0 {
		return apps
	}
	// fallback to bins
	bins := searchBins(term)
	return bins
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

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// :wallpapers
// read wallpapers yuck config file and return array of paths
func readWallpapersFromFile(path string) ([]string, error) {
	if !exists(path) {
		return nil, errors.New("file not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	txt := string(data)
	// remove ; comments
	reComment := regexp.MustCompile(`;[^\n\r]*`)
	txt = reComment.ReplaceAllString(txt, "")

	start := strings.Index(txt, "`[")
	if start >= 0 {
		rest := txt[start+2:]
		endRel := strings.Index(rest, "]`")
		if endRel > 0 {
			body := rest[:endRel]
			parts := regexp.MustCompile(`,\s*`).Split(body, -1)
			out := []string{}
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"'`)
				if p != "" {
					if strings.HasPrefix(p, "~/") {
						p = filepath.Join(homeDir(), strings.TrimPrefix(p, "~/"))
					}
					out = append(out, p)
				}
			}
			return out, nil
		}
	}

	defIdx := strings.Index(txt, "CONFIG_WALLPAPERS_DATA")
	if defIdx >= 0 {
		brackIdx := strings.Index(txt[defIdx:], "[")
		if brackIdx >= 0 {
			rest := txt[defIdx+brackIdx:]
			end := strings.Index(rest, "]")
			if end > 0 {
				body := rest[1:end]
				parts := regexp.MustCompile(`,\s*`).Split(body, -1)
				out := []string{}
				for _, p := range parts {
					p = strings.TrimSpace(p)
					p = strings.Trim(p, `"'`)
					if p != "" {
						if after, ok :=strings.CutPrefix(p, "~/"); ok  {
							p = filepath.Join(homeDir(), after)
						}
						out = append(out, p)
					}
				}
				return out, nil
			}
		}
	}

	return nil, errors.New("no wallpapers array found")
}


func searchWallpapersMode(term string) []Result {
	home := homeDir()
	conf := filepath.Join(home, ".config", "eww", "config", "wallpapers.yuck")
	paths, err := readWallpapersFromFile(conf)
	if err != nil {
		return []Result{
			{
				Name:    "No wallpapers config parsed",
				GUI:     false,
				Type:    "wallpaper",
				Source:  conf,
				Command: "Ensure file exists and has defvar CONFIG_WALLPAPERS_DATA `[...]`",
			},
		}
	}
	toks := tokensFrom(term)
	out := []Result{}
	for _, p := range paths {
		if p == "" {
			continue
		}
		ok, _ := matchScore(toks, filepath.Base(p), p)
		if !ok && term != "" && !strings.Contains(strings.ToLower(p), strings.ToLower(term)) {
			continue
		}
		// prefer swww, else feh, else xdg-open to just show
		cmd := ""
		if pth, _ := exec.LookPath("swww"); pth != "" {
			cmd = fmt.Sprintf("swww img %s", shellEscape(p))
		} else if pth, _ := exec.LookPath("feh"); pth != "" {
			cmd = fmt.Sprintf("feh --bg-scale %s", shellEscape(p))
		} else {
			cmd = "xdg-open " + shellEscape(p)
		}
		out = append(out, Result{
			Name:    p,
			GUI:     false,
			Type:    "wallpaper",
			Source:  p,
			Command: cmd,
		})
	}

	// If no matches and term non-empty, return the full list
	if len(out) == 0 && term != "" {
		full := []Result{}
		for _, p := range paths {
			cmd := ""
			if pth, _ := exec.LookPath("swww"); pth != "" {
				cmd = fmt.Sprintf("swww img %s", shellEscape(p))
			} else if pth, _ := exec.LookPath("feh"); pth != "" {
				cmd = fmt.Sprintf("feh --bg-scale %s", shellEscape(p))
			} else {
				cmd = "xdg-open " + shellEscape(p)
			}
			full = append(full, Result{
				Name:    p,
				GUI:     false,
				Type:    "wallpaper",
				Source:  p,
				Command: cmd,
			})
		}
		return full
	}

	return out
}

// :workspaces
func readDefvarJSON(path, varname string) ([]byte, error) {
	if !exists(path) {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	txt := string(data)
	// remove ; comments
	reComment := regexp.MustCompile(`;[^\n\r]*`)
	txt = reComment.ReplaceAllString(txt, "")

	// Try pattern: defvar VAR `[...]`
	startTag := fmt.Sprintf("defvar %s `[", varname)
	start := strings.Index(txt, startTag)
	if start >= 0 {
		rest := txt[start+len(startTag):]
		endRel := strings.Index(rest, "]`")
		if endRel > 0 {
			body := rest[:endRel]
			body = strings.TrimSpace(body)
			return []byte("[" + body + "]"), nil
		}
	}

	// Fallback: just find the first `[ ... ]` block (like wallpaper)
	start = strings.Index(txt, "`[")
	if start >= 0 {
		rest := txt[start+2:]
		endRel := strings.Index(rest, "]`")
		if endRel > 0 {
			body := rest[:endRel]
			body = strings.TrimSpace(body)
			return []byte("[" + body + "]"), nil
		}
	}

	return nil, errors.New("no defvar JSON block found")
}



func searchWorkspacesMode(term string) []Result {
	home := homeDir()
	conf := filepath.Join(home, ".config", "eww", "config", "workspaces.yuck")
	raw, err := readDefvarJSON(conf, "CONFIG_WORKSPACES_DATA")
	if err != nil {
		return []Result{
			{
				Name:    "No workspaces config found",
				GUI:     false,
				Type:    "workspace",
				Source:  conf,
				Command: "Ensure CONFIG_WORKSPACES_DATA exists in eww config",
			},
		}
	}
	
	clean := bytes.Trim(raw, "` \n\t")
	clean = bytes.ReplaceAll(clean, []byte(`\'`), []byte(`'`))

	var arr []map[string]any
	if err := json.Unmarshal(clean, &arr); err != nil {
		return []Result{
			{
				Name:    "Failed to parse workspaces JSON",
				GUI:     false,
				Type:    "workspace",
				Source:  conf,
				Command: fmt.Sprintf("parsing error: %v", err),
			},
		}
	}



	out := []Result{}
	toks := tokensFrom(term)
	for i, obj := range arr {
		title := toString(obj["title"])
		icon := toString(obj["icon"])
		cmd := toString(obj["cmd"])
		ok, _ := matchScore(toks, title, icon, cmd)
		if !ok && term != "" {
			continue
		}
		name := title
		if name == "" {
			name = fmt.Sprintf("workspace-%d", i+1)
		}
		command := cmd
		if command == "" {
			command = fmt.Sprintf("echo 'Workspace: %s' ", shellEscape(name))
		}
		out = append(out, Result{
			Name:    name,
			GUI:     false,
			Type:    "workspace",
			Source:  fmt.Sprintf("%s#%d", conf, i),
			Command: command,
			Icon:    icon,
		})
	}
	return out
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// :bookmarks
func searchBookmarksMode(term string) []Result {
	home := homeDir()
	conf := filepath.Join(home, ".config", "eww", "config", "bookmarks.yuck")
	raw, err := readDefvarJSON(conf, "CONFIG_BOOKMARKS_DATA")
	if err != nil {
		alt := filepath.Join(home, ".config", "bookmarks.yuck")
		raw, err = readDefvarJSON(alt, "CONFIG_BOOKMARKS_DATA")
		if err != nil {
			return []Result{
				{
					Name:    "No bookmarks config found",
					GUI:     false,
					Type:    "bookmark",
					Source:  conf,
					Command: "create CONFIG_BOOKMARKS_DATA in eww config",
				},
			}
		}
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err != nil {
		fixed := bytes.ReplaceAll(raw, []byte("`"), []byte(""))
		fixed = bytes.ReplaceAll(fixed, []byte(`'`), []byte(`"`))
		if err2 := json.Unmarshal(fixed, &arr); err2 != nil {
			return []Result{
				{
					Name:    "Failed to parse bookmarks JSON",
					GUI:     false,
					Type:    "bookmark",
					Source:  conf,
					Command: "parsing error",
				},
			}
		}
	}
	out := []Result{}
	toks := tokensFrom(term)
	for _, obj := range arr {
		title := toString(obj["title"])
		icon := toString(obj["icon"])
		urls := toString(obj["url"])
		ok, _ := matchScore(toks, title, icon, urls)
		if !ok && term != "" {
			continue
		}
		cmd := "xdg-open " + shellEscape(urls)
		out = append(out, Result{
			Name:    title,
			GUI:     false,
			Type:    "bookmark",
			Source:  urls,
			Command: cmd,
			Icon:    icon,
		})
	}
	return out
}

// :google / :youtube

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


// :translate
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



// :accessories
var embeddedAcc = []map[string]string{
	{"name": "Screenshot", "id": "screenshot-utils"},
	{"name": "Power menu", "id": "wlogout"},
	{"name": "Media Control", "id": "media-control"},
	{"name": "System monitor", "id": "system-monitor"},
	{"name": "Wallpapers", "id": "wallpaper"},
	{"name": "Planner", "id": "planner"},
	{"name": "notification", "id": "notification"},
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

// :calc
func searchCalcMode(expr string) []Result {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return []Result{
			{
				Name:    "Calculator",
				GUI:     false,
				Type:    "calc",
				Source:  "internal",
				Command: "Usage: :calc 1+2*3",
			},
		}
	}

	allowed := regexp.MustCompile(`^[0-9+\-*/%^().\s]+$`)
	if !allowed.MatchString(expr) {
		expr = regexp.MustCompile(`[^0-9+\-*/%^().\s]`).ReplaceAllString(expr, "")
		if expr == "" {
			return []Result{
				{
					Name:    "Expression contained invalid characters",
					GUI:     false,
					Type:    "calc",
					Source:  "internal",
					Command: "Invalid expression",
				},
			}
		}
	}

	result, err := evalExpression(expr)
	if err != nil {
		return []Result{
			{
				Name:    "Error evaluating expression: " + err.Error(),
				GUI:     false,
				Type:    "calc",
				Source:  "internal",
				Command: expr,
			},
		}
	}

	return []Result{
		{
			Name:    fmt.Sprintf("%v = %v", expr, result),
			GUI:     false,
			Type:    "calc",
			Source:  "internal",
			Command: "echo " + strconv.FormatFloat(result, 'f', -1, 64),
		},
	}
}

func evalExpression(expr string) (float64, error) {
	tokens := tokenize(expr)
	if len(tokens) == 0 {
		return 0, fmt.Errorf("empty expression")
	}
	return parseExpr(tokens)
}

// Tokenize input into numbers, operators, parentheses
func tokenize(expr string) []string {
	var tokens []string
	var buf strings.Builder
	for _, r := range expr {
		switch {
		case r >= '0' && r <= '9' || r == '.':
			buf.WriteRune(r)
		case strings.ContainsRune("+-*/%^()", r):
			if buf.Len() > 0 {
				tokens = append(tokens, buf.String())
				buf.Reset()
			}
			tokens = append(tokens, string(r))
		case r == ' ' || r == '\t':
			if buf.Len() > 0 {
				tokens = append(tokens, buf.String())
				buf.Reset()
			}
		}
	}
	if buf.Len() > 0 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}

// Recursive descent parser for + - * / % ^ and parentheses
func parseExpr(tokens []string) (float64, error) {
	var pos int

	var parsePrimary func() (float64, error)
	var parseFactor func() (float64, error)
	var parseTerm func() (float64, error)
	var parseSum func() (float64, error)

	parsePrimary = func() (float64, error) {
		if pos >= len(tokens) {
			return 0, fmt.Errorf("unexpected end of expression")
		}
		tok := tokens[pos]
		if tok == "(" {
			pos++
			val, err := parseSum()
			if err != nil {
				return 0, err
			}
			if pos >= len(tokens) || tokens[pos] != ")" {
				return 0, fmt.Errorf("expected ')'")
			}
			pos++
			return val, nil
		}
		pos++
		return strconv.ParseFloat(tok, 64)
	}

	parseFactor = func() (float64, error) {
		val, err := parsePrimary()
		if err != nil {
			return 0, err
		}
		for pos < len(tokens) && tokens[pos] == "^" {
			pos++
			right, err := parsePrimary()
			if err != nil {
				return 0, err
			}
			val = math.Pow(val, right)
		}
		return val, nil
	}

	parseTerm = func() (float64, error) {
		val, err := parseFactor()
		if err != nil {
			return 0, err
		}
		for pos < len(tokens) && (tokens[pos] == "*" || tokens[pos] == "/" || tokens[pos] == "%") {
			op := tokens[pos]
			pos++
			right, err := parseFactor()
			if err != nil {
				return 0, err
			}
			switch op {
			case "*":
				val *= right
			case "/":
				val /= right
			case "%":
				val = math.Mod(val, right)
			}
		}
		return val, nil
	}

	parseSum = func() (float64, error) {
		val, err := parseTerm()
		if err != nil {
			return 0, err
		}
		for pos < len(tokens) && (tokens[pos] == "+" || tokens[pos] == "-") {
			op := tokens[pos]
			pos++
			right, err := parseTerm()
			if err != nil {
				return 0, err
			}
			if op == "+" {
				val += right
			} else {
				val -= right
			}
		}
		return val, nil
	}

	res, err := parseSum()
	if err != nil {
		return 0, err
	}
	if pos != len(tokens) {
		return 0, fmt.Errorf("unexpected token: %s", tokens[pos])
	}
	return res, nil
}


// :files with :dir and :max parsing
// parse inline options such as :dir 'x' and :max N
func parseFileOptions(arg string) (dir string, max int, remainder string) {
	dir = ""
	max = 20 // default
	// tokenise keeping quoted strings as single tokens
	toks := tokenizeKeepingQuotes(arg)
	rem := []string{}
	i := 0
	for i < len(toks) {
		t := toks[i]
		if t == ":dir" && i+1 < len(toks) {
			dir = strings.Trim(toks[i+1], `"'`)
			i += 2
			continue
		}
		if t == ":max" && i+1 < len(toks) {
			n, err := strconv.Atoi(strings.Trim(toks[i+1], `"'`))
			if err == nil && n > 0 {
				max = n
			}
			i += 2
			continue
		}
		// allow :dir= and :max=
		if strings.HasPrefix(t, ":dir=") {
			dir = strings.TrimPrefix(t, ":dir=")
			dir = strings.Trim(dir, `"'`)
			i++
			continue
		}
		if strings.HasPrefix(t, ":max=") {
			nstr := strings.TrimPrefix(t, ":max=")
			n, err := strconv.Atoi(nstr)
			if err == nil && n > 0 {
				max = n
			}
			i++
			continue
		}
		// otherwise it's part of remainder
		rem = append(rem, t)
		i++
	}
	remainder = strings.TrimSpace(strings.Join(rem, " "))
	return
}

// simple tokenizer that keeps quoted substrings intact
func tokenizeKeepingQuotes(s string) []string {
	out := []string{}
	cur := ""
	inq := rune(0)
	esc := false
	for _, r := range s {
		switch {
		case esc:
			cur += string(r)
			esc = false
		case r == '\\':
			esc = true
		case r == '"' || r == '\'':
			if inq == 0 {
				inq = r
				cur += string(r)
			} else if inq == r {
				cur += string(r)
				out = append(out, strings.TrimSpace(cur))
				cur = ""
				inq = 0
			} else {
				cur += string(r)
			}
		case r == ' ' && inq == 0:
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		default:
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func searchFilesMode(arg string) []Result {
	// arg may contain inline options :dir and :max followed by search term
	dir, max, rem := parseFileOptions(arg)
	if dir == "" {
		dir = homeDir()
	} else {
		// expand ~
		if strings.HasPrefix(dir, "~/") {
			dir = filepath.Join(homeDir(), strings.TrimPrefix(dir, "~/"))
		} else if dir == "~" {
			dir = homeDir()
		}
	}
	if max <= 0 {
		max = 20
	}
	term := strings.TrimSpace(rem)

	// walk concurrently but stop when max reached
	found := make(chan Result, max)
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	done := make(chan struct{})

	if !exists(dir) {
		return []Result{
			{
				Name:    fmt.Sprintf("Directory not found: %s", dir),
				GUI:     false,
				Type:    "file",
				Source:  dir,
				Command: "check directory",
			},
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
			select {
			case <-done:
				return errors.New("stopped")
			default:
			}
			if err != nil {
				return nil
			}
			
			base := de.Name()
			if base == ".git" || base == "node_modules" || strings.HasPrefix(base, ".cache") {
				if de.IsDir() {
					return filepath.SkipDir
				}
			}
			if de.IsDir() {
				return nil
			}
			name := filepath.Base(path)
			if term == "" || strings.Contains(strings.ToLower(name), strings.ToLower(term)) || strings.Contains(strings.ToLower(path), strings.ToLower(term)) {
				mu.Lock()
				if count >= max {
					mu.Unlock()
					close(done)
					return errors.New("max reached")
				}
				count++
				mu.Unlock()
				found <- Result{
					Name:    name,
					GUI:     false,
					Type:    "file",
					Source:  path,
					Command: fileOpenCommand(path),
				}
			}
			return nil
		})
	}()

	go func() {
		wg.Wait()
		close(found)
	}()

	out := []Result{}
	for r := range found {
		out = append(out, r)
		if len(out) >= max {
			break
		}
	}
	// ensure unique by source
	uniq := map[string]Result{}
	for _, r := range out {
		uniq[r.Source] = r
	}
	final := []Result{}
	for _, v := range uniq {
		final = append(final, v)
	}
	// sort by name
	sort.Slice(final, func(i, j int) bool { return strings.ToLower(final[i].Name) < strings.ToLower(final[j].Name) })
	return final
}

func fileOpenCommand(p string) string {
	// prefer xdg-open, then gio
	if pth, _ := exec.LookPath("xdg-open"); pth != "" {
		return "xdg-open " + shellEscape(p)
	}
	if pth, _ := exec.LookPath("gio"); pth != "" {
		return "gio open " + shellEscape(p)
	}
	return "xdg-open " + shellEscape(p)
}

func searchCmdMode(input string) []Result {
	input = strings.TrimSpace(input)
	if input == "" {
		return []Result{{
			Name:    "No command provided",
			GUI:     false,
			Type:    "cmd",
			Source:  "system",
			Command: "",
		}}
	}

	toks := tokensFrom(input)
	terminalPrefix := ""
	cmdTokens := toks

	if len(toks) >= 2 && toks[0] == "--terminal" {
		terminalPrefix = toks[1]
		if len(toks) > 2 {
			cmdTokens = toks[2:]
		} else {
			cmdTokens = []string{}
		}
	}

	cmdStr := strings.Join(cmdTokens, " ")
	fullCmd := cmdStr
	if terminalPrefix != "" && cmdStr != "" {
		fullCmd = fmt.Sprintf("%s %s", terminalPrefix, cmdStr)
	}

	if len(cmdTokens) == 1 {
		helpWord := cmdTokens[0]
		helpCmd := fmt.Sprintf("%s -h", helpWord)
		return []Result{{
			Name:    runHelp(helpWord),
			GUI:     false,
			Type:    "cmd",
			Source:  "help",
			Command: helpCmd,
		}}
	}

	return []Result{{
		Name:    cmdStr,
		GUI:     false,
		Type:    "cmd",
		Source:  "user",
		Command: fullCmd,
	}}
}

func searchURLMode(link string) []Result {
	link = strings.TrimSpace(link)
	if link == "" {
		return []Result{{
			Name:    "No URL provided",
			GUI:     false,
			Type:    "url",
			Source:  "",
			Command: "",
		}}
	}

	if _, err := url.ParseRequestURI(link); err != nil {
		return []Result{{
			Name:    "Invalid URL",
			GUI:     false,
			Type:    "url",
			Source:  link,
			Command: "",
		}}
	}

	cmd := fmt.Sprintf("xdg-open %s", shellEscape(link))
	return []Result{{
		Name:    link,
		GUI:     false,
		Type:    "url",
		Source:  link,
		Command: cmd,
	}}
}


