package subscribe

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func ConfigEvents() <-chan string {
	events := make(chan string)

	viper.OnConfigChange(func(e fsnotify.Event) {
		select {
		case events <- "changed":
		case <-time.After(10 * time.Millisecond):
		}
	})

	go func() {
		viper.WatchConfig()
		log.Println("contents: [subscribe] Listening for config file system events...")
	}()

	return events
}