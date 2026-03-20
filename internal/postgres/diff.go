// Package postgres provides schema diff helpers for dbfork.
package postgres

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
