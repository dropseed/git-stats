package cmd

import (
	"fmt"
	"math"
	"strings"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/stats"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "diff [<commit-a>] [<commit-b>]",
		Short: "Compare stats between two commits",
		Long: `Show the difference in stats between two commits.

Useful for comparing branches, reviewing PR impact, or checking
progress over time. Shows the value at each commit and the change.

Accepts git diff-style syntax:
  git stats diff                   # HEAD~1 vs HEAD
  git stats diff main              # main vs HEAD
  git stats diff main feature      # main vs feature
  git stats diff main..feature     # main vs feature`,
		Example: `  # Compare HEAD to the previous commit
  git stats diff

  # Compare main to HEAD
  git stats diff main

  # Compare two branches
  git stats diff main feature-branch
  git stats diff main..feature-branch`,
		Args: cobra.MaximumNArgs(2),
		RunE: runDiff,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().String("config", "", "Path to config file (default: git-stats.yml in repo)")
	rootCmd.AddCommand(cmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	var commitA, commitB string
	switch len(args) {
	case 0:
		commitA = "HEAD~1"
		commitB = "HEAD"
	case 1:
		// Support "main..feature" syntax
		if parts := strings.SplitN(args[0], "..", 2); len(parts) == 2 {
			commitA = parts[0]
			commitB = parts[1]
			if commitB == "" {
				commitB = "HEAD"
			}
		} else {
			commitA = args[0]
			commitB = "HEAD"
		}
	case 2:
		commitA = args[0]
		commitB = args[1]
	}

	sA, err := stats.Load(keys, cfg, true, []string{commitA, "-n", "1"})
	if err != nil {
		return fmt.Errorf("loading stats for %s: %w", commitA, err)
	}
	sB, err := stats.Load(keys, cfg, true, []string{commitB, "-n", "1"})
	if err != nil {
		return fmt.Errorf("loading stats for %s: %w", commitB, err)
	}

	if len(sA.Commits) == 0 {
		return fmt.Errorf("no commit found for %s", commitA)
	}
	if len(sB.Commits) == 0 {
		return fmt.Errorf("no commit found for %s", commitB)
	}

	hashA := sA.Commits[0]
	hashB := sB.Commits[0]
	shortA := sA.ShortHash[hashA]
	shortB := sB.ShortHash[hashB]

	// Calculate column widths
	keyWidth := len("stat")
	valAWidth := len(shortA)
	valBWidth := len(shortB)

	type row struct {
		key    string
		valA   string
		valB   string
		change string
	}

	var rows []row
	for _, key := range keys {
		if len(key) > keyWidth {
			keyWidth = len(key)
		}

		var valAStr, valBStr, changeStr string

		statA := sA.Stats[key]
		statB := sB.Stats[key]

		vA, okA := statA.Get(hashA)
		vB, okB := statB.Get(hashB)

		if okA {
			valAStr = statA.FormatValue(vA)
		}
		if okB {
			valBStr = statB.FormatValue(vB)
		}

		if okA && okB {
			delta := vB - vA
			if delta == 0 {
				changeStr = "="
			} else {
				sign := "+"
				if delta < 0 {
					sign = ""
				}
				if statA.Type == "%" {
					changeStr = fmt.Sprintf("%s%g%%", sign, math.Round(delta*100)/100)
				} else if delta == math.Trunc(delta) {
					changeStr = fmt.Sprintf("%s%d", sign, int(delta))
				} else {
					changeStr = fmt.Sprintf("%s%g", sign, math.Round(delta*100)/100)
				}
			}
		}

		if len(valAStr) > valAWidth {
			valAWidth = len(valAStr)
		}
		if len(valBStr) > valBWidth {
			valBWidth = len(valBStr)
		}

		rows = append(rows, row{key, valAStr, valBStr, changeStr})
	}

	changeWidth := len("change")
	for _, r := range rows {
		if len(r.change) > changeWidth {
			changeWidth = len(r.change)
		}
	}

	// Header
	fmt.Printf("\033[1m%-*s  %*s  %*s  %*s\033[0m\n", keyWidth, "stat", valAWidth, shortA, valBWidth, shortB, changeWidth, "change")

	// Rows
	for _, r := range rows {
		changeColor := "\033[2m" // dim for "="
		resetColor := "\033[0m"
		if len(r.change) > 0 && r.change != "=" {
			if r.change[0] == '+' {
				changeColor = "\033[32m"
			} else {
				changeColor = "\033[31m"
			}
		}
		fmt.Printf("%-*s  \033[36m%*s\033[0m  \033[36m%*s\033[0m  %s%*s%s\n", keyWidth, r.key, valAWidth, r.valA, valBWidth, r.valB, changeColor, changeWidth, r.change, resetColor)
	}

	return nil
}
