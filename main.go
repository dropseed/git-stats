package main

import (
	"os"

	"github.com/dropseed/git-stats/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
