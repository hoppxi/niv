package watchers

import (
	"github.com/hoppxi/niv/internal/subscribe"
	"github.com/hoppxi/niv/pkgs/btinfo"
	"github.com/hoppxi/niv/pkgs/iconsinfo"
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
