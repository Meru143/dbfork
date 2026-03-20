// Package output renders dbfork terminal output.
package output

import (
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

// BranchRow is a rendered branch-list row.
type BranchRow struct {
	Active  string
	Name    string
	Source  string
	Created string
	SizeMB  string
	Status  string
}

// RenderBranchList renders the branch list to stdout.
func RenderBranchList(branches []BranchRow) {
	RenderBranchListTo(os.Stdout, branches)
}

// RenderBranchListTo renders the branch list to a writer.
func RenderBranchListTo(w io.Writer, branches []BranchRow) {
	table := tablewriter.NewWriter(w)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeader([]string{"", "BRANCH", "SOURCE", "CREATED", "SIZE (MB)", "STATUS"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_LEFT,
	})

	for _, branch := range branches {
		table.Append([]string{
			branch.Active,
			branch.Name,
			branch.Source,
			branch.Created,
			branch.SizeMB,
			branch.Status,
		})
	}

	table.Render()
}
