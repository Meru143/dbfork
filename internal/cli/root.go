// Package cli defines the dbfork command tree.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dbfork",
	Short: "Git-style branching for local Postgres databases",
}

// Execute runs the root dbfork command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
