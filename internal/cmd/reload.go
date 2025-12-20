package cmd

import (
	"fmt"
	"strings"

	"github.com/hoppxi/wigo/internal/manager"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload EWW widgets and watchers",
	Run: func(cmd *cobra.Command, args []string) {
		response, err := manager.Manage.SendIPCCommand("STOP")
		if err != nil {
			fmt.Printf("Error: %v (Is the daemon running?)\n", err)
			return
		}

		fmt.Printf("Server response: %s\n", response)

		startCmd.Run(cmd, args) // restart in-place

		if strings.Contains(response, "OK") {
			fmt.Println("Niv daemon successfully reloaded.")
		}
	},
}
