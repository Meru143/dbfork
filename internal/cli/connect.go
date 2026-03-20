// Package cli provides the dbfork connect command.
package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

var connectCmd = &cobra.Command{
	Use:   "connect <name>",
	Short: "Print a branch connection string or open psql",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}

		psql, err := cmd.Flags().GetBool("psql")
		if err != nil {
			return err
		}

		return runConnect(cmd.Context(), connectDeps{
			loadConfig: func() (config.Config, error) {
				return config.LoadConfig()
			},
			loadState: func() (*state.State, error) {
				return state.Load()
			},
			runPSQL: func(dsn string) error {
				psqlCmd := exec.Command("psql", "-d", dsn)
				psqlCmd.Stdin = cmd.InOrStdin()
				psqlCmd.Stdout = cmd.OutOrStdout()
				psqlCmd.Stderr = cmd.ErrOrStderr()
				return psqlCmd.Run()
			},
			output: cmd.OutOrStdout(),
		}, args[0], format, psql)
	},
}

func init() {
	connectCmd.Flags().String("format", "url", "Output format: url, env, psql")
	connectCmd.Flags().Bool("psql", false, "Open psql for the selected branch")
	rootCmd.AddCommand(connectCmd)
}

type connectDeps struct {
	loadConfig func() (config.Config, error)
	loadState  func() (*state.State, error)
	runPSQL    func(string) error
	output     io.Writer
}

func runConnect(_ context.Context, deps connectDeps, name, format string, psql bool) error {
	cfg, err := deps.loadConfig()
	if err != nil {
		return err
	}

	branches, err := deps.loadState()
	if err != nil {
		return err
	}

	branch, ok := findBranchByDisplayName(branches, name)
	if !ok {
		return fmt.Errorf("Branch '%s' not found.", name)
	}

	dsnCfg := config.Config{
		Host:     branch.Host,
		Port:     branch.Port,
		User:     branch.User,
		Password: cfg.Password,
		Database: branch.Database,
	}
	if dsnCfg.Host == "" {
		dsnCfg.Host = cfg.Host
	}
	if dsnCfg.Port == 0 {
		dsnCfg.Port = cfg.Port
	}
	if dsnCfg.User == "" {
		dsnCfg.User = cfg.User
	}

	dsn := postgres.BuildDSN(dsnCfg)

	if psql || format == "psql" {
		return deps.runPSQL(dsn)
	}

	switch format {
	case "url":
		_, _ = fmt.Fprintln(deps.output, dsn)
		return nil
	case "env":
		_, _ = fmt.Fprintf(deps.output, "export DATABASE_URL=%s\n", dsn)
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func findBranchByDisplayName(s *state.State, name string) (state.Branch, bool) {
	for _, branch := range s.Branches {
		if branch.Name == name || DisplayName(branch.Database) == name {
			return branch, true
		}
	}

	return state.Branch{}, false
}
