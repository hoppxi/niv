package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <widget|all>",
	Short: "Close a widget or all widgets (bar and clock stay open)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]
		if widget == "all" {
			exec.Command("eww", "close", "all").Run()
			exec.Command("eww", "open", "bar").Run()
			exec.Command("eww", "open", "clock").Run()
		} else {
			exec.Command("eww", "close", widget).Run()
		}
	},
}
