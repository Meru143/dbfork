//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/owner/dbfork/internal/postgres"
)

func TestDiffEmpty(t *testing.T) {
	ctx := context.Background()
	branchName := createTestBranch(t)
	conn := branchConn(t, branchName)

	diff, err := postgres.ComputeSchemaDiff(ctx, testConn, conn)
	if err != nil {
		t.Fatalf("compute schema diff: %v", err)
	}

	if len(diff.AddedTables) != 0 || len(diff.DroppedTables) != 0 || len(diff.ChangedTables) != 0 {
		t.Fatalf("expected empty diff, got %+v", diff)
	}
}

func TestDiffAddedColumn(t *testing.T) {
	ctx := context.Background()
	branchName := createTestBranch(t)
	conn := branchConn(t, branchName)

	if _, err := conn.Exec(ctx, "ALTER TABLE users ADD COLUMN bio TEXT"); err != nil {
		t.Fatalf("alter users table: %v", err)
	}

	diff, err := postgres.ComputeSchemaDiff(ctx, testConn, conn)
	if err != nil {
		t.Fatalf("compute schema diff: %v", err)
	}

	if len(diff.ChangedTables) != 1 || diff.ChangedTables[0].Name != "users" {
		t.Fatalf("expected users changed table, got %+v", diff.ChangedTables)
	}

	columns := diff.ChangedTables[0].Columns
	if len(columns) != 1 || columns[0].Name != "bio" || columns[0].Status != "added" {
		t.Fatalf("expected added bio column diff, got %+v", columns)
	}
}

func TestDiffDroppedTable(t *testing.T) {
	ctx := context.Background()
	branchName := createTestBranch(t)
	conn := branchConn(t, branchName)

	if _, err := conn.Exec(ctx, "DROP TABLE sessions"); err != nil {
		t.Fatalf("drop sessions table: %v", err)
	}

	diff, err := postgres.ComputeSchemaDiff(ctx, testConn, conn)
	if err != nil {
		t.Fatalf("compute schema diff: %v", err)
	}

	if len(diff.DroppedTables) != 1 || diff.DroppedTables[0] != "sessions" {
		t.Fatalf("expected dropped table sessions, got %+v", diff.DroppedTables)
	}
}
