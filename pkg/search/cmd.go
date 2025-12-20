package search

import (
	"fmt"
	"os/exec"
	"strings"
)

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
			Comment: "Enter command to run",
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
			Comment: "",
		}}
	}

	return []Result{{
		Name:    cmdStr,
		GUI:     false,
		Type:    "cmd",
		Source:  "user",
		Command: fullCmd,
		Comment: "run" + fullCmd,
	}}
}
