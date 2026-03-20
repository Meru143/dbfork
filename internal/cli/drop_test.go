package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/state"
)

func TestRunDropDropsBranchAndSavesState(t *testing.T) {
	var stdout bytes.Buffer
	saved := &state.State{}
	dropped := ""

	err := runDrop(context.Background(), dropDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Host: "localhost", Port: 5432, User: "tester"}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{Name: "feature-add-users", Database: "dbfork_feature_add_users"},
					{Name: "keep-me", Database: "dbfork_keep_me"},
				},
			}, nil
		},
		saveState: func(s *state.State) error {
			*saved = *s
			return nil
		},
		connectMaintenance: func(context.Context, config.Config) (*pgx.Conn, error) {
			return nil, nil
		},
		dropBranch: func(_ context.Context, _ *pgx.Conn, branchName string) error {
			dropped = branchName
			return nil
		},
		input:  strings.NewReader(""),
		output: &stdout,
	}, "feature-add-users", true)
	if err != nil {
		t.Fatalf("run drop: %v", err)
	}

	if dropped != "dbfork_feature_add_users" {
		t.Fatalf("expected branch db to be dropped, got %q", dropped)
	}

	if len(saved.Branches) != 1 || saved.Branches[0].Name != "keep-me" {
		t.Fatalf("unexpected saved state: %+v", saved.Branches)
	}

	if !strings.Contains(stdout.String(), "✓ Dropped branch 'feature-add-users'") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunDropPromptsAndStopsWhenNotConfirmed(t *testing.T) {
	var stdout bytes.Buffer
	dropCalled := false
	saveCalled := false

	err := runDrop(context.Background(), dropDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{Name: "feature-add-users", Database: "dbfork_feature_add_users"},
				},
			}, nil
		},
		saveState: func(*state.State) error {
			saveCalled = true
			return nil
		},
		dropBranch: func(context.Context, *pgx.Conn, string) error {
			dropCalled = true
			return nil
		},
		input:  strings.NewReader("n\n"),
		output: &stdout,
	}, "feature-add-users", false)
	if err != nil {
		t.Fatalf("run drop: %v", err)
	}

	if !strings.Contains(stdout.String(), "Drop branch 'feature-add-users'? This cannot be undone. [y/N]: ") {
		t.Fatalf("expected confirmation prompt, got %q", stdout.String())
	}

	if dropCalled {
		t.Fatal("expected dropBranch not to be called")
	}

	if saveCalled {
		t.Fatal("expected state not to be saved")
	}
}

func TestRunDropReturnsErrorWhenBranchIsMissing(t *testing.T) {
	err := runDrop(context.Background(), dropDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		output: &bytes.Buffer{},
	}, "missing-branch", true)
	if err == nil {
		t.Fatal("expected missing branch error")
	}

	if !strings.Contains(err.Error(), "Branch 'missing-branch' not found.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDropCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"drop"})
	if err != nil {
		t.Fatalf("find drop command: %v", err)
	}

	if cmd == nil || cmd.Use != "drop <name>" {
		t.Fatalf("expected drop command to be registered, got %#v", cmd)
	}
}
