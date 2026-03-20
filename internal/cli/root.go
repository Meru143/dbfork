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

func init() {
	flags := rootCmd.PersistentFlags()
	flags.String("host", "localhost", "Postgres host")
	flags.Int("port", 5432, "Postgres port")
	flags.String("user", "", "Postgres user")
	flags.String("password", "", "Postgres password")
	flags.String("source", "", "Source database")
	flags.Bool("verbose", false, "Show SQL queries executed")
}

// Execute runs the root dbfork command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
