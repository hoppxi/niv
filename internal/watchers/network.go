package watchers

import (
	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/iconsinfo"
	"github.com/hoppxi/wigo/pkg/netinfo"
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
