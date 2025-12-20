package cmd

import (
	"fmt"

	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/ncruces/zenity"
	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Control network connections",
	Run: func(cmd *cobra.Command, args []string) {
		if connect, _ := cmd.Flags().GetBool("connect"); connect {
			ssid, _ := cmd.Flags().GetString("ssid")
			_, password, err := zenity.Password(zenity.Title(ssid))
			switch {
			case err == nil:
				if err := operation.Network.Connect(ssid, password); err != nil {
					fmt.Println(err)
				}
			case err == zenity.ErrCanceled:
				println("Input cancelled.")
			default:
			}
		}
		if airplane, _ := cmd.Flags().GetBool("airplane"); airplane {
			operation.Network.AirplaneMode()
		}
		if disableWiFi, _ := cmd.Flags().GetBool("disable-wifi"); disableWiFi {
			operation.Network.DisableWiFi()
		}
		if toggleWiFi, _ := cmd.Flags().GetBool("toggle-wifi"); toggleWiFi {
			operation.Network.DisableWiFi()
			operation.Network.EnableWiFi()
		}
		if scan, _ := cmd.Flags().GetBool("scan"); scan {
			if networks, err := operation.Network.ScanNetworks(); err != nil {
				fmt.Println(networks)
			}
		}
	},
}

func init() {
	networkCmd.Flags().Bool("connect", false, "Connect to a network")
	networkCmd.Flags().String("ssid", "", "SSID for connection")
	networkCmd.Flags().Bool("airplane", false, "Enable airplane mode")
	networkCmd.Flags().Bool("disable-wifi", false, "Disable WiFi")
	networkCmd.Flags().Bool("toggle-wifi", false, "Toggle WiFi")
	networkCmd.Flags().Bool("scan", false, "Scan available networks")
}
