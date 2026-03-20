package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Meru143/dbfork/internal/config"
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

func TestGetTableNamesReturnsSortedBaseTables(t *testing.T) {
	ctx := context.Background()
	container, conn, _ := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	if _, err := conn.Exec(ctx, "CREATE TABLE zebra (id BIGINT PRIMARY KEY)"); err != nil {
		t.Fatalf("create zebra table: %v", err)
	}
	if _, err := conn.Exec(ctx, "CREATE TABLE alpha (id BIGINT PRIMARY KEY)"); err != nil {
		t.Fatalf("create alpha table: %v", err)
	}
	if _, err := conn.Exec(ctx, "CREATE VIEW not_a_table AS SELECT id FROM alpha"); err != nil {
		t.Fatalf("create view: %v", err)
	}

	tables, err := GetTableNames(ctx, conn, "public")
	if err != nil {
		t.Fatalf("get table names: %v", err)
	}

	if len(tables) != 2 || tables[0] != "alpha" || tables[1] != "zebra" {
		t.Fatalf("expected tables [alpha zebra], got %v", tables)
	}
}

func TestGetColumnsReturnsColumnMetadataInOrdinalOrder(t *testing.T) {
	ctx := context.Background()
	container, conn, _ := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE widgets (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT ''::text
		)
	`); err != nil {
		t.Fatalf("create widgets table: %v", err)
	}

	columns, err := GetColumns(ctx, conn, "public", "widgets")
	if err != nil {
		t.Fatalf("get columns: %v", err)
	}

	if len(columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(columns))
	}

	if columns[0].Name != "id" || columns[0].DataType != "bigint" || columns[0].IsNullable != "NO" {
		t.Fatalf("unexpected first column: %+v", columns[0])
	}

	if columns[1].Name != "name" || columns[1].DataType != "text" || columns[1].IsNullable != "NO" {
		t.Fatalf("unexpected second column: %+v", columns[1])
	}

	if columns[2].Name != "description" || columns[2].DataType != "text" || columns[2].ColumnDefault == "" {
		t.Fatalf("unexpected third column: %+v", columns[2])
	}
}

func TestDiffColumnsDetectsAddedDroppedTypeAndNullabilityChanges(t *testing.T) {
	diffs := DiffColumns(
		[]ColumnInfo{
			{Name: "id", DataType: "bigint", IsNullable: "NO"},
			{Name: "email", DataType: "text", IsNullable: "YES"},
			{Name: "legacy_code", DataType: "text", IsNullable: "YES"},
		},
		[]ColumnInfo{
			{Name: "id", DataType: "uuid", IsNullable: "NO"},
			{Name: "email", DataType: "text", IsNullable: "NO"},
			{Name: "bio", DataType: "text", IsNullable: "YES"},
		},
	)

	if len(diffs) != 4 {
		t.Fatalf("expected 4 diffs, got %d: %+v", len(diffs), diffs)
	}

	if diffs[0].Name != "id" || diffs[0].Status != "type_changed" || diffs[0].OldType != "bigint" || diffs[0].NewType != "uuid" {
		t.Fatalf("unexpected type diff: %+v", diffs[0])
	}

	if diffs[1].Name != "email" || diffs[1].Status != "nullability_changed" {
		t.Fatalf("unexpected nullability diff: %+v", diffs[1])
	}

	if diffs[2].Name != "bio" || diffs[2].Status != "added" || diffs[2].NewType != "text" {
		t.Fatalf("unexpected added diff: %+v", diffs[2])
	}

	if diffs[3].Name != "legacy_code" || diffs[3].Status != "dropped" || diffs[3].OldType != "text" {
		t.Fatalf("unexpected dropped diff: %+v", diffs[3])
	}
}

func TestGetIndexesReturnsSortedIndexNames(t *testing.T) {
	ctx := context.Background()
	container, conn, _ := startPostgresContainer(t, ctx)
	defer func() {
		conn.Close(ctx)
		_ = container.Terminate(ctx)
	}()

	if _, err := conn.Exec(ctx, "CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL)"); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := conn.Exec(ctx, "CREATE INDEX idx_users_created_at ON users (created_at)"); err != nil {
		t.Fatalf("create created_at index: %v", err)
	}
	if _, err := conn.Exec(ctx, "CREATE UNIQUE INDEX idx_users_email ON users (email)"); err != nil {
		t.Fatalf("create email index: %v", err)
	}

	indexes, err := GetIndexes(ctx, conn, "users")
	if err != nil {
		t.Fatalf("get indexes: %v", err)
	}

	expected := []string{"idx_users_created_at", "idx_users_email", "users_pkey"}
	if len(indexes) != len(expected) {
		t.Fatalf("expected indexes %v, got %v", expected, indexes)
	}

	for i := range expected {
		if indexes[i] != expected[i] {
			t.Fatalf("expected indexes %v, got %v", expected, indexes)
		}
	}
}

func TestDiffIndexesReturnsAddedAndDroppedIndexes(t *testing.T) {
	added, dropped := DiffIndexes(
		[]string{"idx_users_email", "users_pkey"},
		[]string{"idx_users_created_at", "users_pkey"},
	)

	if len(added) != 1 || added[0] != "idx_users_created_at" {
		t.Fatalf("expected added indexes [idx_users_created_at], got %v", added)
	}

	if len(dropped) != 1 || dropped[0] != "idx_users_email" {
		t.Fatalf("expected dropped indexes [idx_users_email], got %v", dropped)
	}
}

func TestComputeSchemaDiffDetectsTableColumnAndIndexChanges(t *testing.T) {
	ctx := context.Background()
	container, sourceConn, cfg := startPostgresContainer(t, ctx)
	defer func() {
		_ = container.Terminate(ctx)
	}()

	if _, err := sourceConn.Exec(ctx, "CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL)"); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := sourceConn.Exec(ctx, "CREATE TABLE sessions (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL)"); err != nil {
		t.Fatalf("create sessions table: %v", err)
	}

	maintenanceConn, err := ConnectToDatabase(ctx, cfg, "postgres")
	if err != nil {
		t.Fatalf("connect to maintenance db: %v", err)
	}
	defer maintenanceConn.Close(ctx)

	branchName := fmt.Sprintf("dbfork_diff_%d", time.Now().UnixNano())
	sourceConn.Close(ctx)
	if err := CreateBranch(ctx, maintenanceConn, cfg.Database, branchName); err != nil {
		t.Fatalf("create branch db: %v", err)
	}
	defer func() {
		_ = DropBranch(ctx, maintenanceConn, branchName)
	}()

	sourceConn, err = ConnectToDatabase(ctx, cfg, cfg.Database)
	if err != nil {
		t.Fatalf("reconnect to source db: %v", err)
	}
	defer sourceConn.Close(ctx)

	branchConn, err := ConnectToDatabase(ctx, cfg, branchName)
	if err != nil {
		t.Fatalf("connect to branch db: %v", err)
	}
	defer branchConn.Close(ctx)

	if _, err := branchConn.Exec(ctx, "ALTER TABLE users ADD COLUMN bio TEXT"); err != nil {
		t.Fatalf("add column: %v", err)
	}
	if _, err := branchConn.Exec(ctx, "CREATE INDEX idx_users_email ON users (email)"); err != nil {
		t.Fatalf("create index: %v", err)
	}
	if _, err := branchConn.Exec(ctx, "DROP TABLE sessions"); err != nil {
		t.Fatalf("drop sessions table: %v", err)
	}

	diff, err := ComputeSchemaDiff(ctx, sourceConn, branchConn)
	if err != nil {
		t.Fatalf("compute schema diff: %v", err)
	}

	if len(diff.DroppedTables) != 1 || diff.DroppedTables[0] != "sessions" {
		t.Fatalf("expected dropped tables [sessions], got %+v", diff.DroppedTables)
	}

	if len(diff.ChangedTables) != 1 || diff.ChangedTables[0].Name != "users" {
		t.Fatalf("expected one changed users table, got %+v", diff.ChangedTables)
	}

	if len(diff.ChangedTables[0].Columns) != 1 || diff.ChangedTables[0].Columns[0].Name != "bio" || diff.ChangedTables[0].Columns[0].Status != "added" {
		t.Fatalf("expected bio column add diff, got %+v", diff.ChangedTables[0].Columns)
	}

	if len(diff.ChangedTables[0].AddedIndexes) != 1 || diff.ChangedTables[0].AddedIndexes[0] != "idx_users_email" {
		t.Fatalf("expected added indexes [idx_users_email], got %+v", diff.ChangedTables[0].AddedIndexes)
	}
}

func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, *pgx.Conn, config.Config) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("get mapped port: %v", err)
	}

	conn, err := ConnectToDatabase(ctx, config.Config{
		Host:     host,
		Port:     port.Int(),
		User:     "test",
		Password: "test",
		Database: "testdb",
	}, "testdb")
	if err != nil {
		t.Fatalf("connect to test postgres: %v", err)
	}

	cfg := config.Config{
		Host:     host,
		Port:     port.Int(),
		User:     "test",
		Password: "test",
		Database: "testdb",
	}

	return container, conn, cfg
}
