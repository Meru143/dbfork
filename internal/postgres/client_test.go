package postgres

import (
	"testing"

	"github.com/owner/dbfork/internal/config"
)

func TestBuildDSNWithPassword(t *testing.T) {
	got := BuildDSN(config.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "tester",
		Password: "secret",
		Database: "myapp_development",
	})

	if got != "postgres://tester:secret@localhost:5432/myapp_development" {
		t.Fatalf("unexpected dsn: %q", got)
	}
}

func TestBuildDSNWithoutPassword(t *testing.T) {
	got := BuildDSN(config.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "tester",
		Database: "myapp_development",
	})

	if got != "postgres://tester@localhost:5432/myapp_development" {
		t.Fatalf("unexpected dsn: %q", got)
	}
}
