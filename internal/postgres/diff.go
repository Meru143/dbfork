// Package postgres provides schema diff helpers for dbfork.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SchemaDiff captures structural differences between two databases.
type SchemaDiff struct {
	AddedTables   []string
	DroppedTables []string
	ChangedTables []TableDiff
}

// TableDiff describes changes to a table.
type TableDiff struct {
	Name    string
	Status  string
	Columns []ColumnDiff
}

// ColumnDiff describes changes to a column.
type ColumnDiff struct {
	Name    string
	Status  string
	OldType string
	NewType string
}

// ColumnInfo describes a database column.
type ColumnInfo struct {
	Name          string
	DataType      string
	IsNullable    string
	ColumnDefault string
}

// DiffTables computes added and dropped tables between source and branch.
func DiffTables(source, branch []string) (added, dropped []string) {
	sourceSet := make(map[string]struct{}, len(source))
	for _, table := range source {
		sourceSet[table] = struct{}{}
	}

	branchSet := make(map[string]struct{}, len(branch))
	for _, table := range branch {
		branchSet[table] = struct{}{}
		if _, ok := sourceSet[table]; !ok {
			added = append(added, table)
		}
	}

	for _, table := range source {
		if _, ok := branchSet[table]; !ok {
			dropped = append(dropped, table)
		}
	}

	return added, dropped
}

// GetTableNames returns sorted base table names from a schema.
func GetTableNames(ctx context.Context, conn *pgx.Conn, schema string) ([]string, error) {
	rows, err := conn.Query(
		ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE' ORDER BY table_name",
		schema,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// GetColumns returns ordered column metadata for a table.
func GetColumns(ctx context.Context, conn *pgx.Conn, schema, tableName string) ([]ColumnInfo, error) {
	rows, err := conn.Query(
		ctx,
		"SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position",
		schema,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var (
			column        ColumnInfo
			columnDefault *string
		)
		if err := rows.Scan(&column.Name, &column.DataType, &column.IsNullable, &columnDefault); err != nil {
			return nil, err
		}
		if columnDefault != nil {
			column.ColumnDefault = *columnDefault
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}
