package cmd

import (
	"fmt"

	"github.com/hoppxi/niv/pkgs/bibleverse"
	"github.com/spf13/cobra"
)

var verseCmd = &cobra.Command{
	Use:   "verse",
	Short: "Get daily bible verse in compact JSON",
	Run: func(cmd *cobra.Command, args []string) {
		if v, err := bibleverse.GetDailyVerseJSON(); err != nil {
			fmt.Println("Unable to get verse")
		} else {
			fmt.Println(string(v)) // prints compact JSON string
		}
	},
}
