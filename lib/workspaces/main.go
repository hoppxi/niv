package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Workspace struct {
	ID      int `json:"id"`
	Windows int `json:"windows"`
}

type ActiveWindow struct {
	Title     string `json:"title"`
	Workspace int    `json:"workspace"`
}

func main() {
	sock := getHyprlandSocket()
	conn, err := net.Dial("unix", sock)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to Hyprland socket: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Fprintf(os.Stderr, "connected to %s\n", sock)

	scanner := bufio.NewScanner(conn)

	var lastActive, lastWorkspace, lastWorkspaces string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "activewindow>>") {
			data := getActiveWindow()
			payload, _ := json.Marshal(data)
			if string(payload) != lastActive {
				lastActive = string(payload)
				runEww("ACTIVE_WINDOW", lastActive)
			}
		}

		if strings.HasPrefix(line, "workspace>>") ||
			strings.HasPrefix(line, "focusedmon>>") {
			focused := getFocusedWorkspace()
			payload, _ := json.Marshal(focused)
			if string(payload) != lastWorkspace {
				lastWorkspace = string(payload)
				runEww("FOCUSED_WORKSPACE", lastWorkspace)
			}

			// Also update WORKSPACES
			workspaces := getWorkspaces()
			wsPayload, _ := json.Marshal(workspaces)
			if string(wsPayload) != lastWorkspaces {
				lastWorkspaces = string(wsPayload)
				runEww("WORKSPACES", lastWorkspaces)
			}
		}
	}
}

func runEww(label, payload string) {
	cmd := exec.Command("eww", "update", fmt.Sprintf("%s=%s", label, payload))
	_ = cmd.Start()
}

func getHyprlandSocket() string {
	sig := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if sig == "" {
		fmt.Fprintln(os.Stderr, "HYPRLAND_INSTANCE_SIGNATURE not set; is Hyprland running?")
		os.Exit(1)
	}

	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = "/tmp"
	}

	socket := filepath.Join(runtime, fmt.Sprintf("hypr/%s/.socket2.sock", sig))
	return socket
}

func getHyprctlJSON(arg string) []byte {
	out, err := exec.Command("hyprctl", arg, "-j").Output()
	if err != nil {
		return nil
	}
	return out
}

func getActiveWindow() ActiveWindow {
	data := getHyprctlJSON("activewindow")
	aw := ActiveWindow{Title: "Desktop", Workspace: 1}
	if data != nil {
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err == nil {
			if t, ok := parsed["title"].(string); ok {
				aw.Title = t
			}
			if ws, ok := parsed["workspace"].(map[string]any); ok {
				if id, ok := ws["id"].(float64); ok {
					aw.Workspace = int(id)
				}
			}
		}
	}
	return aw
}

func getFocusedWorkspace() Workspace {
	data := getHyprctlJSON("activeworkspace")
	ws := Workspace{ID: 1, Windows: 0}
	if data != nil {
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err == nil {
			if id, ok := parsed["id"].(float64); ok {
				ws.ID = int(id)
			}
		}
	}
	return ws
}

// Persistent first 5 workspaces + dynamic
func getWorkspaces() []Workspace {
	base := []Workspace{
		{ID: 1, Windows: 0},
		{ID: 2, Windows: 0},
		{ID: 3, Windows: 0},
		{ID: 4, Windows: 0},
		{ID: 5, Windows: 0},
	}

	data := getHyprctlJSON("workspaces")
	if data == nil {
		return base
	}

	var all []map[string]any
	if err := json.Unmarshal(data, &all); err != nil {
		return base
	}

	existing := make(map[int]int)
	for _, b := range base {
		existing[b.ID] = b.Windows
	}

	for _, ws := range all {
		if id, ok := ws["id"].(float64); ok {
			idInt := int(id)
			windows := 0
			if wins, ok := ws["windows"].(float64); ok {
				windows = int(wins)
			}
			existing[idInt] = windows
		}
	}

	result := []Workspace{}
	for i := 1; i <= 5; i++ {
		result = append(result, Workspace{ID: i, Windows: existing[i]})
	}

	extra := []int{}
	for id := range existing {
		if id > 5 {
			extra = append(extra, id)
		}
	}

	sort.Ints(extra)

	for _, id := range extra {
		result = append(result, Workspace{ID: id, Windows: existing[id]})
	}

	return result
}
