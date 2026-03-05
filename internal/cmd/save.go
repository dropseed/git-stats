package cmd

import (
	"fmt"
	"os"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/dropseed/git-stats/internal/github"
	"github.com/dropseed/git-stats/internal/notes"
	"github.com/dropseed/git-stats/internal/stats"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save stats for the current commit",
		RunE:  runSave,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	rootCmd.AddCommand(cmd)
}

func runSave(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	return doSave(config.ResolveKeys(keys, cfg), cfg)
}

func doSave(keys []string, cfg *config.Config) error {
	commitish := "HEAD"

	for _, key := range keys {
		command, err := cfg.CommandForStat(key)
		if err != nil {
			return err
		}

		fmt.Printf("Generating value for %s: ", key)
		value, err := git.RunShell(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31merror: %v\033[0m\n", err)
			continue
		}

		if value != "" {
			fmt.Printf("\033[32m%s\033[0m\n", value)
			if err := notes.Save(key, value, commitish); err != nil {
				return fmt.Errorf("saving note for %s: %w", key, err)
			}

			if github.IsCI() {
				goal := cfg.GoalForStat(key)
				if goal != "" {
					reportGoalStatus(key, value, goal, cfg)
				}
			}
		} else {
			fmt.Printf("\033[33mSkipping empty value for %s\033[0m\n", key)
		}
	}

	fmt.Printf("\n\033[1mStats for %s:\033[0m\n", commitish)
	return notes.Show(commitish)
}

func reportGoalStatus(key, currentValueStr, goal string, cfg *config.Config) {
	stat := stats.NewCommitStat(cfg.TypeForStat(key))
	currentValue, err := stat.ParseValue(currentValueStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse current value for goal comparison: %v\n", err)
		return
	}

	prevStats, err := stats.Load([]string{key}, cfg, false, []string{"-n", "1", "--skip", "1"})
	if err != nil || len(prevStats.Commits) == 0 {
		if err := github.ReportStatus(key, currentValue, 0, false, goal, stat); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not report GitHub status: %v\n", err)
		}
		return
	}

	previousValue := 0.0
	hasPrevious := false
	prevCommit := prevStats.Commits[0]
	if prevStat, ok := prevStats.Stats[key]; ok {
		if v, has := prevStat.Get(prevCommit); has {
			previousValue = v
			hasPrevious = true
		}
	}

	if err := github.ReportStatus(key, currentValue, previousValue, hasPrevious, goal, stat); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not report GitHub status: %v\n", err)
	}
}
