package watchers

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/hoppxi/niv/internal/subscribe"
	"github.com/hoppxi/niv/pkgs/audioinfo"
	"github.com/hoppxi/niv/pkgs/iconsinfo"
)

func StartAudioWatcher(stop <-chan struct{}) {
	info, _ := audioinfo.GetAudioInfo()
	updateEww("AUDIO_INFO", info)

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
		}
	}
}

func updateEww(module string, data any) {
	jsonData, _ := json.Marshal(data)
	cmd := exec.Command("eww", "update", module+"="+string(jsonData))
	_ = cmd.Start()
}

func updateEwwNoJson(module string, data any) {
	cmd := exec.Command("eww", "update", fmt.Sprintf("%s=%v", module, data))
	_ = cmd.Start()
}
