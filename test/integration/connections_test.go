//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/owner/dbfork/internal/postgres"
)

func TestTerminateIdleConnections(t *testing.T) {
	ctx := context.Background()
	maintenance := maintenanceConn(t)

	closeSourceConn(t)
	defer reconnectSourceConn(t)

	connCh := make(chan *pgx.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := postgres.ConnectToDatabase(ctx, testConfig, testConfig.Database)
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()

	var idleConn *pgx.Conn
	select {
	case err := <-errCh:
		t.Fatalf("open idle connection: %v", err)
	case idleConn = <-connCh:
	}
	defer idleConn.Close(ctx)

	terminated, err := postgres.TerminateIdleConnections(ctx, maintenance, testConfig.Database)
	if err != nil {
		t.Fatalf("terminate idle connections: %v", err)
	}

	if terminated != 1 {
		t.Fatalf("expected 1 terminated connection, got %d", terminated)
	}
}

func TestGetDatabaseSizeMB(t *testing.T) {
	ctx := context.Background()
	maintenance := maintenanceConn(t)

	sizeMB, err := postgres.GetDatabaseSizeMB(ctx, maintenance, testConfig.Database)
	if err != nil {
		t.Fatalf("get database size: %v", err)
	}

	if sizeMB <= 0 {
		t.Fatalf("expected positive database size, got %f", sizeMB)
	}
}
