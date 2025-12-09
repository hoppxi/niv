package watchers

import (
	"time"

	"github.com/hoppxi/niv/pkgs/miscinfo"
)

func StartMiscWatcher(stop <-chan struct{}) {
	info := miscinfo.GetMisc()
	updateEww("MISC_INFO", info)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			info := miscinfo.GetMisc()
				updateEww("MISC_INFO", info)
		}
	}
}

