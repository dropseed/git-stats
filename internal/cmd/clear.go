package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete all existing stats",
		RunE:  runClear,
	}
	cmd.Flags().Bool("remote", false, "Clear remote stats instead of local")
	rootCmd.AddCommand(cmd)
}

func runClear(cmd *cobra.Command, args []string) error {
	remote, _ := cmd.Flags().GetBool("remote")

	if !confirmPrompt("Are you sure you want to clear all existing stats?") {
		return nil
	}

	if err := notes.Clear(remote); err != nil {
		return err
	}

	if remote {
		fmt.Println("\033[32mRemote stats cleared\033[0m")
	} else {
		fmt.Println("\033[32mLocal stats cleared\033[0m")
	}

	return nil
}
