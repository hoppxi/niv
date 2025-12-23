package cmd

import (
	"fmt"

	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var idleCmd = &cobra.Command{
	Use:   "idle",
	Short: "Control idle settings",
	Run: func(cmd *cobra.Command, args []string) {
		action, _ := cmd.Flags().GetString("action")

		switch action {
		case "enable":
			if err := operation.Idle.Inhibit(); err != nil {
				fmt.Println("Error enabling idle inhibition:", err)
			} else {
				fmt.Println("Idle inhibition enabled")
			}
		case "disable":
			if err := operation.Idle.UnInhibit(); err != nil {
				fmt.Println("Error disabling idle inhibition:", err)
			} else {
				fmt.Println("Idle inhibition disabled")
			}
		case "toggle":
			if err := operation.Idle.Toggle(); err != nil {
				fmt.Println("Error toggling idle inhibition:", err)
			} else {
				fmt.Println("Idle inhibition toggled")
			}
		default:
			fmt.Println("Unknown action. Use enable, disable, toggle, or status.")
		}
	},
}

func init() {
	idleCmd.Flags().StringP("action", "a", "status", "Action to perform: enable, disable, toggle")
}
