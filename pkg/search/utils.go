package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

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

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func toString(v any) string {
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

func printJSON(arr []Result) {
	enc, _ := json.MarshalIndent(arr, "", "  ")
	fmt.Println(string(enc))
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

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
