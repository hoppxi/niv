package cmd

import (
	"os/exec"

	"github.com/spf13/cobra"
)

var toggleCmd = &cobra.Command{
	Use:   "toggle <widget>",
	Short: "Toggle a widget",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		widget := args[0]
		exec.Command("eww", "open", "--toggle", widget).Run()
	},
}
