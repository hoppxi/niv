package watchers

import (
	"github.com/hoppxi/niv/internal/subscribe"
	"github.com/hoppxi/niv/pkgs/displayinfo"
)

func StartDisplayWatcher(stop <-chan struct{}) {
	info, _ := displayinfo.GetDisplayInfo()
	updateEww("DISPLAY_INFO", info)
	
	events := subscribe.DisplayEvents()

	for {
		select {
		case <-stop:
			return
		case <-events:
			info, _ := displayinfo.GetDisplayInfo()
			updateEww("DISPLAY_INFO", info)
		}
	}
}
