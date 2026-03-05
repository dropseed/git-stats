package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "All-in-one fetch, save, regen, push",
		RunE:  runCI,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().Bool("regen-missing", true, "Regenerate stats on commits missing them")
	cmd.Flags().String("git-name", "github-actions", "Git user.name for commits")
	cmd.Flags().String("git-email", "github-actions@github.com", "Git user.email for commits")
	rootCmd.AddCommand(cmd)
}

func runCI(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	regenMissing, _ := cmd.Flags().GetBool("regen-missing")
	gitName, _ := cmd.Flags().GetString("git-name")
	gitEmail, _ := cmd.Flags().GetString("git-email")

	cfg, err := config.Load("")
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	fmt.Println("\033[36mSetting git user.name and user.email...\033[0m")
	if _, err := git.Exec("config", "user.name", gitName); err != nil {
		return fmt.Errorf("setting git user.name: %w", err)
	}
	if _, err := git.Exec("config", "user.email", gitEmail); err != nil {
		return fmt.Errorf("setting git user.email: %w", err)
	}

	fmt.Println("\033[36mFetching stats from remote...\033[0m")
	if err := doFetch(false); err != nil {
		return err
	}

	fmt.Println("\n\033[36mSaving stats for the current commit...\033[0m")
	if err := doSave(keys, cfg); err != nil {
		return err
	}

	if regenMissing {
		fmt.Println("\n\033[36mRegenerating stats for last 10 commits if they are missing...\033[0m")
		if err := doRegen(keys, cfg, true, []string{"-n", "10"}, nil); err != nil {
			return err
		}
	}

	fmt.Println("\n\033[36mPushing stats to remote...\033[0m")
	if err := doPush(); err != nil {
		return err
	}

	return nil
}
