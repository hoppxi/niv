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
	Name string `json:"name"`
}

type WidgetState struct {
	OpenToggles []ToggleEntry `json:"open_toggles"`
}

var (
	stateFile = "/tmp/niv_state.json"

	togglableWidgets = []string{
		"quick-settings", "planner", "shelf", "system-monitor",
		"media-control", "launcher", "wlogout", "screenshot-utils", "wallpaper",
	}

	stackVarMap = map[string]string{
		"quick-settings": "ACTIVE_STACK_QUICK_SETTINGS",
		"shelf":          "ACTIVE_STACK_SHELF",
		"system-monitor": "ACTIVE_STACK_SYSTEM_MONITOR",
		"planner":        "ACTIVE_STACK_PLANNER",
	}
)

func isTogglable(name string) bool {
	return slices.Contains(togglableWidgets, name)
}

func isStackable(name string) bool {
	_, ok := stackVarMap[name]
	return ok
}

func runEww(args ...string) error {
	cmd := exec.Command("eww", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func readState() (*WidgetState, error) {
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		s := &WidgetState{OpenToggles: []ToggleEntry{}}
		if err := saveState(s); err != nil {
			return nil, err
		}
		return s, nil
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var s WidgetState
	if err := json.Unmarshal(data, &s); err != nil || s.OpenToggles == nil {
		s = WidgetState{OpenToggles: []ToggleEntry{}}
		_ = saveState(&s)
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

func findWidget(s *WidgetState, widget string) (int, *ToggleEntry) {
	for i, e := range s.OpenToggles {
		if e.Name == widget {
			return i, &s.OpenToggles[i]
		}
	}
	return -1, nil
}

func closeAll(s *WidgetState) error {
	for _, e := range s.OpenToggles {
		runEww("close", e.Name)
		if isStackable(e.Name) {
			if varName, ok := stackVarMap[e.Name]; ok {
				_ = runEww("update", fmt.Sprintf("%s=0", varName))
			}
		}
	}
	s.OpenToggles = []ToggleEntry{}
	return saveState(s)
}

func handleOpen(widget string, stack int) error {
	s, err := readState()
	if err != nil {
		return err
	}

	idx, existing := findWidget(s, widget)
	if existing != nil {
		runEww("close", widget)
		if isStackable(widget) {
			if varName, ok := stackVarMap[widget]; ok {
				_ = runEww("update", fmt.Sprintf("%s=0", varName))
			}
		}
		s.OpenToggles = append(s.OpenToggles[:idx], s.OpenToggles[idx+1:]...)
		return saveState(s)
	}

	for _, e := range s.OpenToggles {
		runEww("close", e.Name)
		if isStackable(e.Name) {
			if varName, ok := stackVarMap[e.Name]; ok {
				_ = runEww("update", fmt.Sprintf("%s=0", varName))
			}
		}
	}
	s.OpenToggles = []ToggleEntry{}

	if err := runEww("open", widget); err != nil {
		return err
	}

	if isStackable(widget) {
		if varName, ok := stackVarMap[widget]; ok {
			_ = runEww("update", fmt.Sprintf("%s=%d", varName, stack))
		}
	}

	if isTogglable(widget) {
		s.OpenToggles = append(s.OpenToggles, ToggleEntry{Name: widget})
	}

	return saveState(s)
}

func handleClose(widget string) error {
	s, err := readState()
	if err != nil {
		return err
	}

	if widget == "all" {
		return closeAll(s)
	}

	runEww("close", widget)
	if isStackable(widget) {
		if varName, ok := stackVarMap[widget]; ok {
			_ = runEww("update", fmt.Sprintf("%s=0", varName))
		}
	}

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
	if isStackable(widget) {
		if varName, ok := stackVarMap[widget]; ok {
			_ = runEww("update", fmt.Sprintf("%s=%d", varName, stack))
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  niv open <widget> [--stack n]")
		fmt.Println("  niv close <widget|all>")
		fmt.Println("  niv update <widget> [--stack n]")
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
