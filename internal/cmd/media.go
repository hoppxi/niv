package cmd

import (
	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Control media playback",
	Run: func(cmd *cobra.Command, args []string) {
		if play, _ := cmd.Flags().GetBool("play"); play {
			operation.Media.ControlMedia("play")
		}
		if pause, _ := cmd.Flags().GetBool("pause"); pause {
			operation.Media.ControlMedia("pause")
		}
		if playPause, _ := cmd.Flags().GetBool("play-pause"); playPause {
			operation.Media.ControlMedia("play-pause")
		}
		if next, _ := cmd.Flags().GetBool("next"); next {
			operation.Media.ControlMedia("next")
		}
		if previous, _ := cmd.Flags().GetBool("previous"); previous {
			operation.Media.ControlMedia("previous")
		}
		if position, err := cmd.Flags().GetFloat64("position"); err == nil {
			operation.Media.SetMediaPosition(position)
		}
	},
}

func init() {
	mediaCmd.Flags().Bool("play", false, "Play media")
	mediaCmd.Flags().Bool("pause", false, "pause media")
	mediaCmd.Flags().Bool("play-pause", false, "toggle play state of media")
	mediaCmd.Flags().Bool("previous", false, "play previous media")
	mediaCmd.Flags().Bool("next", false, "play next media")
	mediaCmd.Flags().Float64("position", 0, "Set media position 0-100 in percent")
}
