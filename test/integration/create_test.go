//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/owner/dbfork/internal/postgres"
)

func TestCreateBranch(t *testing.T) {
	ctx := context.Background()
	conn := maintenanceConn(t)
	branchName := uniqueBranchName(t)

	closeSourceConn(t)
	err := postgres.CreateBranch(ctx, conn, testConfig.Database, branchName)
	reconnectSourceConn(t)
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}

	t.Cleanup(func() {
		_ = postgres.DropBranch(context.Background(), conn, branchName)
	})

	exists, err := postgres.DatabaseExists(ctx, conn, branchName)
	if err != nil {
		t.Fatalf("database exists: %v", err)
	}
	if !exists {
		t.Fatalf("expected branch database %q to exist", branchName)
	}
}

func TestCreateBranchHasSameSchema(t *testing.T) {
	ctx := context.Background()
	branchName := createTestBranch(t)
	conn := branchConn(t, branchName)

	tables, err := postgres.GetTableNames(ctx, conn, "public")
	if err != nil {
		t.Fatalf("get branch tables: %v", err)
	}

	expected := map[string]bool{
		"sessions": true,
		"users":    true,
	}
	if len(tables) != len(expected) {
		t.Fatalf("expected tables %v, got %v", expected, tables)
	}
	for _, tableName := range tables {
		if !expected[tableName] {
			t.Fatalf("unexpected table %q in branch schema", tableName)
		}
	}
}

func TestCreateBranchDuplicateName(t *testing.T) {
	ctx := context.Background()
	conn := maintenanceConn(t)
	branchName := uniqueBranchName(t)

	closeSourceConn(t)
	err := postgres.CreateBranch(ctx, conn, testConfig.Database, branchName)
	reconnectSourceConn(t)
	if err != nil {
		t.Fatalf("first create branch: %v", err)
	}

	t.Cleanup(func() {
		_ = postgres.DropBranch(context.Background(), conn, branchName)
	})

	closeSourceConn(t)
	err = postgres.CreateBranch(ctx, conn, testConfig.Database, branchName)
	reconnectSourceConn(t)
	if !errors.Is(err, postgres.ErrBranchExists) {
		t.Fatalf("expected ErrBranchExists, got %v", err)
	}
}

func TestDropBranch(t *testing.T) {
	ctx := context.Background()
	conn := maintenanceConn(t)
	branchName := createTestBranch(t)

	if err := postgres.DropBranch(ctx, conn, branchName); err != nil {
		t.Fatalf("drop branch: %v", err)
	}

	exists, err := postgres.DatabaseExists(ctx, conn, branchName)
	if err != nil {
		t.Fatalf("database exists: %v", err)
	}
	if exists {
		t.Fatalf("expected branch database %q to be dropped", branchName)
	}
}
