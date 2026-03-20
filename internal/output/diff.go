// Package output renders dbfork terminal output.
package output

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"

	"github.com/owner/dbfork/internal/postgres"
)

var (
	addedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	droppedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	changedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// RenderDiff renders a schema diff to stdout.
func RenderDiff(diff *postgres.SchemaDiff) {
	RenderDiffTo(os.Stdout, diff)
}

// RenderDiffTo renders a schema diff to a writer.
func RenderDiffTo(w io.Writer, diff *postgres.SchemaDiff) {
	if len(diff.AddedTables) == 0 && len(diff.DroppedTables) == 0 && len(diff.ChangedTables) == 0 {
		_, _ = fmt.Fprintln(w, "No schema differences found.")
		return
	}

	_, _ = fmt.Fprintln(w, "Tables")
	for _, tableName := range diff.AddedTables {
		_, _ = fmt.Fprintf(w, "%s %s\n", addedStyle.Render("+"), addedStyle.Render(tableName))
	}
	for _, tableName := range diff.DroppedTables {
		_, _ = fmt.Fprintf(w, "%s %s\n", droppedStyle.Render("-"), droppedStyle.Render(tableName))
	}

	for _, table := range diff.ChangedTables {
		_, _ = fmt.Fprintf(w, "\n%s %s\n", changedStyle.Render("~"), changedStyle.Render(table.Name))
		for _, column := range table.Columns {
			switch column.Status {
			case "added":
				_, _ = fmt.Fprintf(w, "  %s column %s (%s)\n", addedStyle.Render("+"), addedStyle.Render(column.Name), column.NewType)
			case "dropped":
				_, _ = fmt.Fprintf(w, "  %s column %s (%s)\n", droppedStyle.Render("-"), droppedStyle.Render(column.Name), column.OldType)
			case "type_changed":
				_, _ = fmt.Fprintf(w, "  %s column %s: %s -> %s\n", changedStyle.Render("~"), changedStyle.Render(column.Name), column.OldType, column.NewType)
			case "nullability_changed":
				_, _ = fmt.Fprintf(w, "  %s column %s nullability changed\n", changedStyle.Render("~"), changedStyle.Render(column.Name))
			}
		}
		for _, indexName := range table.AddedIndexes {
			_, _ = fmt.Fprintf(w, "  %s index %s\n", addedStyle.Render("+"), addedStyle.Render(indexName))
		}
		for _, indexName := range table.DroppedIndexes {
			_, _ = fmt.Fprintf(w, "  %s index %s\n", droppedStyle.Render("-"), droppedStyle.Render(indexName))
		}
	}
}
