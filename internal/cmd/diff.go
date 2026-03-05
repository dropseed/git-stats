package cmd

import (
	"fmt"
	"math"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/stats"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "diff <commit-a> <commit-b>",
		Short: "Compare stats between two commits",
		Long: `Show the difference in stats between two commits.

Useful for comparing branches, reviewing PR impact, or checking
progress over time. Shows the value at each commit and the change.`,
		Example: `  # Compare HEAD to the previous commit
  git stats diff HEAD~1 HEAD

  # Compare a feature branch to main
  git stats diff main feature-branch`,
		Args: cobra.ExactArgs(2),
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

	commitA := args[0]
	commitB := args[1]

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
