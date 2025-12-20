package cmd

import (
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

const (
	stateFile    = "/tmp/wigo/wigo_widget_state"
	closerWidget = "closer"
)

var (
	permanentWidgets  = []string{"bar", "wallpaper", "clock", "notification-view", "osd"}
	toggleableWidgets = []string{"media", "panel", "quick-settings", "power", "launcher"}
)

func openCloser() {
	exec.Command("eww", "open", closerWidget).Run()
}

func closeCloser() {
	exec.Command("eww", "close", closerWidget).Run()
}

func isPermanent(widget string) bool {
	return slices.Contains(permanentWidgets, widget)
}

func getTrackedWidget() string {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func setTrackedWidget(widget string) error {
	if err := os.MkdirAll("/tmp/wigo", 0755); err != nil {
		return err
	}

	if widget == "" {
		f, err := os.OpenFile(stateFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		return f.Close()
	}

	return os.WriteFile(stateFile, []byte(widget), 0644)
}

func closeWidget(widget string) {
	exec.Command("eww", "close", widget).Run()
	if !isPermanent(widget) {
		setTrackedWidget("")
	}

	closeCloser()
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

		openCloser()
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
		tracked := getTrackedWidget()

		// If toggling OFF the same widget
		if tracked == widget {
			exec.Command("eww", "close", widget).Run()
			setTrackedWidget("")
			closeCloser()
			return
		}

		if !isPermanent(widget) {
			if tracked != "" && tracked != widget {
				closeWidget(tracked)
			}
			setTrackedWidget(widget)
		}

		openCloser()
		exec.Command("eww", "open", "--toggle", widget).Run()
	},
}

var closeCmd = &cobra.Command{
	Use:   "close <widget|all>",
	Short: "Close a widget or all widgets (bar and clock stay open)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]

		switch widget {
		case "all":
			if tracked := getTrackedWidget(); tracked != "" {
				closeWidget(tracked)
			}
			for _, w := range toggleableWidgets {
				exec.Command("eww", "close", w).Run()
			}
			closeCloser()
			return

		case "entire":
			for _, w := range append(toggleableWidgets, permanentWidgets...) {
				exec.Command("eww", "close", w).Run()
			}
			setTrackedWidget("")
			closeCloser()
			return
		}

		closeWidget(widget)
	},
}
