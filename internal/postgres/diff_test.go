package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/owner/dbfork/internal/config"
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
	container, conn := startPostgresContainer(t, ctx)
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
	container, conn := startPostgresContainer(t, ctx)
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

func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, *pgx.Conn) {
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

	return container, conn
}
