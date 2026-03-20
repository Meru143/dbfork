package postgres

import (
	"testing"
)

func TestDiffTablesReturnsAddedAndDroppedTables(t *testing.T) {
	added, dropped := DiffTables(
		[]string{"accounts", "users"},
		[]string{"profiles", "users"},
	)

	if len(added) != 1 || added[0] != "profiles" {
		t.Fatalf("expected added tables [profiles], got %v", added)
	}

	if len(dropped) != 1 || dropped[0] != "accounts" {
		t.Fatalf("expected dropped tables [accounts], got %v", dropped)
	}
}
