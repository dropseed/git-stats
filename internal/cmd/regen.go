package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/dropseed/git-stats/internal/notes"
	"github.com/dropseed/git-stats/internal/stats"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "regen [flags] [-- git log args...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Regenerate stats for commits matching git log args",
		RunE:  runRegen,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().Bool("missing-only", false, "Only regenerate missing stats")
	rootCmd.AddCommand(cmd)
}

func runRegen(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	missingOnly, _ := cmd.Flags().GetBool("missing-only")

	cfg, err := config.Load("")
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	return doRegen(keys, cfg, missingOnly, args)
}

func doRegen(keys []string, cfg *config.Config, missingOnly bool, gitLogArgs []string) error {
	currentBranch, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	status, err := git.Output("status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("working tree is not clean; commit or stash your changes first")
	}

	s, err := stats.Load(keys, cfg, false, gitLogArgs)
	if err != nil {
		return err
	}

	commits := s.Commits

	if missingOnly {
		totalCommits := len(commits)
		commits = s.CommitsMissingStats(keys)
		if len(commits) == 0 {
			fmt.Println("\033[32mNo missing stats!\033[0m")
			return nil
		}
		prompt := fmt.Sprintf("Regenerate %v stats for %d of %d commits?", keys, len(commits), totalCommits)
		if !confirmPrompt(prompt) {
			return nil
		}
	} else {
		prompt := fmt.Sprintf("Regenerate %v stats for %d commits?", keys, len(commits))
		if !confirmPrompt(prompt) {
			return nil
		}
	}

	defer func() {
		_, _ = git.Exec("checkout", currentBranch)
	}()

	for i, commit := range commits {
		if isTerminal() {
			fmt.Fprintf(os.Stderr, "\r[%d/%d] %s", i+1, len(commits), commit[:12])
		} else {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", i+1, len(commits), commit[:12])
		}

		_, err := git.Exec("checkout", commit)
		if err != nil {
			return fmt.Errorf("failed to checkout %s: %w", commit, err)
		}

		for _, key := range keys {
			if missingOnly && s.CommitHasStat(commit, key) {
				continue
			}

			command, err := cfg.CommandForStat(key)
			if err != nil {
				return err
			}

			value, err := git.RunShell(command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: command failed for %s on %s: %v\n", key, commit[:12], err)
				continue
			}

			if value != "" {
				if err := notes.Save(key, value, commit); err != nil {
					return fmt.Errorf("saving note for %s on %s: %w", key, commit, err)
				}
			}
		}
	}
	if isTerminal() {
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func confirmPrompt(prompt string) bool {
	if !isTerminal() {
		return true
	}

	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
