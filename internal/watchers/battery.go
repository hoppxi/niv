package watchers

import (
	"fmt"
	"time"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/batteryinfo"
)

func StartBatteryWatcher(stop <-chan struct{}) {

	info, err := batteryinfo.GetBatteryInfo()
	if err != nil {
		fmt.Println(err)
	}
	updateEww("BATTERY_INFO", info)

	events := subscribe.BatteryEvents()

	for {
		select {
		case <-stop:
			return
		case <-events.BatteryFull:
			updateEwwNoJson("OSD_BATTERY_FULL", true)
			time.Sleep(5 * time.Second)
			updateEwwNoJson("OSD_BATTERY_FULL", false)

		case <-events.BatteryLow20:
			updateEwwNoJson("OSD_BATTERY_LOW_20", true)
			time.Sleep(5 * time.Second)
			updateEwwNoJson("OSD_BATTERY_LOW_20", false)
		case <-events.BatteryLow5:
			updateEwwNoJson("OSD_BATTERY_LOW_5", true)
			time.Sleep(5 * time.Second)
			updateEwwNoJson("OSD_BATTERY_LOW_5", false)
		case <-events.ChargerPlugged:
			updateEwwNoJson("OSD_CHARGER_PLUGGED", true)
			time.Sleep(5 * time.Second)
			updateEwwNoJson("OSD_CHARGER_PLUGGED", false)
		case <-events.ChargerUnplugged:
			updateEwwNoJson("OSD_CHARGER_UNPLUGGED", true)
			time.Sleep(5 * time.Second)
			updateEwwNoJson("OSD_CHARGER_UNPLUGGED", false)
		case <-events.DynamicChange:
			dynamicInfo, err := batteryinfo.GetBatteryDynamicInfo()
			if err != nil {
				fmt.Println(err)
			}

			updateEww("BATTERY_DYNAMIC_INFO", dynamicInfo)
		}
	}
}
