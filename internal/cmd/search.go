package cmd

import (
	"github.com/hoppxi/wigo/internal/manager"
	"github.com/hoppxi/wigo/pkg/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search ':filter term'",
	Short: "search and do other stuff",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := manager.Config.Load()
		search.Search(args, cfg, "launcher-ext")
	},
}
