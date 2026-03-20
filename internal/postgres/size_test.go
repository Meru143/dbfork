package postgres

import (
	"context"
	"testing"
)

func TestGetDatabaseSizeMBReturnsPositiveValueForExistingDatabase(t *testing.T) {
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

	sizeMB, err := GetDatabaseSizeMB(ctx, maintenanceConn, cfg.Database)
	if err != nil {
		t.Fatalf("get database size: %v", err)
	}
	if sizeMB <= 0 {
		t.Fatalf("expected positive size, got %f", sizeMB)
	}
}

func TestGetDatabaseSizeMBReturnsZeroForMissingDatabase(t *testing.T) {
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

	sizeMB, err := GetDatabaseSizeMB(ctx, maintenanceConn, "missing_db")
	if err != nil {
		t.Fatalf("get missing database size: %v", err)
	}
	if sizeMB != 0 {
		t.Fatalf("expected zero size for missing database, got %f", sizeMB)
	}
}
