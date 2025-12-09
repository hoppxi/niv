package cmd

import (
	"fmt"

	"github.com/hoppxi/niv/internal/watchers"
	"github.com/spf13/cobra"
)

var (
	dndFlag           string
	dndToggle     		bool
	dndState      		bool
	clearFlag     		bool
	countFlag     		bool
	execFlag          []string
	closeID 			    uint32
	closeViewOnly 		bool
	actionIDFlag      string
	actionNotifIDFlag uint32
)

var notificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Notification daemon and tools",
	Run: func(cmd *cobra.Command, args []string) {

		// --- One-shot: clear history ---
		if clearFlag {
			if err := watchers.NotificationHelper.ClearHistory(); err != nil {
				fmt.Println("Error clearing history:", err)
			} else {
				fmt.Println("History cleared")
			}
			return
		}

		// --- One-shot: DND toggle ---
		if dndToggle {
			newState := watchers.NotificationHelper.ToggleDND()
			fmt.Println(newState)
			return
		}

		// --- One-shot: DND state ---
		if dndState {
			state := watchers.NotificationHelper.GetDNDState()
			fmt.Println(state)
			return
		}

		// --- One-shot: set DND explicitly ---
		if dndFlag != "" {
			if err := watchers.NotificationHelper.SetDNDState(dndFlag); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("DND set to", dndFlag)
			}
			return
		}

		// --- One-shot: count notifications in history ---
		if countFlag {
			fmt.Println(watchers.NotificationHelper.GetHistoryCount())
			return
		}

		if closeID != 0 {
			if closeViewOnly {
					watchers.NotificationHelper.CloseNotificationViewOnly(closeID)
			} else {
					watchers.NotificationHelper.CloseNotificationByID(closeID)
			}
			return
		}

		if actionIDFlag != "" && actionNotifIDFlag != 0 {
			err := watchers.NotificationHelper.InvokeAction(actionNotifIDFlag, actionIDFlag)
			if err != nil {
					fmt.Println("Error performing action:", err)
			} else {
					fmt.Printf("Action '%s' invoked on notification %d\n", actionIDFlag, actionNotifIDFlag)
			}
			return
		}

		// --- Start daemon ---
		watchers.StartNotificationWatcher(execFlag)
	},
}

func init() {
	notificationCmd.Flags().StringVar(&dndFlag, "dnd", "", "Set do-not-disturb explicitly (on|off)")
	notificationCmd.Flags().BoolVar(&dndToggle, "dnd-toggle", false, "Toggle DND state")
	notificationCmd.Flags().BoolVar(&dndState, "dnd-state", false, "Print current DND state (true/false)")
	notificationCmd.Flags().BoolVar(&clearFlag, "clear-history", false, "Clear notification history")
	notificationCmd.Flags().BoolVar(&countFlag, "count", false, "Print number of notifications in history")
	notificationCmd.Flags().StringSliceVar(&execFlag, "exec", nil, "Command to run for notifications")
	notificationCmd.Flags().Uint32Var(&closeID, "close", 0, "Close notification by ID")
	notificationCmd.Flags().BoolVar(&closeViewOnly, "view-only", false, "When used with --close: only close from current NOTIFICATION view, keep history")
	notificationCmd.Flags().StringVar(&actionIDFlag, "action", "", "Perform action ID on a notification")
	notificationCmd.Flags().Uint32Var(&actionNotifIDFlag, "id", 0, "Notification ID for --action")
}
