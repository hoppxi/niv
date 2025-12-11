package cmd

import (
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

const stateFile = "/tmp/niv_widget_state"

var (
	permanentWidgets  = []string{"bar", "wallpaper", "clock", "notification-view"}
	toggleableWidgets = []string{"media", "panel", "qs", "notification", "power", "launcher", "wallpaper-selector"}
)

func isPermanent(widget string) bool {
	if slices.Contains(permanentWidgets, widget) {
		return true
	}
	return false
}

func getTrackedWidget() string {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func setTrackedWidget(widget string) error {
	if widget == "" {
		return os.Remove(stateFile)
	}
	return os.WriteFile(stateFile, []byte(widget), 0644)
}

func closeWidget(widget string) {
	exec.Command("eww", "close", widget).Run()
	if !isPermanent(widget) {
		setTrackedWidget("")
	}
}

// OPEN COMMAND
var openCmd = &cobra.Command{
	Use:   "open <widget>",
	Short: "Open a widget",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]

		if !isPermanent(widget) {
			if tracked := getTrackedWidget(); tracked != "" && tracked != widget {
				closeWidget(tracked)
			}
			setTrackedWidget(widget)
		}

		exec.Command("eww", "open", widget).Run()
	},
}

// TOGGLE COMMAND
var toggleCmd = &cobra.Command{
	Use:   "toggle <widget>",
	Short: "Toggle a widget",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]

		if !isPermanent(widget) {
			if tracked := getTrackedWidget(); tracked != "" && tracked != widget {
				closeWidget(tracked)
			}
			setTrackedWidget(widget)
		}

		exec.Command("eww", "open", "--toggle", widget).Run()
	},
}

var closeCmd = &cobra.Command{
	Use:   "close <widget|all>",
	Short: "Close a widget or all widgets (bar and clock stay open)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]

		switch {
		case widget == "all":
			if tracked := getTrackedWidget(); tracked != "" {
				closeWidget(tracked)
			}
			for _, w := range toggleableWidgets {
				closeWidget(w)
			}
			return
		case widget == "entire":
			for _, w := range append(toggleableWidgets, permanentWidgets...) {
				closeWidget(w)
			}
		}

		closeWidget(widget)
	},
}
