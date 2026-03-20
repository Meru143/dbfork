package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Meru143/dbfork/internal/config"
	"github.com/Meru143/dbfork/internal/state"
)

func TestRunConnectPrintsURLFormat(t *testing.T) {
	var stdout bytes.Buffer

	err := runConnect(context.Background(), connectDeps{
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
						Database: "dbfork_feature_add_users",
					},
				},
			}, nil
		},
		output: &stdout,
	}, "feature-add-users", "url", false)
	if err != nil {
		t.Fatalf("run connect: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != "postgres://tester:secret@localhost:5432/dbfork_feature_add_users" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunConnectPrintsEnvFormat(t *testing.T) {
	var stdout bytes.Buffer

	err := runConnect(context.Background(), connectDeps{
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
						Database: "dbfork_feature_add_users",
					},
				},
			}, nil
		},
		output: &stdout,
	}, "feature-add-users", "env", false)
	if err != nil {
		t.Fatalf("run connect: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != "export DATABASE_URL=postgres://tester:secret@localhost:5432/dbfork_feature_add_users" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunConnectLaunchesPSQL(t *testing.T) {
	var stdout bytes.Buffer
	called := false

	err := runConnect(context.Background(), connectDeps{
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
						Database: "dbfork_feature_add_users",
					},
				},
			}, nil
		},
		runPSQL: func(dsn string) error {
			called = true
			if dsn != "postgres://tester:secret@localhost:5432/dbfork_feature_add_users" {
				t.Fatalf("unexpected dsn: %q", dsn)
			}
			return nil
		},
		output: &stdout,
	}, "feature-add-users", "url", true)
	if err != nil {
		t.Fatalf("run connect: %v", err)
	}

	if !called {
		t.Fatal("expected psql to be launched")
	}

	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout when launching psql, got %q", stdout.String())
	}
}

func TestRunConnectReturnsErrorWhenBranchIsMissing(t *testing.T) {
	err := runConnect(context.Background(), connectDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		output: &bytes.Buffer{},
	}, "missing-branch", "url", false)
	if err == nil {
		t.Fatal("expected missing branch error")
	}

	if !strings.Contains(err.Error(), "Branch 'missing-branch' not found.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConnectReturnsErrorForUnsupportedFormat(t *testing.T) {
	err := runConnect(context.Background(), connectDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{
				Version: 1,
				Branches: []state.Branch{
					{Name: "feature-add-users", Host: "localhost", Port: 5432, User: "tester", Database: "dbfork_feature_add_users"},
				},
			}, nil
		},
		output: &bytes.Buffer{},
	}, "feature-add-users", "json", false)
	if err == nil {
		t.Fatal("expected unsupported format error")
	}

	if !strings.Contains(err.Error(), "unsupported format \"json\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"connect"})
	if err != nil {
		t.Fatalf("find connect command: %v", err)
	}

	if cmd == nil || cmd.Use != "connect <name>" {
		t.Fatalf("expected connect command to be registered, got %#v", cmd)
	}
}
