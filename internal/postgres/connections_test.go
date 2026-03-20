package postgres

import (
	"context"
	"testing"
)

func TestCountActiveConnectionsCountsOtherSessions(t *testing.T) {
	ctx := context.Background()
	container, conn, cfg := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	maintenanceConn, err := ConnectToDatabase(ctx, cfg, "postgres")
	if err != nil {
		t.Fatalf("connect maintenance db: %v", err)
	}
	defer maintenanceConn.Close(ctx)

	otherConn, err := ConnectToDatabase(ctx, cfg, cfg.Database)
	if err != nil {
		t.Fatalf("connect extra session: %v", err)
	}
	defer otherConn.Close(ctx)

	count, err := CountActiveConnections(ctx, maintenanceConn, cfg.Database)
	if err != nil {
		t.Fatalf("count active connections: %v", err)
	}

	if count < 2 {
		t.Fatalf("expected at least 2 active connections, got %d", count)
	}
}
