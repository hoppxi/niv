package watchers

import (
	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/btinfo"
	"github.com/hoppxi/wigo/pkg/iconsinfo"
)

func StartBluetoothWatcher(stop <-chan struct{}) {
	info, _ := btinfo.GetBluetoothInfo()
	updateEww("BLUETOOTH_INFO", info)

	iconsInfo := iconsinfo.GetIcons()
	updateEww("ICONS_INFO", iconsInfo)

	events := subscribe.BluetoothEvents()

	for {
		select {
		case <-stop:
			return
		case <-events:
			info, _ := btinfo.GetBluetoothInfo()
			updateEww("BLUETOOTH_INFO", info)

			iconsInfo := iconsinfo.GetIcons()
			updateEww("ICONS_INFO", iconsInfo)
		}
	}
}
