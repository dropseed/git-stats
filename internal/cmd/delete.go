package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/notes"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a stat from a commit",
		RunE:  runDelete,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to delete (all if not specified)")
	cmd.Flags().String("commitish", "HEAD", "Commit to delete stats from")
	cmd.Flags().String("config", "", "Path to config file (default: git-stats.yml in repo)")
	rootCmd.AddCommand(cmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	commitish, _ := cmd.Flags().GetString("commitish")
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	for _, key := range keys {
		if err := notes.DeleteKey(key, commitish); err != nil {
			fmt.Printf("Warning: could not delete %s: %v\n", key, err)
		}
	}

	_ = notes.Show(commitish)
	return nil
}
