package watchers

import (
	"time"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/mediainfo"
)

func StartMediaWatcher(stop <-chan struct{}) {
	events := subscribe.MediaEvents()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var currentPos int64 = 0
	var totalLen int64 = 0
	var isPlaying bool = false

	syncState := func() {
		info, _ := mediainfo.GetMediaInfo()

		currentPos = info.PositionRaw
		totalLen = info.LengthRaw
		isPlaying = info.Playing

		updateEww("MEDIA_INFO", info)
		updateTime(currentPos, totalLen)
	}

	syncState()

	for {
		select {
		case <-stop:
			return
		case <-events:
			syncState()
		case <-ticker.C:
			if isPlaying {
				currentPos += 1_000_000
				if currentPos > totalLen {
					currentPos = totalLen
				}
				updateTime(currentPos, totalLen)
			}
		}
	}
}

func updateTime(pos, length int64) {
	elapsedStr := mediainfo.FormatDurationMicros(pos)
	normFloat := mediainfo.FormatNormalized(pos, length)
	updateEwwNoJson("MEDIA_ELAPSED_TIME", elapsedStr)
	updateEwwNoJson("MEDIA_NORMALIZED_ELAPSED_TIME", normFloat)
}
