// Package state defines dbfork branch metadata persisted to disk.
package state

import "time"

// Branch stores metadata about a cloned database branch.
type Branch struct {
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	User      string    `json:"user"`
	Database  string    `json:"database"`
	CreatedAt time.Time `json:"created_at"`
}

// State stores all known dbfork branches.
type State struct {
	Version  int      `json:"version"`
	Branches []Branch `json:"branches"`
}
