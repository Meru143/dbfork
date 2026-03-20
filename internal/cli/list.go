// Package cli provides the dbfork list command.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/output"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List known database branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.Context(), listDeps{
			loadConfig: func() (config.Config, error) {
				return config.LoadConfig()
			},
			loadState: func() (*state.State, error) {
				return state.Load()
			},
			connectMaintenance: func(ctx context.Context, cfg config.Config) (*pgx.Conn, error) {
				return postgres.ConnectToDatabase(ctx, cfg, "postgres")
			},
			getDatabaseSizeMB: postgres.GetDatabaseSizeMB,
			databaseExists:    postgres.DatabaseExists,
			readActiveBranch:  readActiveBranchFile,
			render:            output.RenderBranchList,
			output:            cmd.OutOrStdout(),
		})
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

type listDeps struct {
	loadConfig         func() (config.Config, error)
	loadState          func() (*state.State, error)
	connectMaintenance func(context.Context, config.Config) (*pgx.Conn, error)
	getDatabaseSizeMB  func(context.Context, *pgx.Conn, string) (float64, error)
	databaseExists     func(context.Context, *pgx.Conn, string) (bool, error)
	readActiveBranch   func() (string, error)
	render             func([]output.BranchRow)
	output             io.Writer
}

func runList(ctx context.Context, deps listDeps) error {
	cfg, err := deps.loadConfig()
	if err != nil {
		return err
	}

	branches, err := deps.loadState()
	if err != nil {
		return err
	}

	if len(branches.Branches) == 0 {
		_, _ = fmt.Fprintln(deps.output, "No branches. Run 'dbfork create <name>' to get started.")
		return nil
	}

	conn, err := deps.connectMaintenance(ctx, cfg)
	if err != nil {
		return fmt.Errorf("Cannot connect to postgres://%s@%s:%d. Check your config with 'dbfork init'.", cfg.User, cfg.Host, cfg.Port)
	}
	if conn != nil {
		defer conn.Close(ctx)
	}

	activeBranch, err := deps.readActiveBranch()
	if err != nil {
		return err
	}

	rows := make([]output.BranchRow, 0, len(branches.Branches))
	for _, branch := range branches.Branches {
		exists, err := deps.databaseExists(ctx, conn, branch.Database)
		if err != nil {
			return err
		}

		status := "ready"
		size := "0.0"
		if exists {
			sizeMB, err := deps.getDatabaseSizeMB(ctx, conn, branch.Database)
			if err != nil {
				return err
			}
			size = fmt.Sprintf("%.1f", sizeMB)
		} else {
			status = "orphaned"
		}

		activeMarker := ""
		if activeBranch != "" && (activeBranch == branch.Name || activeBranch == DisplayName(branch.Database)) {
			activeMarker = "*"
		}

		rows = append(rows, output.BranchRow{
			Active:  activeMarker,
			Name:    branch.Name,
			Source:  branch.Source,
			Created: branch.CreatedAt.Format(time.RFC3339),
			SizeMB:  size,
			Status:  status,
		})
	}

	deps.render(rows)
	return nil
}

func readActiveBranchFile() (string, error) {
	data, err := os.ReadFile(".dbfork")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
