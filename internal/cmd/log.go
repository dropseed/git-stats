package cmd

import (
	"fmt"

	"github.com/dropseed/git-stats/internal/config"
	"github.com/dropseed/git-stats/internal/stats"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "log [flags] [-- git log args...]",
		Short: "Log stats for commits matching git log args",
		RunE:  runLog,
		Args:  cobra.ArbitraryArgs,
	}
	cmd.Flags().StringSliceP("key", "k", nil, "Stats to include (all if not specified)")
	cmd.Flags().Bool("values-only", false, "Only print values (no headers)")
	cmd.Flags().String("format", "pretty", "Output format: pretty, table, tsv, csv, json, sparklines")
	cmd.Flags().String("config", "", "Path to config file (default: git-stats.yml in repo)")
	rootCmd.AddCommand(cmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	keys, _ := cmd.Flags().GetStringSlice("key")
	valuesOnly, _ := cmd.Flags().GetBool("values-only")
	format, _ := cmd.Flags().GetString("format")
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	keys = config.ResolveKeys(keys, cfg)

	s, err := stats.Load(keys, cfg, true, args)
	if err != nil {
		return err
	}

	switch format {
	case "table":
		s.PrintTable(valuesOnly)
	case "pretty":
		s.PrintPretty(valuesOnly)
	case "tsv":
		s.Print(valuesOnly, "\t")
	case "csv":
		s.Print(valuesOnly, ",")
	case "json":
		s.PrintJSON()
	case "sparklines":
		s.Sparklines()
	default:
		return fmt.Errorf("unknown format %q (valid: pretty, table, tsv, csv, json, sparklines)", format)
	}

	return nil
}
