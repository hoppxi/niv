package cmd

import (
	"fmt"

	"github.com/hoppxi/niv/pkgs/operation"
	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Control network connections",
	Run: func(cmd *cobra.Command, args []string) {
		if connect, _ := cmd.Flags().GetBool("connect"); connect {
			ssid, _ := cmd.Flags().GetString("ssid")
			password, _ := cmd.Flags().GetString("password")
			operation.Network.Connect(ssid, password)
		}
		if airplane, _ := cmd.Flags().GetBool("airplane"); airplane {
			operation.Network.AirplaneMode()
		}
		if disableWiFi, _ := cmd.Flags().GetBool("disable-wifi"); disableWiFi {
			operation.Network.DisableWiFi()
		}
		if scan, _ := cmd.Flags().GetBool("scan"); scan {
			if networks, err := operation.Network.ScanNetworks(); err !=nil {
				fmt.Println(networks)
			}
		}
	},
}

func init() {
	networkCmd.Flags().Bool("connect", false, "Connect to a network")
	networkCmd.Flags().String("ssid", "", "SSID for connection")
	networkCmd.Flags().String("password", "", "Password for connection")
	networkCmd.Flags().Bool("airplane", false, "Enable airplane mode")
	networkCmd.Flags().Bool("disable-wifi", false, "Disable WiFi")
	networkCmd.Flags().Bool("scan", false, "Scan available networks")
}
