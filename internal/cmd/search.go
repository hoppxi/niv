package cmd

import (
	"github.com/hoppxi/niv/pkgs/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search ':filter term'",
	Short: "saech and do other stuff",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		search.Search(args)
	},
}
