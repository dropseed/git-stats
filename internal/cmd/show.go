package cmd

import (
	"github.com/dropseed/git-stats/internal/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "show [commitish]",
		Short: "Show stats for a commit",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commitish := "HEAD"
			if len(args) > 0 {
				commitish = args[0]
			}
			return notes.Show(commitish)
		},
	}
	rootCmd.AddCommand(cmd)
}
