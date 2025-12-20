package cmd

import (
	"fmt"

	"github.com/hoppxi/wigo/pkg/audioinfo"
	"github.com/hoppxi/wigo/pkg/operation"
	"github.com/spf13/cobra"
)

var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "Control audio output/input or music",
	Run: func(cmd *cobra.Command, args []string) {
		if lvl, _ := cmd.Flags().GetInt("output-level"); lvl != 0 {
			operation.Audio.SetOutputLevel(lvl)
		}
		if mute, _ := cmd.Flags().GetBool("output-mute"); mute {
			operation.Audio.MuteOutput()
		}
		if lvl, _ := cmd.Flags().GetInt("input-level"); lvl != 0 {
			operation.Audio.SetInputLevel(lvl)
		}
		if mute, _ := cmd.Flags().GetBool("input-mute"); mute {
			operation.Audio.MuteInput()
		}
		if unmute, _ := cmd.Flags().GetBool("input-unmute"); unmute {
			operation.Audio.UnmuteInput()
		}
		if unmute, _ := cmd.Flags().GetBool("output-unmute"); unmute {
			operation.Audio.UnmuteOutput()
		}
		if toggleMute, _ := cmd.Flags().GetBool("input-toggle-mute"); toggleMute {
			operation.Audio.ToggleMuteInput()
		}
		if toggle, _ := cmd.Flags().GetBool("output-toggle-mute"); toggle {
			operation.Audio.ToggleMuteOutput()
		}
		if info, _ := cmd.Flags().GetBool("info"); info {
			info, err := audioinfo.GetAudioInfoJSON()
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(string(info))
		}
	},
}

func init() {
	audioCmd.Flags().Bool("info", false, "Output all current audio info in json format")

	audioCmd.Flags().Int("output-level", 0, "Set audio output level")
	audioCmd.Flags().Bool("output-mute", false, "Mute audio output")
	audioCmd.Flags().Bool("output-unmute", false, "Unmute audio output")
	audioCmd.Flags().Bool("output-toggle-mute", false, "Toggle mute state of audio output")

	audioCmd.Flags().Int("input-level", 0, "Set audio input level")
	audioCmd.Flags().Bool("input-mute", false, "Mute audio input")
	audioCmd.Flags().Bool("input-unmute", false, "Unmute audio input")
	audioCmd.Flags().Bool("input-toggle-mute", false, "Toggle mute state of audio input")

	audioCmd.Flags().String("media", "", "Media control: play, pause, next, previous, play-pause")
	audioCmd.Flags().Float64("media-pos", 0, "Set media position 0-100 in percent")
}
