package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <widget>",
	Short: "Open a widget",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]
		exec.Command("eww", "open", widget).Run()
	},
}
