package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type ToggleEntry struct {
	Name  string `json:"name"`
	Stack int    `json:"stack"`
}

type WidgetState struct {
	OpenToggles []ToggleEntry `json:"open_toggles"`
}

var stateFile = "/tmp/niv_state.json"

func readState() (*WidgetState, error) {
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		defaultState := &WidgetState{OpenToggles: []ToggleEntry{}}
		if err := saveState(defaultState); err != nil {
			return nil, fmt.Errorf("failed to create default state file: %w", err)
		}
		return defaultState, nil
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var s WidgetState
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Println("Corrupted JSON, resetting state file")
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

func bindEsc() error {
	cmd := exec.Command("hyprctl", "keyword", "bind", ",escape,exec,~/.config/eww/bin/niv-ws close all")
	return cmd.Run()
}

func unbindEsc() error {
	cmd := exec.Command("hyprctl", "keyword", "unbind", ",escape")
	return cmd.Run()
}

func main() {
	lastStateOpen := false

	for {
		state, err := readState()
		if err != nil {
			fmt.Println("Error reading state:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		anyOpen := len(state.OpenToggles) > 0

		if anyOpen && !lastStateOpen {
			fmt.Println("Widgets open, binding ESC...")
			if err := bindEsc(); err != nil {
				fmt.Println("Failed to bind ESC:", err)
			}
		} else if !anyOpen && lastStateOpen {
			fmt.Println("No widgets open, unbinding ESC...")
			if err := unbindEsc(); err != nil {
				fmt.Println("Failed to unbind ESC:", err)
			}
		}

		lastStateOpen = anyOpen
		time.Sleep(500 * time.Millisecond)
	}
}
