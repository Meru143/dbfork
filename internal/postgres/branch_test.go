package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCreateBranchCreatesDatabaseAndDatabaseExists(t *testing.T) {
	ctx := context.Background()
	container, sourceConn, cfg := startPostgresContainer(t, ctx)
	defer func() {
		sourceConn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	maintenanceConn, err := ConnectToDatabase(ctx, cfg, "postgres")
	if err != nil {
		t.Fatalf("connect maintenance db: %v", err)
	}
	defer maintenanceConn.Close(ctx)

	branchName := fmt.Sprintf("dbfork_branch_%d", time.Now().UnixNano())
	sourceConn.Close(ctx)
	if err := CreateBranch(ctx, maintenanceConn, cfg.Database, branchName); err != nil {
		t.Fatalf("create branch: %v", err)
	}
	defer func() {
		_ = DropBranch(ctx, maintenanceConn, branchName)
	}()

	exists, err := DatabaseExists(ctx, maintenanceConn, branchName)
	if err != nil {
		t.Fatalf("database exists: %v", err)
	}
	if !exists {
		t.Fatalf("expected branch %q to exist", branchName)
	}
}

func TestCreateBranchReturnsErrBranchExists(t *testing.T) {
	ctx := context.Background()
	container, sourceConn, cfg := startPostgresContainer(t, ctx)
	defer func() {
		sourceConn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	maintenanceConn, err := ConnectToDatabase(ctx, cfg, "postgres")
	if err != nil {
		t.Fatalf("connect maintenance db: %v", err)
	}
	defer maintenanceConn.Close(ctx)

	branchName := fmt.Sprintf("dbfork_exists_%d", time.Now().UnixNano())
	sourceConn.Close(ctx)
	if err := CreateBranch(ctx, maintenanceConn, cfg.Database, branchName); err != nil {
		t.Fatalf("first create branch: %v", err)
	}
	defer func() {
		_ = DropBranch(ctx, maintenanceConn, branchName)
	}()

	sourceConn, err = ConnectToDatabase(ctx, cfg, cfg.Database)
	if err != nil {
		t.Fatalf("reconnect source db: %v", err)
	}
	sourceConn.Close(ctx)

	err = CreateBranch(ctx, maintenanceConn, cfg.Database, branchName)
	if !errors.Is(err, ErrBranchExists) {
		t.Fatalf("expected ErrBranchExists, got %v", err)
	}
}

func TestCreateBranchReturnsErrSourceBusy(t *testing.T) {
	ctx := context.Background()
	container, sourceConn, cfg := startPostgresContainer(t, ctx)
	defer func() {
		sourceConn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	maintenanceConn, err := ConnectToDatabase(ctx, cfg, "postgres")
	if err != nil {
		t.Fatalf("connect maintenance db: %v", err)
	}
	defer maintenanceConn.Close(ctx)

	branchName := fmt.Sprintf("dbfork_busy_%d", time.Now().UnixNano())
	err = CreateBranch(ctx, maintenanceConn, cfg.Database, branchName)
	if !errors.Is(err, ErrSourceBusy) {
		t.Fatalf("expected ErrSourceBusy, got %v", err)
	}
}
