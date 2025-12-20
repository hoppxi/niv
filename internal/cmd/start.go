package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/hoppxi/wigo/internal/manager"
	"github.com/hoppxi/wigo/internal/wallpaper"
	"github.com/hoppxi/wigo/internal/watchers"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start EWW widgets and event-driven watchers",
	Run: func(cmd *cobra.Command, args []string) {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		fmt.Println("Starting EWW widgets...")

		ewwCmd, ewwCancel := manager.NewCmd("eww", "daemon", "--no-daemonize")
		eww := manager.StartTrackedCmd(ewwCmd, ewwCancel)
		if eww == nil {
			fmt.Println("Failed to start eww daemon")
			return
		}
		time.Sleep(1 * time.Second)

		cfg := manager.Config.Load()
		wallpaper.SetWallpaperStartup()
		watchers.ConfigUpdate(cfg)
		manager.Config.Watch(func() {
			watchers.ConfigUpdate(cfg)
		})

		manager.Manage.StartWatcher(watchers.StartAudioWatcher)
		manager.Manage.StartWatcher(watchers.StartBatteryWatcher)
		manager.Manage.StartWatcher(watchers.StartNetworkWatcher)
		manager.Manage.StartWatcher(watchers.StartBluetoothWatcher)
		manager.Manage.StartWatcher(watchers.StartDisplayWatcher)
		manager.Manage.StartWatcher(watchers.StartMediaWatcher)
		manager.Manage.StartWatcher(watchers.StartWorkspaceWatcher)
		manager.Manage.StartWatcher(watchers.StartMiscWatcher)
		manager.Manage.StartWatcher(watchers.StartEscWatcher)

		time.Sleep(500 * time.Millisecond)

		if err := exec.Command("eww", "open-many", "bar", "wallpaper", "clock", "notification-view", "osd").Run(); err != nil {
			fmt.Printf("Failed to start widgets %v\n", err)
		}

		go manager.Manage.StartIPCServer()

		fmt.Println("wigo started. Listening on IPC socket or press Ctrl+C to stop.")
		<-sigChan

		fmt.Println("\nReceived local shutdown signal, stopping all processes and watchers...")
		manager.Manage.StopAll()
		os.Exit(0)
	},
}
