package cmd

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/hoppxi/wigo/internal/watchers"
	"github.com/spf13/cobra"
)

var (
	dndFlag           string
	dndToggle         bool
	dndState          bool
	clearFlag         bool
	countFlag         bool
	closeID           uint32
	closeViewOnly     bool
	execFlag          []string
	actionIDFlag      string
	actionNotifIDFlag uint32
	notifyFlag        bool

	appNameFlag       string
	replacesIDFlag    uint32
	appIconFlag       string
	summaryFlag       string
	bodyFlag          string
	actionsFlag       []string
	expireTimeoutFlag int32
)

var notificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Notification daemon and tools",
	Run: func(cmd *cobra.Command, args []string) {
		if notifyFlag {
			if summaryFlag == "" || bodyFlag == "" {
				fmt.Println("Error: --summary (-s) and --body (-b) flags are required when using --notify.")
				_ = cmd.Usage()
				return
			}

			hints := make(map[string]dbus.Variant)

			id := (*watchers.NotificationDaemon).Notify(
				&watchers.NotificationDaemon{},
				appNameFlag,
				replacesIDFlag,
				appIconFlag,
				summaryFlag,
				bodyFlag,
				actionsFlag,
				hints,
				expireTimeoutFlag,
			)

			fmt.Printf("Notification sent. ID: %d\n", id)
			return
		}

		if clearFlag {
			if err := watchers.NotificationHelper.ClearHistory(); err != nil {
				fmt.Println("Error clearing history:", err)
			} else {
				fmt.Println("History cleared")
			}
			return
		}

		if dndToggle {
			newState := watchers.NotificationHelper.ToggleDND()
			fmt.Println(newState)
			return
		}

		if dndState {
			state := watchers.NotificationHelper.GetDNDState()
			fmt.Println(state)
			return
		}

		if dndFlag != "" {
			if err := watchers.NotificationHelper.SetDNDState(dndFlag); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("DND set to", dndFlag)
			}
			return
		}

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
	notificationCmd.Flags().BoolVar(&notifyFlag, "notify", false, "Send a new desktop notification")
	notificationCmd.Flags().StringVarP(&appNameFlag, "app-name", "a", "CLI Notification", "Application name for the notification")
	notificationCmd.Flags().Uint32VarP(&replacesIDFlag, "replace-id", "r", 0, "ID of the notification to replace")
	notificationCmd.Flags().StringVar(&appIconFlag, "icon", "", "Application icon path or name")
	notificationCmd.Flags().StringVarP(&summaryFlag, "summary", "s", "", "Notification summary/title (Required with --notify)")
	notificationCmd.Flags().StringVarP(&bodyFlag, "body", "b", "", "Notification body/content (Required with --notify)")
	notificationCmd.Flags().StringSliceVar(&actionsFlag, "actions", nil, "List of action IDs (e.g., 'default,Close')")
	notificationCmd.Flags().Int32Var(&expireTimeoutFlag, "timeout", -1, "Notification timeout in milliseconds (-1 for default)")
}
