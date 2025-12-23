package watchers

import (
	"fmt"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/workspace"
)

var lastActiveWindow workspace.Window
var lastActiveWorkspace workspace.Workspace
var lastWorkspaceList []workspace.Workspace

// Compare windows
func equalWindow(a, b workspace.Window) bool {
	return a.Title == b.Title && a.Class == b.Class && a.Workspace == b.Workspace
}

// Compare workspace
func equalWorkspace(a, b workspace.Workspace) bool {
	if a.ID != b.ID {
		return false
	}
	if len(a.Windows) != len(b.Windows) {
		return false
	}
	for i := range a.Windows {
		if !equalWindow(a.Windows[i], b.Windows[i]) {
			return false
		}
	}
	return true
}

// Compare workspace lists
func equalWorkspaces(a, b []workspace.Workspace) bool {
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
	aw := workspace.GetActiveWindow()
	if equalWindow(aw, lastActiveWindow) {
		return
	}
	lastActiveWindow = aw
	updateEww("ACTIVE_WINDOW", aw)
}

func updateActiveWorkspace() {
	aws := workspace.GetActiveWorkspace()
	if equalWorkspace(aws, lastActiveWorkspace) {
		return
	}
	lastActiveWorkspace = aws
	updateEww("ACTIVE_WORKSPACE", aws)
}

func updateWorkspaceList() {
	ws := workspace.GetWorkspaces()
	if equalWorkspaces(ws, lastWorkspaceList) {
		return
	}
	lastWorkspaceList = ws
	updateEww("WORKSPACES", ws)
}

func StartWorkspaceWatcher(stop <-chan struct{}) {
	fmt.Println("[DEBUG] Starting workspace watcher...")

	rawEvents, err := subscribe.SubscribeEvents()
	if err != nil {
		fmt.Println("Cannot subscribe Hyprland events:", err)
		return
	}

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
