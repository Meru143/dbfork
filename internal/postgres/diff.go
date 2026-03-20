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
	Name           string
	Status         string
	Columns        []ColumnDiff
	AddedIndexes   []string
	DroppedIndexes []string
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

// DiffColumns compares ordered column metadata between source and branch.
func DiffColumns(sourceCols, branchCols []ColumnInfo) []ColumnDiff {
	sourceByName := make(map[string]ColumnInfo, len(sourceCols))
	for _, column := range sourceCols {
		sourceByName[column.Name] = column
	}

	var diffs []ColumnDiff
	seen := make(map[string]struct{}, len(branchCols))
	for _, branchColumn := range branchCols {
		sourceColumn, ok := sourceByName[branchColumn.Name]
		if !ok {
			diffs = append(diffs, ColumnDiff{
				Name:    branchColumn.Name,
				Status:  "added",
				NewType: branchColumn.DataType,
			})
			continue
		}

		seen[branchColumn.Name] = struct{}{}
		switch {
		case sourceColumn.DataType != branchColumn.DataType:
			diffs = append(diffs, ColumnDiff{
				Name:    branchColumn.Name,
				Status:  "type_changed",
				OldType: sourceColumn.DataType,
				NewType: branchColumn.DataType,
			})
		case sourceColumn.IsNullable != branchColumn.IsNullable:
			diffs = append(diffs, ColumnDiff{
				Name:   branchColumn.Name,
				Status: "nullability_changed",
			})
		}
	}

	for _, sourceColumn := range sourceCols {
		if _, ok := seen[sourceColumn.Name]; ok {
			continue
		}

		diffs = append(diffs, ColumnDiff{
			Name:    sourceColumn.Name,
			Status:  "dropped",
			OldType: sourceColumn.DataType,
		})
	}

	return diffs
}

// GetIndexes returns sorted index names for a table in the public schema.
func GetIndexes(ctx context.Context, conn *pgx.Conn, tableName string) ([]string, error) {
	rows, err := conn.Query(
		ctx,
		"SELECT indexname FROM pg_indexes WHERE schemaname='public' AND tablename=$1 ORDER BY indexname",
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			return nil, err
		}
		indexes = append(indexes, indexName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexes, nil
}

// DiffIndexes computes added and dropped index names between source and branch.
func DiffIndexes(sourceIdxs, branchIdxs []string) (added, dropped []string) {
	sourceSet := make(map[string]struct{}, len(sourceIdxs))
	for _, indexName := range sourceIdxs {
		sourceSet[indexName] = struct{}{}
	}

	branchSet := make(map[string]struct{}, len(branchIdxs))
	for _, indexName := range branchIdxs {
		branchSet[indexName] = struct{}{}
		if _, ok := sourceSet[indexName]; !ok {
			added = append(added, indexName)
		}
	}

	for _, indexName := range sourceIdxs {
		if _, ok := branchSet[indexName]; !ok {
			dropped = append(dropped, indexName)
		}
	}

	return added, dropped
}

// ComputeSchemaDiff computes a schema diff between the source and branch databases.
func ComputeSchemaDiff(ctx context.Context, sourceConn, branchConn *pgx.Conn) (*SchemaDiff, error) {
	sourceTables, err := GetTableNames(ctx, sourceConn, "public")
	if err != nil {
		return nil, err
	}

	branchTables, err := GetTableNames(ctx, branchConn, "public")
	if err != nil {
		return nil, err
	}

	addedTables, droppedTables := DiffTables(sourceTables, branchTables)
	diff := &SchemaDiff{
		AddedTables:   addedTables,
		DroppedTables: droppedTables,
	}

	branchSet := make(map[string]struct{}, len(branchTables))
	for _, tableName := range branchTables {
		branchSet[tableName] = struct{}{}
	}

	for _, tableName := range sourceTables {
		if _, ok := branchSet[tableName]; !ok {
			continue
		}

		sourceColumns, err := GetColumns(ctx, sourceConn, "public", tableName)
		if err != nil {
			return nil, err
		}

		branchColumns, err := GetColumns(ctx, branchConn, "public", tableName)
		if err != nil {
			return nil, err
		}

		columnDiffs := DiffColumns(sourceColumns, branchColumns)

		sourceIndexes, err := GetIndexes(ctx, sourceConn, tableName)
		if err != nil {
			return nil, err
		}

		branchIndexes, err := GetIndexes(ctx, branchConn, tableName)
		if err != nil {
			return nil, err
		}

		addedIndexes, droppedIndexes := DiffIndexes(sourceIndexes, branchIndexes)
		if len(columnDiffs) == 0 && len(addedIndexes) == 0 && len(droppedIndexes) == 0 {
			continue
		}

		diff.ChangedTables = append(diff.ChangedTables, TableDiff{
			Name:           tableName,
			Status:         "changed",
			Columns:        columnDiffs,
			AddedIndexes:   addedIndexes,
			DroppedIndexes: droppedIndexes,
		})
	}

	return diff, nil
}
