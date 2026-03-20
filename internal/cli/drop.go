// Package cli provides the dbfork drop command.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/Meru143/dbfork/internal/config"
	"github.com/Meru143/dbfork/internal/postgres"
	"github.com/Meru143/dbfork/internal/state"
)

var dropCmd = &cobra.Command{
	Use:   "drop <name>",
	Short: "Drop a branched database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}

		return runDrop(cmd.Context(), dropDeps{
			loadConfig: func() (config.Config, error) {
				return config.LoadConfig()
			},
			loadState: func() (*state.State, error) {
				return state.Load()
			},
			saveState: func(s *state.State) error {
				return state.Save(s)
			},
			connectMaintenance: func(ctx context.Context, cfg config.Config) (*pgx.Conn, error) {
				return postgres.ConnectToDatabase(ctx, cfg, "postgres")
			},
			dropBranch: postgres.DropBranch,
			input:      cmd.InOrStdin(),
			output:     cmd.OutOrStdout(),
		}, args[0], force)
	},
}

func init() {
	dropCmd.Flags().Bool("force", false, "Drop without confirmation")
	rootCmd.AddCommand(dropCmd)
}

type dropDeps struct {
	loadConfig         func() (config.Config, error)
	loadState          func() (*state.State, error)
	saveState          func(*state.State) error
	connectMaintenance func(context.Context, config.Config) (*pgx.Conn, error)
	dropBranch         func(context.Context, *pgx.Conn, string) error
	input              io.Reader
	output             io.Writer
}

func runDrop(ctx context.Context, deps dropDeps, name string, force bool) error {
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

	if !force {
		_, _ = fmt.Fprintf(deps.output, "Drop branch '%s'? This cannot be undone. [y/N]: ", name)
		reader := bufio.NewReader(deps.input)
		answer, err := reader.ReadString('\n')
		if err != nil && len(answer) == 0 {
			return err
		}
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			return nil
		}
	}

	conn, err := deps.connectMaintenance(ctx, cfg)
	if err != nil {
		return fmt.Errorf("Cannot connect to postgres://%s@%s:%d. Check your config with 'dbfork init'.", cfg.User, cfg.Host, cfg.Port)
	}
	if conn != nil {
		defer conn.Close(ctx)
	}

	if err := deps.dropBranch(ctx, conn, branch.Database); err != nil {
		return err
	}

	state.RemoveBranch(branches, branch.Name)
	if err := deps.saveState(branches); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(deps.output, "✓ Dropped branch '%s'\n", branch.Name)
	return nil
}
