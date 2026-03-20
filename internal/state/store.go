// Package state loads and saves dbfork state.json metadata.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const currentVersion = 1

func statePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".dbfork", "state.json")
	}

	return filepath.Join(homeDir, ".dbfork", "state.json")
}

// Load reads the dbfork state file from disk.
func Load() (*State, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Version: currentVersion, Branches: []Branch{}}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Version == 0 {
		state.Version = currentVersion
	}
	if state.Branches == nil {
		state.Branches = []Branch{}
	}

	return &state, nil
}

// Save writes the dbfork state file to disk.
func Save(s *State) error {
	stateDir := filepath.Dir(statePath())
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}

	s.Version = currentVersion
	if s.Branches == nil {
		s.Branches = []Branch{}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath(), data, 0o600)
}

// AddBranch appends a branch to the state.
func AddBranch(s *State, b Branch) {
	s.Version = currentVersion
	s.Branches = append(s.Branches, b)
}

// RemoveBranch removes a branch by name.
func RemoveBranch(s *State, name string) {
	filtered := make([]Branch, 0, len(s.Branches))
	for _, branch := range s.Branches {
		if branch.Name == name {
			continue
		}
		filtered = append(filtered, branch)
	}
	s.Branches = filtered
}

// FindBranch returns the branch with the given name.
func FindBranch(s *State, name string) (*Branch, bool) {
	for i := range s.Branches {
		if s.Branches[i].Name == name {
			return &s.Branches[i], true
		}
	}

	return nil, false
}
