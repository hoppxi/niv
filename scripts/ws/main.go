/*
	A script to open, close and toggle widgets by tracking them.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ToggleEntry struct {
	Name  string `json:"name"`
	Stack int    `json:"stack"`
}

type WidgetState struct {
	OpenToggles []ToggleEntry `json:"open_toggles"`
}

var (
	stateFile = "/tmp/niv_state.json"

	// Widgets that can be toggled
	togglableWidgets = []string{"quick-settings", "planner", "shelf", "system-monitor", "media-control", "launcher"}

	// Widgets that support stack states
	stackVarMap = map[string]string{
		"quick-settings": "ACTIVE_STACK_QUICK_SETTINGS",
		"shelf":          "ACTIVE_STACK_SHELF",
	}
)

func isTogglable(name string) bool {
	return slices.Contains(togglableWidgets, name)
}

func isStackable(name string) bool {
	_, ok := stackVarMap[name]
	return ok
}

func readState() (*WidgetState, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &WidgetState{OpenToggles: []ToggleEntry{}}, nil
		}
		return nil, err
	}
	var s WidgetState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func saveState(s *WidgetState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0644)
}

func runEww(args ...string) error {
	cmd := exec.Command("eww", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findWidget(s *WidgetState, widget string) (int, *ToggleEntry) {
	for i, e := range s.OpenToggles {
		if e.Name == widget {
			return i, &s.OpenToggles[i]
		}
	}
	return -1, nil
}

func resetStack(widget string) {
	if varName, ok := stackVarMap[widget]; ok {
		runEww("update", fmt.Sprintf("%s=0", varName))
	}
}

func setStack(widget string, stack int) {
	if varName, ok := stackVarMap[widget]; ok {
		runEww("update", fmt.Sprintf("%s=%d", varName, stack))
	}
}

func closeAllToggles(s *WidgetState) {
	for _, entry := range s.OpenToggles {
		resetStack(entry.Name)
		runEww("close", entry.Name)
	}
	s.OpenToggles = []ToggleEntry{}
	saveState(s)
}

func handleOpen(widget string, stack int) error {
	s, err := readState()
	if err != nil {
		return err
	}

	_, existing := findWidget(s, widget)

	if len(s.OpenToggles) > 0 {
		for _, entry := range s.OpenToggles {
			resetStack(entry.Name)
			runEww("close", entry.Name)
		}
		s.OpenToggles = []ToggleEntry{}
	}

	if existing != nil {
		saveState(s)
		return nil
	}

	if err := runEww("open", widget); err != nil {
		return err
	}

	if isStackable(widget) {
		setStack(widget, stack)
	}

	if isTogglable(widget) {
		s.OpenToggles = append(s.OpenToggles, ToggleEntry{Name: widget, Stack: stack})
	}

	return saveState(s)
}

func handleClose(widget string) error {
	s, err := readState()
	if err != nil {
		return err
	}

	if widget == "all" {
		closeAllToggles(s)
		return nil
	}

	runEww("close", widget)
	resetStack(widget)

	if isTogglable(widget) {
		newList := []ToggleEntry{}
		for _, e := range s.OpenToggles {
			if e.Name != widget {
				newList = append(newList, e)
			}
		}
		s.OpenToggles = newList
	}
	return saveState(s)
}

func handleUpdate(widget string, stack int) error {
	if !isStackable(widget) {
		return fmt.Errorf("%s is not stackable", widget)
	}

	s, err := readState()
	if err != nil {
		return err
	}

	setStack(widget, stack)

	idx, entry := findWidget(s, widget)
	if entry != nil {
		s.OpenToggles[idx].Stack = stack
		saveState(s)
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  niv open <widget> [--stack n]")
		fmt.Println("  niv close <widget|all>")
		fmt.Println("  niv update <widget> --stack n")
		return
	}

	action := strings.ToLower(os.Args[1])
	widget := os.Args[2]
	stack := 0

	for i := 3; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--stack" {
			fmt.Sscanf(os.Args[i+1], "%d", &stack)
		}
	}

	os.MkdirAll(filepath.Dir(stateFile), 0755)

	switch action {
	case "open":
		if err := handleOpen(widget, stack); err != nil {
			fmt.Println("Error:", err)
		}
	case "close":
		if err := handleClose(widget); err != nil {
			fmt.Println("Error:", err)
		}
	case "update":
		if err := handleUpdate(widget, stack); err != nil {
			fmt.Println("Error:", err)
		}
	default:
		fmt.Println("Invalid command. Use open, close, or update.")
	}
}
