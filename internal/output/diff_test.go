package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/owner/dbfork/internal/postgres"
)

func TestRenderDiffToPrintsEmptyMessage(t *testing.T) {
	var out bytes.Buffer

	RenderDiffTo(&out, &postgres.SchemaDiff{})

	if got := strings.TrimSpace(out.String()); got != "No schema differences found." {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRenderDiffToPrintsTableAndColumnChanges(t *testing.T) {
	var out bytes.Buffer

	RenderDiffTo(&out, &postgres.SchemaDiff{
		AddedTables:   []string{"widgets"},
		DroppedTables: []string{"sessions"},
		ChangedTables: []postgres.TableDiff{
			{
				Name: "users",
				Columns: []postgres.ColumnDiff{
					{Name: "bio", Status: "added", NewType: "text"},
					{Name: "nickname", Status: "dropped", OldType: "text"},
					{Name: "email", Status: "type_changed", OldType: "text", NewType: "citext"},
					{Name: "admin", Status: "nullability_changed"},
				},
				AddedIndexes:   []string{"idx_users_email"},
				DroppedIndexes: []string{"idx_users_legacy"},
			},
		},
	})

	got := out.String()
	for _, want := range []string{
		"Tables",
		"widgets",
		"sessions",
		"users",
		"column bio (text)",
		"column nickname (text)",
		"column email: text -> citext",
		"column admin nullability changed",
		"index idx_users_email",
		"index idx_users_legacy",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected diff output to contain %q, got %q", want, got)
		}
	}
}

func TestRenderDiffWritesToStdout(t *testing.T) {
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = writer
	RenderDiff(&postgres.SchemaDiff{AddedTables: []string{"widgets"}})
	_ = writer.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	if !strings.Contains(string(data), "widgets") {
		t.Fatalf("expected stdout diff output, got %q", string(data))
	}
}
