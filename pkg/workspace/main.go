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
	Address      string `json:"address"`
	Workspace    struct {
		ID int `json:"id"`
	} `json:"workspace"`
}

// Updated Workspace struct to hold a slice of Window structs
type Workspace struct {
	ID      int      `json:"id"`
	Windows []Window `json:"windows"`
	Active  bool     `json:"active"`
}

// Updated Window struct to include the Icon field
type Window struct {
	Title     string `json:"title"`
	Workspace int    `json:"workspace"`
	Class     string `json:"class"`
	Address   string `json:"address"`
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
		return Workspace{ID: 1, Windows: nil}
	}
	// Return empty slice instead of 0
	return Workspace{ID: data.ID, Windows: []Window{}}
}

func GetActiveWindow() Window {
	var data hyprWindow
	res, err := hyprQuery("j/activewindow")
	if err != nil || json.Unmarshal(res, &data) != nil {
		return Window{Title: "Desktop", Workspace: 1, Class: "Wigo"}
	}

	title, class := data.Title, data.Class
	if title == "" {
		title = "Desktop"
	}
	if class == "" {
		class = data.InitialClass
	}

	return Window{
		Title:     title,
		Workspace: data.Workspace.ID,
		Class:     class,
		Address:   data.Address,
	}
}

func GetWorkspaces() []Workspace {
	persistent := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	workspaceMap := make(map[int][]Window)

	for _, id := range persistent {
		workspaceMap[id] = []Window{}
	}

	var hyprWs []hyprWorkspace
	if res, err := hyprQuery("j/workspaces"); err == nil {
		if err := json.Unmarshal(res, &hyprWs); err == nil {
			for _, w := range hyprWs {
				if _, ok := workspaceMap[w.ID]; !ok {
					workspaceMap[w.ID] = []Window{}
				}
			}
		}
	}

	var clients []hyprWindow
	if res, err := hyprQuery("j/clients"); err == nil {
		if err := json.Unmarshal(res, &clients); err == nil {
			for _, c := range clients {
				class := c.Class
				if class == "" {
					class = c.InitialClass
				}

				win := Window{
					Title:     c.Title,
					Workspace: c.Workspace.ID,
					Class:     class,
					Address:   c.Address,
				}

				workspaceMap[c.Workspace.ID] = append(workspaceMap[c.Workspace.ID], win)
			}
		}
	}

	ids := make([]int, 0, len(workspaceMap))
	for id := range workspaceMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	out := make([]Workspace, 0, len(ids))
	activeWs := GetActiveWorkspace()
	for _, id := range ids {
		out = append(out, Workspace{
			ID:      id,
			Windows: workspaceMap[id],
			Active:  id == activeWs.ID,
		})
	}

	return out
}

func GetWorkspace(id int) Workspace {
	// 1. Determine if this workspace is the currently focused one
	activeWs := GetActiveWorkspace()
	isActive := (id == activeWs.ID)

	windows := []Window{}
	if res, err := hyprQuery("j/clients"); err == nil {
		var clients []hyprWindow
		if err := json.Unmarshal(res, &clients); err == nil {
			for _, c := range clients {
				if c.Workspace.ID == id {
					class := c.Class
					if class == "" {
						class = c.InitialClass
					}

					windows = append(windows, Window{
						Title:     c.Title,
						Workspace: c.Workspace.ID,
						Class:     class,
						Address:   c.Address,
					})
				}
			}
		}
	}

	return Workspace{
		ID:      id,
		Windows: windows,
		Active:  isActive,
	}
}
