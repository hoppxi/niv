package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hoppxi/wigo/internal/manager"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start EWW widgets and event-driven watchers",
	Run: func(cmd *cobra.Command, args []string) {
		if conn, err := manager.Manage.ConnectIPC(); err == nil {
			defer conn.Close()
			fmt.Println("Daemon already running. Sending start command...")
			if _, err := manager.Manage.SendIPCCommand("START"); err != nil {
				fmt.Printf("Failed to send start command: %v\n", err)
			}
			return
		}

		fmt.Println("Starting daemon...")

		ewwCmd, ewwCancel := manager.NewCmd("eww", "daemon", "--no-daemonize")
		eww := manager.StartTrackedCmd(ewwCmd, ewwCancel)
		if eww == nil {
			fmt.Println("Failed to start eww daemon")
			return
		}

		go manager.Manage.StartIPCServer()

		time.Sleep(100 * time.Millisecond)

		if _, err := manager.Manage.SendIPCCommand("START"); err != nil {
			fmt.Printf("Failed to initialize daemon: %v\n", err)
			manager.Manage.StopAll()
			return
		}

		fmt.Println("Daemon started successfully. Press Ctrl+C to stop.")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nReceived shutdown signal, stopping all processes and watchers...")
		manager.Manage.StopAll()
	},
}
