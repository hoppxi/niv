package watchers

import (
	"time"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/displayinfo"
)

func StartDisplayWatcher(stop <-chan struct{}) {
	prev, _ := displayinfo.GetDisplayInfo()
	updateEww("DISPLAY_INFO", prev)

	events := subscribe.DisplayEvents()
	var osdTimer *time.Timer

	for {
		select {
		case <-stop:
			return
		case <-events:
			info, _ := displayinfo.GetDisplayInfo()
			updateEww("DISPLAY_INFO", info)

			if prev.Level != info.Level {

				updateEwwNoJson("OSD_DISPLAY", true)
				if osdTimer != nil {
					osdTimer.Stop()
				}

				osdTimer = time.AfterFunc(3*time.Second, func() {
					updateEwwNoJson("OSD_DISPLAY", false)
				})

				prev = info
			}
		}
	}
}
