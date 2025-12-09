package watchers

import (
	"fmt"

	"github.com/hoppxi/niv/internal/subscribe"
	"github.com/hoppxi/niv/pkgs/workspaces"
)

var lastActiveWindow workspaces.Window
var lastActiveWorkspace workspaces.Workspace
var lastWorkspaceList []workspaces.Workspace

// Compare windows
func equalWindow(a, b workspaces.Window) bool {
	return a.Title == b.Title && a.Class == b.Class && a.Workspace == b.Workspace
}

// Compare workspaces
func equalWorkspace(a, b workspaces.Workspace) bool {
	return a.ID == b.ID && a.Windows == b.Windows
}

// Compare workspace lists
func equalWorkspaces(a, b []workspaces.Workspace) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalWorkspace(a[i], b[i]) {
			return false
		}
	}
	return true
}

func updateActiveWindow() {
	aw := workspaces.GetActiveWindow()
	if equalWindow(aw, lastActiveWindow) {
		return
	}
	lastActiveWindow = aw
	updateEww("ACTIVE_WINDOW", aw)
}

func updateActiveWorkspace() {
	aws := workspaces.GetActiveWorkspace()
	if equalWorkspace(aws, lastActiveWorkspace) {
		return
	}
	lastActiveWorkspace = aws
	updateEww("ACTIVE_WORKSPACE", aws)
}

func updateWorkspaceList() {
	ws := workspaces.GetWorkspaces()
	if equalWorkspaces(ws, lastWorkspaceList) {
		return
	}
	lastWorkspaceList = ws
	updateEww("WORKSPACES", ws)
}

// StartWorkspaceWatcher listens to Hyprland events and updates EWW accordingly
func StartWorkspaceWatcher(stop <-chan struct{}) {
	fmt.Println("[DEBUG] Starting workspace watcher...")

	// Subscribe to raw events
	rawEvents, err := subscribe.SubscribeEvents()
	if err != nil {
		fmt.Println("Cannot subscribe Hyprland events:", err)
		return
	}

	// Initial push to EWW
	updateWorkspaceList()
	updateActiveWorkspace()
	updateActiveWindow()
	for {
		select {
		case <-stop:
			return
		case e, ok := <-rawEvents:
			if !ok {
				return
			}

			switch e.Name {
			case "workspace", "createworkspace", "destroyworkspace", "movewindow", "openwindow", "closewindow":
				updateWorkspaceList()
				updateActiveWorkspace()
			case "activewindow", "windowtitle":
				updateActiveWindow()
			}
		}
	}
}
