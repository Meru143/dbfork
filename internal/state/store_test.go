package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReturnsEmptyStateWhenFileNotFound(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	got, err := Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if got.Version != currentVersion {
		t.Fatalf("expected version %d, got %d", currentVersion, got.Version)
	}
	if len(got.Branches) != 0 {
		t.Fatalf("expected no branches, got %+v", got.Branches)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	want := &State{
		Branches: []Branch{
			{
				Name:      "feature-add-users",
				Source:    "myapp_development",
				Host:      "localhost",
				Port:      5432,
				User:      "tester",
				Database:  "dbfork_feature_add_users",
				CreatedAt: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := Save(want); err != nil {
		t.Fatalf("save state: %v", err)
	}

	if gotPath := statePath(); gotPath != filepath.Join(homeDir, ".dbfork", "state.json") {
		t.Fatalf("unexpected state path: %q", gotPath)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if got.Version != currentVersion {
		t.Fatalf("expected version %d, got %d", currentVersion, got.Version)
	}
	if len(got.Branches) != 1 {
		t.Fatalf("expected one branch, got %+v", got.Branches)
	}

	if got.Branches[0] != want.Branches[0] {
		t.Fatalf("unexpected branch round-trip: %+v", got.Branches[0])
	}
}

func TestAddBranchAppends(t *testing.T) {
	s := &State{Version: currentVersion, Branches: []Branch{{Name: "existing"}}}

	AddBranch(s, Branch{Name: "new-branch"})

	if len(s.Branches) != 2 {
		t.Fatalf("expected two branches, got %+v", s.Branches)
	}
	if s.Branches[1].Name != "new-branch" {
		t.Fatalf("expected appended branch, got %+v", s.Branches[1])
	}
}

func TestRemoveBranchRemovesByName(t *testing.T) {
	s := &State{
		Version: currentVersion,
		Branches: []Branch{
			{Name: "keep-me"},
			{Name: "drop-me"},
			{Name: "keep-too"},
		},
	}

	RemoveBranch(s, "drop-me")

	if len(s.Branches) != 2 {
		t.Fatalf("expected two branches after removal, got %+v", s.Branches)
	}
	if s.Branches[0].Name != "keep-me" || s.Branches[1].Name != "keep-too" {
		t.Fatalf("unexpected remaining branches: %+v", s.Branches)
	}
}

func TestFindBranchReturnsCorrectBranch(t *testing.T) {
	s := &State{
		Version: currentVersion,
		Branches: []Branch{
			{Name: "feature-add-users"},
			{Name: "feature-drop-users"},
		},
	}

	got, ok := FindBranch(s, "feature-drop-users")
	if !ok {
		t.Fatal("expected branch to be found")
	}
	if got == nil || got.Name != "feature-drop-users" {
		t.Fatalf("unexpected branch: %+v", got)
	}
}

func TestFindBranchReturnsNilForMissingName(t *testing.T) {
	got, ok := FindBranch(&State{Version: currentVersion, Branches: []Branch{}}, "missing")
	if ok {
		t.Fatal("expected missing branch lookup to return false")
	}
	if got != nil {
		t.Fatalf("expected nil branch, got %+v", got)
	}
}

func setHomeDir(t *testing.T, homeDir string) {
	t.Helper()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
}
