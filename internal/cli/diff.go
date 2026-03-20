// Package cli provides the dbfork diff command.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/output"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

var diffCmd = &cobra.Command{
	Use:   "diff <name>",
	Short: "Show schema differences between a branch and its source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}

		return runDiff(cmd.Context(), diffDeps{
			loadConfig: func() (config.Config, error) {
				return config.LoadConfig()
			},
			loadState: func() (*state.State, error) {
				return state.Load()
			},
			connectToDatabase: postgres.ConnectToDatabase,
			computeSchemaDiff: postgres.ComputeSchemaDiff,
			render:            output.RenderDiff,
			output:            cmd.OutOrStdout(),
		}, args[0], format)
	},
}

func init() {
	diffCmd.Flags().String("format", "text", "Output format: text or json")
	rootCmd.AddCommand(diffCmd)
}

type diffDeps struct {
	loadConfig        func() (config.Config, error)
	loadState         func() (*state.State, error)
	connectToDatabase func(context.Context, config.Config, string) (*pgx.Conn, error)
	computeSchemaDiff func(context.Context, *pgx.Conn, *pgx.Conn) (*postgres.SchemaDiff, error)
	render            func(*postgres.SchemaDiff)
	output            io.Writer
}

func runDiff(ctx context.Context, deps diffDeps, name, format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported format %q", format)
	}

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

	sourceCfg := config.Config{
		Host:     branch.Host,
		Port:     branch.Port,
		User:     branch.User,
		Password: cfg.Password,
		Database: branch.Source,
	}
	branchCfg := sourceCfg
	branchCfg.Database = branch.Database

	if sourceCfg.Host == "" {
		sourceCfg.Host = cfg.Host
		branchCfg.Host = cfg.Host
	}
	if sourceCfg.Port == 0 {
		sourceCfg.Port = cfg.Port
		branchCfg.Port = cfg.Port
	}
	if sourceCfg.User == "" {
		sourceCfg.User = cfg.User
		branchCfg.User = cfg.User
	}

	sourceConn, err := deps.connectToDatabase(ctx, sourceCfg, branch.Source)
	if err != nil {
		return fmt.Errorf("Cannot connect to postgres://%s@%s:%d. Check your config with 'dbfork init'.", sourceCfg.User, sourceCfg.Host, sourceCfg.Port)
	}
	if sourceConn != nil {
		defer sourceConn.Close(ctx)
	}

	branchConn, err := deps.connectToDatabase(ctx, branchCfg, branch.Database)
	if err != nil {
		return fmt.Errorf("Cannot connect to postgres://%s@%s:%d. Check your config with 'dbfork init'.", branchCfg.User, branchCfg.Host, branchCfg.Port)
	}
	if branchConn != nil {
		defer branchConn.Close(ctx)
	}

	diff, err := deps.computeSchemaDiff(ctx, sourceConn, branchConn)
	if err != nil {
		return err
	}

	if format == "json" {
		data, err := json.MarshalIndent(diff, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(deps.output, string(data))
		return nil
	}

	deps.render(diff)
	return nil
}
