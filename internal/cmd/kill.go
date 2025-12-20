package cmd

import (
	"fmt"
	"strings"

	"github.com/hoppxi/wigo/internal/manager"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill daemon and stop watchers.",
	Run: func(cmd *cobra.Command, args []string) {
		// Use the helper method defined in the manager
		response, err := manager.Manage.SendIPCCommand("STOP")
		if err != nil {
			fmt.Printf("Error: %v (Is the daemon running?)\n", err)
			return
		}

		fmt.Printf("Server response: %s\n", response)

		if strings.Contains(response, "OK") {
			fmt.Println("Niv daemon successfully shut down.")
		}
	},
}
