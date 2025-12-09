package watchers

import (
	"github.com/hoppxi/niv/internal/subscribe"
	"github.com/hoppxi/niv/pkgs/netinfo"
	"github.com/hoppxi/niv/pkgs/iconsinfo"
)

func StartNetworkWatcher(stop <-chan struct{}) {
	info, _ := netinfo.GetNetworkInfo()
	updateEww("NETWORK_INFO", info)
	
	iconsInfo := iconsinfo.GetIcons()
	updateEww("ICONS_INFO", iconsInfo)
	
	events := subscribe.NetworkEvents()

	for {
		select {
		case <-stop:
			return
		case <-events:
			info, _ := netinfo.GetNetworkInfo()
			updateEww("NETWORK_INFO", info)

			iconsInfo := iconsinfo.GetIcons()
			updateEww("ICONS_INFO", iconsInfo)
		}
	}
}
