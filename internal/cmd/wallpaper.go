package cmd

import (
	"github.com/hoppxi/wigo/internal/manager"
	"github.com/hoppxi/wigo/internal/wallpaper"
	"github.com/spf13/cobra"
)

var wallpaperCmd = &cobra.Command{
	Use:   "wallpaper",
	Short: "Wallpaper command for wigo to enable you change and do other stuff related to wallpaper",
	Run: func(cmd *cobra.Command, args []string) {

		if set, _ := cmd.Flags().GetString("set"); set != "" {
			wallpaper.SetWallpaper(set)
		}
		if random, _ := cmd.Flags().GetBool("random"); random {
			cfg := manager.Config.Load()
			wallpaper.SetRandomWallpaper(cfg)
		}
		if selectSet, _ := cmd.Flags().GetBool("select"); selectSet {
			wallpaper.SelectAndSetWallpaper()
		}
	},
}

func init() {
	wallpaperCmd.Flags().String("set", "", "To set wallpaper")
	wallpaperCmd.Flags().Bool("random", false, "To set random wallpaper")
	wallpaperCmd.Flags().Bool("select", false, "Select and set from file manager")
}
