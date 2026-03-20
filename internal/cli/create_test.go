package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
	"github.com/owner/dbfork/internal/state"
)

func TestRunCreateCreatesBranchAndSavesState(t *testing.T) {
	var output bytes.Buffer
	saved := &state.State{}
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	createCalled := false

	deps := createDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "tester",
				Password: "secret",
				Database: "myapp_development",
			}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		saveState: func(s *state.State) error {
			*saved = *s
			return nil
		},
		connectMaintenance: func(context.Context, config.Config) (*pgx.Conn, error) {
			return nil, nil
		},
		hasCreateDBPrivilege: func(context.Context, *pgx.Conn) (bool, error) {
			return true, nil
		},
		databaseExists: func(context.Context, *pgx.Conn, string) (bool, error) {
			return false, nil
		},
		terminateIdleConnections: func(context.Context, *pgx.Conn, string) (int, error) {
			return 2, nil
		},
		countActiveConnections: func(context.Context, *pgx.Conn, string) (int, error) {
			return 1, nil
		},
		createBranch: func(_ context.Context, _ *pgx.Conn, sourceName, branchName string) error {
			createCalled = true
			if sourceName != "myapp_development" {
				t.Fatalf("expected source myapp_development, got %q", sourceName)
			}
			if branchName != "dbfork_feature_add_users" {
				t.Fatalf("expected sanitized branch db name, got %q", branchName)
			}
			return nil
		},
		now:    func() time.Time { return now },
		output: &output,
	}

	if err := runCreate(context.Background(), deps, "feature-add-users", ""); err != nil {
		t.Fatalf("run create: %v", err)
	}

	if !createCalled {
		t.Fatal("expected createBranch to be called")
	}

	if len(saved.Branches) != 1 {
		t.Fatalf("expected one saved branch, got %+v", saved.Branches)
	}

	branch := saved.Branches[0]
	if branch.Name != "feature-add-users" || branch.Source != "myapp_development" || branch.Database != "dbfork_feature_add_users" || !branch.CreatedAt.Equal(now) {
		t.Fatalf("unexpected saved branch: %+v", branch)
	}

	out := output.String()
	for _, want := range []string{
		"Terminated 2 idle connection(s)",
		"Warning: 1 active connection(s) remain on source database.",
		"Created branch 'feature-add-users'",
		"Connect: dbfork connect feature-add-users",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestRunCreateReturnsE001WhenBranchAlreadyExistsInState(t *testing.T) {
	deps := createDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Database: "myapp_development"}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{Name: "feature-add-users", Database: "dbfork_feature_add_users"},
				},
			}, nil
		},
		output: &bytes.Buffer{},
	}

	err := runCreate(context.Background(), deps, "feature-add-users", "")
	if err == nil {
		t.Fatal("expected duplicate branch error")
	}

	if !strings.Contains(err.Error(), "Branch 'feature-add-users' already exists. Use 'dbfork drop feature-add-users' to remove it.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("find create command: %v", err)
	}

	if cmd == nil || cmd.Use != "create <name>" {
		t.Fatalf("expected create command to be registered, got %#v", cmd)
	}
}

func TestRunCreateReturnsE002WhenMaintenanceConnectionFails(t *testing.T) {
	err := runCreate(context.Background(), createDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Host: "localhost", Port: 5432, User: "tester", Database: "myapp_development"}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		connectMaintenance: func(context.Context, config.Config) (*pgx.Conn, error) {
			return nil, errors.New("dial failed")
		},
		output: &bytes.Buffer{},
	}, "feature-add-users", "")
	if err == nil {
		t.Fatal("expected connection error")
	}

	if !strings.Contains(err.Error(), "Cannot connect to postgres://tester@localhost:5432. Check your config with 'dbfork init'.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreateReturnsE004WhenUserLacksCreateDBPrivilege(t *testing.T) {
	err := runCreate(context.Background(), createDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Host: "localhost", Port: 5432, User: "tester", Database: "myapp_development"}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		connectMaintenance: func(context.Context, config.Config) (*pgx.Conn, error) {
			return nil, nil
		},
		hasCreateDBPrivilege: func(context.Context, *pgx.Conn) (bool, error) {
			return false, nil
		},
		output: &bytes.Buffer{},
	}, "feature-add-users", "")
	if err == nil {
		t.Fatal("expected privilege error")
	}

	if !strings.Contains(err.Error(), "User 'tester' does not have CREATEDB privilege. Grant with: ALTER USER tester CREATEDB;") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartCreateSpinnerWritesOutput(t *testing.T) {
	var output bytes.Buffer

	stop := startCreateSpinner(&output)
	time.Sleep(150 * time.Millisecond)
	stop()

	if output.Len() == 0 {
		t.Fatal("expected spinner output to be written")
	}
}

var _ = postgres.ErrBranchExists
