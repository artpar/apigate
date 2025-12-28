package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Set via ldflags at build time
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("apigate %s\n", version)
		fmt.Printf("  commit:  %s\n", commit)
		fmt.Printf("  built:   %s\n", buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
