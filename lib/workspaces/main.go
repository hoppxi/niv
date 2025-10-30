/*
 Listens to Hyprland's IPC socket for workspace and active window changes,
 and updates eww variables accordingly.
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
	var lastActive, lastWorkspace string

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
			data := getFocusedWorkspace()
			payload, _ := json.Marshal(data)
			if string(payload) != lastWorkspace {
				lastWorkspace = string(payload)
				runEww("FOCUSED_WORKSPACE", lastWorkspace)
			}
		}
	}
}

func runEww(label, payload string) {
	// Simple, non-blocking call — no bash, minimal overhead
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
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		// Fallback for older versions
		socket = fmt.Sprintf("/tmp/hypr/%s/.socket2.sock", sig)
	}
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
		_ = json.Unmarshal(data, &aw)
	}
	return aw
}

func getFocusedWorkspace() Workspace {
	data := getHyprctlJSON("activeworkspace")
	ws := Workspace{ID: 1, Windows: 0}
	if data != nil {
		_ = json.Unmarshal(data, &ws)
	}
	return ws
}
