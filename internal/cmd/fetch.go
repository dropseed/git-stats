package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch stats from remote",
		RunE:  runFetch,
	}
	cmd.Flags().BoolP("force", "f", false, "Force overwrite local notes")
	rootCmd.AddCommand(cmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	return doFetch(force)
}

func doFetch(force bool) error {
	err := notes.Fetch(force)
	if err != nil {
		fmt.Println("\033[33mNote: no remote stats found (this is normal on first run)\033[0m")
		return nil
	}
	return nil
}
