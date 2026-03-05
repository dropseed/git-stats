package cmd

import (
	"fmt"
	"os"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run stat commands and print values without saving",
		RunE:  runCheck,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().String("config", "", "Path to config file (default: git-stats.yml in repo)")
	rootCmd.AddCommand(cmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

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
		} else {
			fmt.Printf("\033[33mSkipping empty value for %s\033[0m\n", key)
		}
	}

	return nil
}
