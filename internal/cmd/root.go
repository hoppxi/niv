package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "niv",
	Short: "Niv CLI for EWW widgets and system monitoring",
	Long:  "Niv controls EWW widgets and monitors system info like audio, network, bluetooth, workspaces, display, icons, and misc info.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(audioCmd)
	rootCmd.AddCommand(mediaCmd)
	rootCmd.AddCommand(networkCmd)
	rootCmd.AddCommand(bluetoothCmd)
	rootCmd.AddCommand(displayCmd)
	rootCmd.AddCommand(verseCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(notificationCmd)
	rootCmd.AddCommand(wallpaperCmd)
}

