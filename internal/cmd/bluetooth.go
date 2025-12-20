package cmd

import (
	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var bluetoothCmd = &cobra.Command{
	Use:   "bluetooth",
	Short: "Control bluetooth",
	Run: func(cmd *cobra.Command, args []string) {
		if disable, _ := cmd.Flags().GetBool("disable"); disable {
			operation.Bluetooth.Disable()
		}
		if connect, _ := cmd.Flags().GetBool("connect"); connect {
			macAddress, _ := cmd.Flags().GetString("mac")
			operation.Bluetooth.Connect(macAddress)
		}
		if scan, _ := cmd.Flags().GetBool("scan"); scan {
			operation.Bluetooth.Scan()
		}
	},
}

func init() {
	bluetoothCmd.Flags().Bool("disable", false, "Disable Bluetooth")
	bluetoothCmd.Flags().Bool("connect", false, "Connect to a device")
	bluetoothCmd.Flags().String("device", "", "Device ID or name to connect")
	bluetoothCmd.Flags().Bool("scan", false, "Scan Bluetooth devices")
}
