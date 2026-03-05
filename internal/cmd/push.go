package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push stats to remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doPush()
		},
	}
	rootCmd.AddCommand(cmd)
}

func doPush() error {
	if err := notes.Push(); err != nil {
		fmt.Println("\033[33mHave you created any stats yet?\033[0m")
		return err
	}
	return nil
}
