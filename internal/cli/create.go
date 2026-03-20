// Package cli provides the dbfork create command.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a branched database from a source database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source, err := cmd.Flags().GetString("source")
		if err != nil {
			return err
		}

		return runCreate(cmd.Context(), createDeps{
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
			hasCreateDBPrivilege:     postgres.HasCreateDBPrivilege,
			databaseExists:           postgres.DatabaseExists,
			terminateIdleConnections: postgres.TerminateIdleConnections,
			countActiveConnections:   postgres.CountActiveConnections,
			createBranch:             postgres.CreateBranch,
			now:                      time.Now,
			output:                   cmd.OutOrStdout(),
			spinnerEnabled:           true,
		}, args[0], source)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}

type createDeps struct {
	loadConfig               func() (config.Config, error)
	loadState                func() (*state.State, error)
	saveState                func(*state.State) error
	connectMaintenance       func(context.Context, config.Config) (*pgx.Conn, error)
	hasCreateDBPrivilege     func(context.Context, *pgx.Conn) (bool, error)
	databaseExists           func(context.Context, *pgx.Conn, string) (bool, error)
	terminateIdleConnections func(context.Context, *pgx.Conn, string) (int, error)
	countActiveConnections   func(context.Context, *pgx.Conn, string) (int, error)
	createBranch             func(context.Context, *pgx.Conn, string, string) error
	now                      func() time.Time
	output                   io.Writer
	spinnerEnabled           bool
}

func runCreate(ctx context.Context, deps createDeps, branchName, sourceOverride string) error {
	cfg, err := deps.loadConfig()
	if err != nil {
		return err
	}

	branches, err := deps.loadState()
	if err != nil {
		return err
	}

	sanitizedName, err := SanitizeBranchName(branchName)
	if err != nil {
		return err
	}

	for _, branch := range branches.Branches {
		if branch.Database == sanitizedName || branch.Name == branchName {
			return fmt.Errorf("Branch '%s' already exists. Use 'dbfork drop %s' to remove it.", branchName, branchName)
		}
	}

	source := cfg.Database
	if sourceOverride != "" {
		source = sourceOverride
	}

	conn, err := deps.connectMaintenance(ctx, cfg)
	if err != nil {
		return fmt.Errorf("Cannot connect to postgres://%s@%s:%d. Check your config with 'dbfork init'.", cfg.User, cfg.Host, cfg.Port)
	}
	if conn != nil {
		defer conn.Close(ctx)
	}

	allowed, err := deps.hasCreateDBPrivilege(ctx, conn)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("User '%s' does not have CREATEDB privilege. Grant with: ALTER USER %s CREATEDB;", cfg.User, cfg.User)
	}

	exists, err := deps.databaseExists(ctx, conn, sanitizedName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("Branch '%s' already exists. Use 'dbfork drop %s' to remove it.", branchName, branchName)
	}

	terminated, err := deps.terminateIdleConnections(ctx, conn, source)
	if err != nil {
		return err
	}
	if terminated > 0 {
		_, _ = fmt.Fprintf(deps.output, "Terminated %d idle connection(s)\n", terminated)
	}

	active, err := deps.countActiveConnections(ctx, conn, source)
	if err != nil {
		return err
	}
	if active > 0 {
		_, _ = fmt.Fprintf(deps.output, "Warning: %d active connection(s) remain on source database.\n", active)
	}

	stopSpinner := func() {}
	if deps.spinnerEnabled {
		stopSpinner = startCreateSpinner(deps.output)
	}

	err = deps.createBranch(ctx, conn, source, sanitizedName)
	stopSpinner()
	if err != nil {
		if errors.Is(err, postgres.ErrBranchExists) {
			return fmt.Errorf("Branch '%s' already exists. Use 'dbfork drop %s' to remove it.", branchName, branchName)
		}
		return err
	}

	state.AddBranch(branches, state.Branch{
		Name:      branchName,
		Source:    source,
		Host:      cfg.Host,
		Port:      cfg.Port,
		User:      cfg.User,
		Database:  sanitizedName,
		CreatedAt: deps.now(),
	})
	if err := deps.saveState(branches); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(deps.output, "✓ Created branch '%s'\n", branchName)
	_, _ = fmt.Fprintf(deps.output, "  Connect: dbfork connect %s\n", branchName)
	return nil
}

func startCreateSpinner(output io.Writer) func() {
	done := make(chan struct{})
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	frames := []string{"|", "/", "-", `\`}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		index := 0
		for {
			select {
			case <-done:
				_, _ = fmt.Fprint(output, "\r")
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(output, "\r%s Creating branch...", style.Render(frames[index%len(frames)]))
				index++
			}
		}
	}()

	return func() {
		close(done)
	}
}
