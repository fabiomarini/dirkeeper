package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = cobra.Command{
	Use:   "dirkeeper",
	Short: "Directory management utilities",
}

func init() {
	RootCmd.AddCommand(CleanOldCmd)
	RootCmd.AddCommand(MatchCmd)
	RootCmd.AddCommand(WatchCmd)
	RootCmd.AddCommand(FreeSpaceCmd)
}

func Execute() error {
	return RootCmd.Execute()
}
