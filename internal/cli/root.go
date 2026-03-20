// Package cli defines the dbfork command tree.
package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if err := loadRootConfig(cmd); err != nil {
			cobra.CheckErr(err)
		}
	}
}

// Execute runs the root dbfork command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadRootConfig(cmd *cobra.Command) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(filepath.Join(homeDir, ".dbfork"))

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	}

	if cmd.Flags().Changed("host") {
		viper.Set("default_host", cmd.Flags().Lookup("host").Value.String())
	}
	if cmd.Flags().Changed("port") {
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}
		viper.Set("default_port", port)
	}
	if cmd.Flags().Changed("user") {
		viper.Set("default_user", cmd.Flags().Lookup("user").Value.String())
	}
	if cmd.Flags().Changed("password") {
		viper.Set("default_password", cmd.Flags().Lookup("password").Value.String())
	}
	if cmd.Flags().Changed("source") {
		viper.Set("default_database", cmd.Flags().Lookup("source").Value.String())
	}
	if cmd.Flags().Changed("verbose") {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		viper.Set("verbose", verbose)
	}

	return nil
}
