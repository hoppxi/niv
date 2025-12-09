package watchers

import (
	"fmt"
	"time"

	// Import your subscribe package
	"github.com/hoppxi/niv/internal/subscribe" // Change 'myconfig' to your actual module name
	"github.com/spf13/viper"
)

func ConfigUpdate() {
	updateEww("APPS_CONFIG", viper.Get("apps"))
	updateEww("WIDGETS_CONFIG", viper.Get("widgets"))
	updateEww("NOTIFICATION_CONFIG", viper.Get("notifications"))
	updateEww("DISABLED_NOTIFICATION_CONFIG", viper.Get("notifications_disabled"))
	updateEww("GENERAL_CONFIG", viper.Get("general"))
}

// StartDaemon initializes the long-running process that listens for changes
func StartDaemon() {
	fmt.Println("Starting Config Watcher Daemon...")

	// 1. Subscribe to events
	eventChan := subscribe.ConfigEvents()

	select {
	case msg := <-eventChan:
		if msg == "changed" {
			handleUpdate()
		}
	}
}

// handleUpdate contains the logic to apply changes
func handleUpdate() {
	fmt.Println("\n Config Change Detected!")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Updates applied.")
}
