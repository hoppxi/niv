package watchers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/hoppxi/wigo/internal/subscribe"
	"github.com/hoppxi/wigo/pkg/audioinfo"
	"github.com/hoppxi/wigo/pkg/iconsinfo"
)

func StartAudioWatcher(stop <-chan struct{}) {
	prev, _ := audioinfo.GetAudioInfo()
	updateEww("AUDIO_INFO", prev)

	iconInfo := iconsinfo.GetIcons()
	updateEww("ICONS_INFO", iconInfo)

	events := subscribe.AudioEvents()

	for {
		select {
		case <-stop:
			return
		case <-events:
			info, _ := audioinfo.GetAudioInfo()
			updateEww("AUDIO_INFO", info)

			iconInfo := iconsinfo.GetIcons()
			updateEww("ICONS_INFO", iconInfo)
			go func() {
				if prev.Output.Level != info.Output.Level {
					updateEwwNoJson("OSD_VOLUME", true)
					time.Sleep(5 * time.Second)
					updateEwwNoJson("OSD_VOLUME", false)

					prev = info
				}
			}()

		}
	}
}

func updateEww(module string, data any) {
	jsonData, _ := json.Marshal(data)
	exec.Command("eww", "update", module+"="+string(jsonData)).Run()
}

func updateEwwNoJson(module string, data any) {
	exec.Command("eww", "update", fmt.Sprintf("%s=%v", module, data)).Run()
}
