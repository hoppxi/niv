package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoppxi/wigo/internal/manager"
	"github.com/spf13/cobra"
)

var Version = "0.1.1"

var rootCmd = &cobra.Command{
	Use:     "wigo",
	Version: Version,
	Short:   "Wigo CLI for EWW widgets and system monitoring",
	Long:    "Wigo controls EWW widgets and monitors system info",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "setup" || cmd.Name() == "config" || cmd.Name() == "update" || cmd.Name() == "help" {
			return
		}

		wigoDir := getWigoDir()
		criticalFiles := []string{"wigo.yaml", "eww.yuck", "eww.scss"}

		for _, f := range criticalFiles {
			if _, err := os.Stat(filepath.Join(wigoDir, f)); os.IsNotExist(err) {
				fmt.Printf("Missing critical file: %s\n", f)
				fmt.Println("Hint: Run `wigo setup` to initialize your environment.")
				os.Exit(1)
			}
		}

		if cmd.Name() == "start" {
			return
		}

		conn, err := manager.Manage.ConnectIPC()
		if err != nil {
			fmt.Println("Error:", err)
			fmt.Println("Hint: run `wigo start` first")
			os.Exit(1)
		}
		conn.Close()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(generateConfigCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(audioCmd)
	rootCmd.AddCommand(batteryCmd)
	rootCmd.AddCommand(mediaCmd)
	rootCmd.AddCommand(networkCmd)
	rootCmd.AddCommand(bluetoothCmd)
	rootCmd.AddCommand(displayCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(notificationCmd)
	rootCmd.AddCommand(wallpaperCmd)
}
