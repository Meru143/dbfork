package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

func TestRunDiffPrintsJSON(t *testing.T) {
	var stdout bytes.Buffer
	connectCalls := []string{}

	err := runDiff(context.Background(), diffDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Password: "secret"}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{
						Name:     "feature-add-users",
						Host:     "localhost",
						Port:     5432,
						User:     "tester",
						Source:   "myapp_development",
						Database: "dbfork_feature_add_users",
					},
				},
			}, nil
		},
		connectToDatabase: func(_ context.Context, cfg config.Config, dbName string) (*pgx.Conn, error) {
			connectCalls = append(connectCalls, dbName)
			if cfg.Password != "secret" {
				t.Fatalf("expected config password to be propagated, got %q", cfg.Password)
			}
			return nil, nil
		},
		computeSchemaDiff: func(context.Context, *pgx.Conn, *pgx.Conn) (*postgres.SchemaDiff, error) {
			return &postgres.SchemaDiff{AddedTables: []string{"widgets"}}, nil
		},
		render: func(*postgres.SchemaDiff) {
			t.Fatal("did not expect text renderer for json output")
		},
		output: &stdout,
	}, "feature-add-users", "json")
	if err != nil {
		t.Fatalf("run diff: %v", err)
	}

	if len(connectCalls) != 2 || connectCalls[0] != "myapp_development" || connectCalls[1] != "dbfork_feature_add_users" {
		t.Fatalf("unexpected database connections: %+v", connectCalls)
	}

	if !strings.Contains(stdout.String(), "\"AddedTables\": [") || !strings.Contains(stdout.String(), "\"widgets\"") {
		t.Fatalf("unexpected json output: %q", stdout.String())
	}
}

func TestRunDiffRendersTextOutput(t *testing.T) {
	rendered := false

	err := runDiff(context.Background(), diffDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{
						Name:     "feature-add-users",
						Host:     "localhost",
						Port:     5432,
						User:     "tester",
						Source:   "myapp_development",
						Database: "dbfork_feature_add_users",
					},
				},
			}, nil
		},
		connectToDatabase: func(context.Context, config.Config, string) (*pgx.Conn, error) {
			return nil, nil
		},
		computeSchemaDiff: func(context.Context, *pgx.Conn, *pgx.Conn) (*postgres.SchemaDiff, error) {
			return &postgres.SchemaDiff{AddedTables: []string{"widgets"}}, nil
		},
		render: func(diff *postgres.SchemaDiff) {
			rendered = true
			if len(diff.AddedTables) != 1 || diff.AddedTables[0] != "widgets" {
				t.Fatalf("unexpected diff passed to renderer: %+v", diff)
			}
		},
		output: &bytes.Buffer{},
	}, "feature-add-users", "text")
	if err != nil {
		t.Fatalf("run diff: %v", err)
	}

	if !rendered {
		t.Fatal("expected text renderer to be called")
	}
}

func TestRunDiffReturnsErrorWhenBranchIsMissing(t *testing.T) {
	err := runDiff(context.Background(), diffDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		output: &bytes.Buffer{},
	}, "missing-branch", "text")
	if err == nil {
		t.Fatal("expected missing branch error")
	}

	if !strings.Contains(err.Error(), "Branch 'missing-branch' not found.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDiffReturnsErrorForUnsupportedFormat(t *testing.T) {
	err := runDiff(context.Background(), diffDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{Name: "feature-add-users", Host: "localhost", Port: 5432, User: "tester", Source: "myapp_development", Database: "dbfork_feature_add_users"},
				},
			}, nil
		},
		output: &bytes.Buffer{},
	}, "feature-add-users", "yaml")
	if err == nil {
		t.Fatal("expected unsupported format error")
	}

	if !strings.Contains(err.Error(), "unsupported format \"yaml\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiffCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"diff"})
	if err != nil {
		t.Fatalf("find diff command: %v", err)
	}

	if cmd == nil || cmd.Use != "diff <name>" {
		t.Fatalf("expected diff command to be registered, got %#v", cmd)
	}
}
