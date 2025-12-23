package watchers

import (
	"github.com/hoppxi/wigo/internal/subscribe"
)

func StartLEDsWatcher(stop <-chan struct{}) {
	events := subscribe.LEDsEvents()

	for {
		select {
		case <-stop:
			return
		case <-events.CapsOff:
			updateEwwNoJson("OSD_CAPS", false)
		case <-events.CapsOn:
			updateEwwNoJson("OSD_CAPS", true)
		case <-events.NumOff:
			updateEwwNoJson("OSD_NUM", false)
		case <-events.NumOn:
			updateEwwNoJson("OSD_NUM", true)
		case <-events.ScrollOff:
			updateEwwNoJson("OSD_SCROLL", false)
		case <-events.ScrollOn:
			updateEwwNoJson("OSD_SCROLL", true)
		}
	}
}
