package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Meru143/dbfork/internal/config"
	"github.com/Meru143/dbfork/internal/output"
	"github.com/Meru143/dbfork/internal/state"
)

func TestRunListPrintsEmptyStateMessage(t *testing.T) {
	var stdout bytes.Buffer
	err := runList(context.Background(), listDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		output: &stdout,
	})
	if err != nil {
		t.Fatalf("run list: %v", err)
	}

	if !strings.Contains(stdout.String(), "No branches. Run 'dbfork create <name>' to get started.") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestListCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find list command: %v", err)
	}

	if cmd == nil || cmd.Use != "list" {
		t.Fatalf("expected list command to be registered, got %#v", cmd)
	}
}

func TestRunListRendersBranchRows(t *testing.T) {
	var stdout bytes.Buffer
	rendered := []output.BranchRow{}

	err := runList(context.Background(), listDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{
						Name:      "feature-add-users",
						Source:    "myapp_development",
						Database:  "dbfork_feature_add_users",
						CreatedAt: mustParseTime(t, "2026-03-20T12:00:00Z"),
					},
				},
			}, nil
		},
		connectMaintenance: func(context.Context, config.Config) (*pgx.Conn, error) {
			return nil, nil
		},
		getDatabaseSizeMB: func(context.Context, *pgx.Conn, string) (float64, error) {
			return 12.5, nil
		},
		databaseExists: func(context.Context, *pgx.Conn, string) (bool, error) {
			return true, nil
		},
		readActiveBranch: func() (string, error) {
			return "feature-add-users", nil
		},
		render: func(rows []output.BranchRow) {
			rendered = rows
		},
		output: &stdout,
	})
	if err != nil {
		t.Fatalf("run list: %v", err)
	}

	if len(rendered) != 1 {
		t.Fatalf("expected one rendered row, got %+v", rendered)
	}

	row := rendered[0]
	if row.Active != "*" || row.Name != "feature-add-users" || row.Source != "myapp_development" || row.SizeMB != "12.5" || row.Status != "ready" {
		t.Fatalf("unexpected rendered row: %+v", row)
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func TestReadActiveBranchFileReturnsTrimmedValue(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, ".dbfork"), []byte("feature-add-users\n"), 0o600); err != nil {
		t.Fatalf("write active branch file: %v", err)
	}

	got, err := readActiveBranchFile()
	if err != nil {
		t.Fatalf("read active branch file: %v", err)
	}

	if got != "feature-add-users" {
		t.Fatalf("expected trimmed branch name, got %q", got)
	}
}
