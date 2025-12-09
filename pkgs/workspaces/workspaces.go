package workspaces

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
)

func runtimeDir() string {
	return os.Getenv("XDG_RUNTIME_DIR")
}

func instanceSig() string {
	return os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
}

func ctlSocket() string {
	return fmt.Sprintf("%s/hypr/%s/.socket.sock", runtimeDir(), instanceSig())
}

func eventSocket() string {
	return fmt.Sprintf("%s/hypr/%s/.socket2.sock", runtimeDir(), instanceSig())
}

func hyprJSON(cmd string, v any) error {
	// Run hyprctl -j <cmd>
	out, err := exec.Command("hyprctl", "-j", cmd).Output()
	if err != nil {
		return err
	}

	// Parse JSON output
	return json.Unmarshal(out, v)
}

type hyprWorkspace struct {
	Id      int `json:"id"`
	Windows int `json:"windows"`
}

type hyprWindow struct {
	Title        string `json:"title"`
	Class        string `json:"class"`
	InitialClass string `json:"initialClass"`
	Workspace    struct {
		Id int `json:"id"`
	} `json:"workspace"`
}

type Workspace struct {
	ID      int `json:"id"`
	Windows int `json:"windows"`
}

type Window struct {
	Title     string `json:"title"`
	Workspace int    `json:"workspace"`
	Class     string `json:"class"`
}

func GetActiveWorkspace() Workspace {
	var data hyprWorkspace

	err := hyprJSON("activeworkspace", &data)
	if err != nil {
		return Workspace{ID: 1, Windows: 0}
	}

	return Workspace{
		ID:      data.Id,
		Windows: data.Windows,
	}
}

func GetActiveWindow() Window {
	var data hyprWindow

	err := hyprJSON("activewindow", &data)
	if err != nil {
		return Window{Title: "Desktop", Workspace: 1, Class: "Niv"}
	}

	title := data.Title
	if title == "" {
		title = "Desktop"
	}

	class := data.Class
	if class == "" {
		class = data.InitialClass
	}
	if class == "" {
		class = "Niv"
	}

	return Window{
		Title:     title,
		Workspace: data.Workspace.Id,
		Class:     class,
	}
}

func GetWorkspaces() []Workspace {
	// Persistent 1–5
	persistent := []int{1, 2, 3, 4, 5}
	exists := map[int]int{}

	for _, id := range persistent {
		exists[id] = 0
	}

	var ws []hyprWorkspace
	err := hyprJSON("workspaces", &ws)
	if err == nil {
		for _, w := range ws {
			exists[w.Id] = w.Windows
		}
	}

	ids := make([]int, 0, len(exists))
	for id := range exists {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	out := make([]Workspace, 0, len(ids))
	for _, id := range ids {
		out = append(out, Workspace{ID: id, Windows: exists[id]})
	}
	return out
}
