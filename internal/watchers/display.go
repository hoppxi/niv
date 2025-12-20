package watchers

import (
	"time"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/displayinfo"
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

			go func() {
				updateEwwNoJson("OSD_DISPLAY", true)
				time.Sleep(5 * time.Second)
				updateEwwNoJson("OSD_DISPLAY", false)
			}()
		}
	}
}
