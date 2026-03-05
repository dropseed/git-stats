package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		Long: `Regenerate stats for historical commits by checking out each commit,
running the stat commands from your config, and saving the results as git notes.

This is useful for backfilling stats when you first add git-stats to a repo,
or when you add a new stat key and want to populate it for old commits.

How it works:
  1. Gets the list of commits matching the git log args
  2. Checks out each commit in turn
  3. Runs the configured stat commands
  4. Saves results as git notes
  5. Returns to your original branch

Untracked files are preserved through checkouts — only uncommitted changes
to tracked files will block regen.

Use --config to point at a config file outside the repo (useful when the
config doesn't exist in older commits). Use --keep to preserve files like
scripts or config through checkouts (they get copied to a temp dir and
restored after each checkout).

Pass git log arguments after -- to control which commits are regenerated.`,
		Example: `  # Regenerate stats for the last 50 commits
  git stats regen -- -n 50

  # Only fill in commits that are missing stats
  git stats regen --missing-only -- -n 50

  # Use a config file that doesn't exist in old commits
  git stats regen --config /tmp/git-stats.yml -- -n 20

  # Keep scripts and config through checkouts
  git stats regen --keep scripts/bench,git-stats.yml -- -n 20

  # Regenerate a specific stat
  git stats regen -k loc -- -n 50`,
		RunE: runRegen,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().Bool("missing-only", false, "Only regenerate missing stats")
	cmd.Flags().String("config", "", "Path to config file (default: git-stats.yml in repo)")
	cmd.Flags().StringSlice("keep", nil, "Files to preserve through checkouts (e.g. scripts/bench)")
	rootCmd.AddCommand(cmd)
}

func runRegen(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	missingOnly, _ := cmd.Flags().GetBool("missing-only")
	configPath, _ := cmd.Flags().GetString("config")
	keepFiles, _ := cmd.Flags().GetStringSlice("keep")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	return doRegen(cmd, keys, cfg, missingOnly, args, keepFiles)
}

type regenSummary struct {
	saved   int
	skipped int
	failed  int
	empty   int
}

func doRegen(cmd *cobra.Command, keys []string, cfg *config.Config, missingOnly bool, gitLogArgs []string, keepFiles []string) error {
	for _, f := range keepFiles {
		if filepath.IsAbs(f) {
			return fmt.Errorf("--keep paths must be relative: %s", f)
		}
	}

	currentBranch, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if err := checkTrackedFilesClean(); err != nil {
		return err
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
		if !confirmPrompt(cmd, prompt) {
			return nil
		}
	} else {
		prompt := fmt.Sprintf("Regenerate %v stats for %d commits?", keys, len(commits))
		if !confirmPrompt(cmd, prompt) {
			return nil
		}
	}

	// Set up keep files: copy to temp dir so we can restore after each checkout
	var keepDir string
	if len(keepFiles) > 0 {
		keepDir, err = os.MkdirTemp("", "git-stats-keep-*")
		if err != nil {
			return fmt.Errorf("creating temp dir for --keep files: %w", err)
		}
		defer func() { _ = os.RemoveAll(keepDir) }()

		for _, f := range keepFiles {
			if err := copyFileToDir(f, keepDir); err != nil {
				return fmt.Errorf("backing up --keep file %q: %w", f, err)
			}
		}
	}

	defer func() {
		if len(keepFiles) > 0 {
			// Force checkout to avoid conflicts from restored keep files
			_, _ = git.Exec("checkout", "-f", currentBranch)
		} else {
			_, _ = git.Exec("checkout", currentBranch)
		}
	}()

	summary := regenSummary{}
	var failedCommits []string

	for i, commit := range commits {
		if isTerminal() {
			fmt.Fprintf(os.Stderr, "\r\033[K[%d/%d] %s", i+1, len(commits), commit[:12])
		} else {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", i+1, len(commits), commit[:12])
		}

		checkoutArgs := []string{"checkout"}
		if len(keepFiles) > 0 {
			checkoutArgs = append(checkoutArgs, "-f")
		}
		checkoutArgs = append(checkoutArgs, commit)

		_, err := git.Exec(checkoutArgs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to checkout %s, skipping: %v\n", commit[:12], err)
			failedCommits = append(failedCommits, commit[:12])
			continue
		}

		// Restore keep files after checkout
		if len(keepFiles) > 0 {
			for _, f := range keepFiles {
				if err := restoreFileFromDir(f, keepDir); err != nil {
					fmt.Fprintf(os.Stderr, "\nWarning: failed to restore --keep file %q: %v\n", f, err)
				}
			}
		}

		for _, key := range keys {
			if missingOnly && s.CommitHasStat(commit, key) {
				summary.skipped++
				continue
			}

			command, err := cfg.CommandForStat(key)
			if err != nil {
				return err
			}

			value, err := git.RunShell(command)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: %s failed on %s: %v\n", key, commit[:12], err)
				summary.failed++
				continue
			}

			if value == "" {
				summary.empty++
				continue
			}

			if err := notes.Save(key, value, commit); err != nil {
				return fmt.Errorf("saving note for %s on %s: %w", key, commit, err)
			}
			summary.saved++
		}
	}

	if isTerminal() {
		fmt.Fprintln(os.Stderr)
	}

	// Print summary
	fmt.Fprintf(os.Stderr, "\n\033[1mRegen complete:\033[0m %d saved", summary.saved)
	if summary.skipped > 0 {
		fmt.Fprintf(os.Stderr, ", %d skipped", summary.skipped)
	}
	if summary.failed > 0 {
		fmt.Fprintf(os.Stderr, ", \033[31m%d failed\033[0m", summary.failed)
	}
	if summary.empty > 0 {
		fmt.Fprintf(os.Stderr, ", %d empty", summary.empty)
	}
	fmt.Fprintln(os.Stderr)

	if len(failedCommits) > 0 {
		fmt.Fprintf(os.Stderr, "\033[33mFailed to checkout: %s\033[0m\n", strings.Join(failedCommits, ", "))
	}

	return nil
}

// checkTrackedFilesClean verifies that tracked files have no uncommitted changes.
// Untracked files are ignored since they survive checkout safely.
func checkTrackedFilesClean() error {
	status, err := git.Output("status", "--porcelain")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(status, "\n") {
		if line == "" {
			continue
		}
		// Ignore untracked (??) and ignored (!!) files
		if strings.HasPrefix(line, "?? ") || strings.HasPrefix(line, "!! ") {
			continue
		}
		return fmt.Errorf("working tree has uncommitted changes to tracked files; commit or stash first")
	}
	return nil
}

func confirmPrompt(cmd *cobra.Command, prompt string) bool {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes || !isTerminal() {
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

func copyFileToDir(relPath string, dir string) error {
	dst := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	src, err := os.Open(relPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, src)
	return err
}

func restoreFileFromDir(relPath string, dir string) error {
	src := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(relPath), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(relPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
