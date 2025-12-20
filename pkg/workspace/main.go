package workspace

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
)

type hyprWorkspace struct {
	ID      int `json:"id"`
	Windows int `json:"windows"`
}

type hyprWindow struct {
	Title        string `json:"title"`
	Class        string `json:"class"`
	InitialClass string `json:"initialClass"`
	Workspace    struct {
		ID int `json:"id"`
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

func ctlSocket() string {
	return filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"), ".socket.sock")
}

func hyprQuery(cmd string) ([]byte, error) {
	conn, err := net.Dial("unix", ctlSocket())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(conn)
}

func Dispatch(command string) error {
	_, err := hyprQuery(command)
	return err
}

func GetActiveWorkspace() Workspace {
	var data hyprWorkspace
	res, err := hyprQuery("j/activeworkspace")
	if err != nil || json.Unmarshal(res, &data) != nil {
		return Workspace{ID: 1, Windows: 0}
	}
	return Workspace{ID: data.ID, Windows: data.Windows}
}

func GetActiveWindow() Window {
	var data hyprWindow
	res, err := hyprQuery("j/activewindow")
	if err != nil || json.Unmarshal(res, &data) != nil {
		return Window{Title: "Desktop", Workspace: 1, Class: "Niv"}
	}

	title, class := data.Title, data.Class
	if title == "" {
		title = "Desktop"
	}
	if class == "" {
		class = data.InitialClass
	}
	if class == "" {
		class = "Niv"
	}

	return Window{
		Title:     title,
		Workspace: data.Workspace.ID,
		Class:     class,
	}
}

func GetWorkspaces() []Workspace {
	persistent := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	exists := map[int]int{}
	for _, id := range persistent {
		exists[id] = 0
	}

	var ws []hyprWorkspace
	res, err := hyprQuery("j/workspaces")
	if err == nil {
		if err := json.Unmarshal(res, &ws); err == nil {
			for _, w := range ws {
				exists[w.ID] = w.Windows
			}
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
