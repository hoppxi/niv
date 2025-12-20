package cmd

import (
	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var displayCmd = &cobra.Command{
	Use:   "display",
	Short: "Control display settings",
	Run: func(cmd *cobra.Command, args []string) {
		if brightness, _ := cmd.Flags().GetString("brightness"); brightness != "0%" {
			operation.Display.SetBrightness(brightness)
		}
	},
}

func init() {
	displayCmd.Flags().String("brightness", "10%", "Set display brightness level")
}
