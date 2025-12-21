package watchers

import (
	"fmt"

	"github.com/hoppxi/wigo/internal/utils"
	"github.com/spf13/viper"

	_ "image/jpeg"
	_ "image/png"
)

func ConfigUpdate(v *viper.Viper) {
	updateEww("APPS_CONFIG", v.Get("apps"))

	general, ok := v.Get("general").(map[string]any)
	if !ok {
		fmt.Println("general config is not a map")
		return
	}

	src, ok := general["profile_pic"].(string)
	if !ok || src == "" {
		fmt.Println("profile_pic missing or invalid")
		return
	}

	croppped, err := utils.ObjectFitCover(src, 40, 40, "/tmp/wigo", "profile_pic_cropped")
	if err != nil {
		fmt.Println("profile pic processing failed:", err)
		return
	}

	rounded, err := utils.ApplyBorderRadius(croppped, 100, 100, 100, 100, "/tmp/wigo", "profile_pic_rounded")
	if err != nil {
		fmt.Println("profile pic processing failed:", err)
		return
	}

	general["profile_pic"] = rounded

	updateEww("GENERAL_CONFIG", general)
}
