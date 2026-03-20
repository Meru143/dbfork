package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/state"
)

func TestRunUseWritesActiveBranchFileAndPrintsDSN(t *testing.T) {
	var stdout bytes.Buffer
	writtenPath := ""
	writtenContents := ""

	err := runUse(context.Background(), useDeps{
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
		getwd: func() (string, error) {
			return `C:\work\app`, nil
		},
		writeFile: func(path string, data []byte, perm os.FileMode) error {
			writtenPath = path
			writtenContents = string(data)
			if perm != 0o644 {
				t.Fatalf("unexpected file mode: %v", perm)
			}
			return nil
		},
		output: &stdout,
	}, "feature-add-users")
	if err != nil {
		t.Fatalf("run use: %v", err)
	}

	if writtenPath != `C:\work\app\.dbfork` {
		t.Fatalf("unexpected file path: %q", writtenPath)
	}

	if writtenContents != "feature-add-users\n" {
		t.Fatalf("unexpected file contents: %q", writtenContents)
	}

	out := stdout.String()
	if !strings.Contains(out, "✓ Switched to branch 'feature-add-users'") {
		t.Fatalf("expected success output, got %q", out)
	}
	if !strings.Contains(out, "DATABASE_URL=postgres://tester:secret@localhost:5432/dbfork_feature_add_users") {
		t.Fatalf("expected DATABASE_URL output, got %q", out)
	}
}

func TestRunUseReturnsErrorWhenBranchIsMissing(t *testing.T) {
	err := runUse(context.Background(), useDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		loadState: func() (*state.State, error) {
			return &state.State{Version: 1, Branches: []state.Branch{}}, nil
		},
		output: &bytes.Buffer{},
	}, "missing-branch")
	if err == nil {
		t.Fatal("expected missing branch error")
	}

	if !strings.Contains(err.Error(), "Branch 'missing-branch' not found.") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"use"})
	if err != nil {
		t.Fatalf("find use command: %v", err)
	}

	if cmd == nil || cmd.Use != "use <name>" {
		t.Fatalf("expected use command to be registered, got %#v", cmd)
	}
}
