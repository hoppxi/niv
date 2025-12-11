package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/hoppxi/niv/internal/manager"
	"github.com/hoppxi/niv/internal/wallpaper"
	"github.com/hoppxi/niv/internal/watchers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start EWW widgets and event-driven watchers",
	Run: func(cmd *cobra.Command, args []string) {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		fmt.Println("Starting EWW widgets...")

		ewwCmd, ewwCancel := manager.NewCmd("eww", "daemon")
		eww := manager.StartTrackedCmd(ewwCmd, ewwCancel)
		if eww == nil {
			fmt.Println("Failed to start eww daemon")
			return
		}
		configDir, _ := os.UserConfigDir()
		nivConf := filepath.Join(configDir, "eww", "niv.yaml")

		viper.SetConfigFile(nivConf)
		viper.SetConfigType("yaml")
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Cannot read niv.yaml")
		}

		wallpaper.SetWallpaperStartup(nivConf)
		wallpaper.ProcessAndWriteWallpapers(nivConf)

		watchers.ConfigUpdate()
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			watchers.ConfigUpdate()
			wallpaper.SetWallpaperStartup(nivConf)
			wallpaper.ProcessAndWriteWallpapers(nivConf)
		})

		manager.Manage.StartWatcher(watchers.StartAudioWatcher)
		manager.Manage.StartWatcher(watchers.StartNetworkWatcher)
		manager.Manage.StartWatcher(watchers.StartBluetoothWatcher)
		manager.Manage.StartWatcher(watchers.StartDisplayWatcher)
		manager.Manage.StartWatcher(watchers.StartMediaWatcher)
		manager.Manage.StartWatcher(watchers.StartWorkspaceWatcher)
		manager.Manage.StartWatcher(watchers.StartMiscWatcher)
		manager.Manage.StartWatcher(watchers.StartEscWatcher)

		for _, w := range []string{"bar", "wallpaper", "clock", "notification-view"} {
			if err := exec.Command("eww", "open", w).Run(); err != nil {
				fmt.Printf("Failed to open widget %s: %v\n", w, err)
			}
		}

		go manager.Manage.StartIPCServer()

		fmt.Println("Niv started. Listening on IPC socket or press Ctrl+C to stop.")
		<-sigChan

		fmt.Println("\nReceived local shutdown signal, stopping all processes and watchers...")
		manager.Manage.StopAll()
		os.Exit(0)
	},
}
