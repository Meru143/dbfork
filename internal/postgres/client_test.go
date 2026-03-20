package postgres

import (
	"context"
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

func TestPingReturnsNilOnOpenConnection(t *testing.T) {
	ctx := context.Background()
	container, conn, _ := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	if err := Ping(ctx, conn); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestHasCreateDBPrivilegeReturnsTrueForContainerUser(t *testing.T) {
	ctx := context.Background()
	container, conn, _ := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	allowed, err := HasCreateDBPrivilege(ctx, conn)
	if err != nil {
		t.Fatalf("has create db privilege: %v", err)
	}
	if !allowed {
		t.Fatal("expected container user to have CREATEDB privilege")
	}
}
