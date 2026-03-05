package cmd

import "github.com/spf13/cobra"

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "git-stats",
	Short:   "Store commit stats as git notes",
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Skip confirmation prompts")
}

func Execute() error {
	return rootCmd.Execute()
}
