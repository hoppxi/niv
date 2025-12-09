package cmd

import (
	"github.com/hoppxi/niv/internal/manager"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload EWW widgets and watchers",
	Run: func(cmd *cobra.Command, args []string) {
		manager.Manage.StopAll()     // stop everything
		startCmd.Run(cmd, args) // restart in-place
	},
}
