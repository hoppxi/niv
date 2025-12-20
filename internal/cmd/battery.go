package cmd

import (
	"fmt"
	"os"

	"github.com/hoppxi/wigo/pkg/batteryinfo"
	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var batteryCmd = &cobra.Command{
	Use:   "battery",
	Short: "Control system power profiles and view battery information",
	Run: func(cmd *cobra.Command, args []string) {

		powerProfile, _ := cmd.Flags().GetString("power-profile")
		if powerProfile != "" {
			switch powerProfile {
			case "performance", "balanced", "power-saver":
				err := operation.Battery.SetPowerMode(powerProfile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error setting power mode: %v\n", err)
				} else {
					fmt.Printf("Power profile set to: %s\n", powerProfile)
				}
			default:
				fmt.Println("Invalid profile. Please use: performance, balanced, or power-saver")
			}
		}

		showInfo, _ := cmd.Flags().GetBool("info")
		if showInfo {
			staticInfo, err := batteryinfo.GetBatteryInfoJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting battery info: %v\n", err)
			} else {
				fmt.Println(string(staticInfo))
			}

			dynamicInfo, err := batteryinfo.GetBatteryDynamicInfoJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting dynamic info: %v\n", err)
			} else {
				fmt.Println(string(dynamicInfo))
			}
		}
	},
}

func init() {
	batteryCmd.Flags().BoolP("info", "i", false, "Output battery info in JSON format")
	batteryCmd.Flags().StringP("power-profile", "p", "", "Set power profile (performance, balanced, power-saver)")
}
