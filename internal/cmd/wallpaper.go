package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hoppxi/niv/internal/wallpaper"
	"github.com/spf13/cobra"
)


var wallpaperCmd = &cobra.Command{
	Use:   "wallpaper",
	Short: "Wallpaper command for niv to enable you change and do other stuff related to wallpaper",
	Run: func(cmd *cobra.Command, args []string) {

		if next, _ := cmd.Flags().GetBool("next"); next {
			wallpaper.NextWallpapers()
		}
		if previous, _ := cmd.Flags().GetBool("previous"); previous {
			wallpaper.PreviousWallpapers()
		}
		if set, _ := cmd.Flags().GetString("set"); set != "" {
			wallpaper.SetWallpaper(set)
		}
		if random, _ := cmd.Flags().GetBool("random"); random {
			out, err := exec.Command("eww", "get", "EWW_CONFIG_DIR").Output()
			if err != nil {
				fmt.Println("Cannot get EWW_CONFIG_DIR:", err)
			}
			cfgDir := strings.TrimSpace(string(out))
			nivConf := filepath.Join(cfgDir, "niv.yaml")

			wallpaper.SetRandomWallpaper(nivConf)
		}
	},
}


func init() {
	wallpaperCmd.Flags().Bool("next", false, "To get the next five wallpaper")
	wallpaperCmd.Flags().Bool("previous", false, "To get the previous five wallpaper")
	wallpaperCmd.Flags().String("set", "", "To set wallpaper")
	wallpaperCmd.Flags().Bool("random", false, "To set random wallpaper")
}
