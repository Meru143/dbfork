package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRenderBranchListToPrintsHeadersAndRows(t *testing.T) {
	var out bytes.Buffer

	RenderBranchListTo(&out, []BranchRow{
		{
			Active:  "*",
			Name:    "feature-add-users",
			Source:  "myapp_development",
			Created: "2026-03-20T12:00:00Z",
			SizeMB:  "12.5",
			Status:  "ready",
		},
	})

	got := out.String()
	for _, want := range []string{
		"BRANCH",
		"SOURCE",
		"CREATED",
		"SIZE (MB)",
		"STATUS",
		"feature-add-users",
		"myapp_development",
		"12.5",
		"ready",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rendered table to contain %q, got %q", want, got)
		}
	}

	if strings.Contains(got, "|") {
		t.Fatalf("expected borderless output, got %q", got)
	}
}

func TestRenderBranchListWritesToStdout(t *testing.T) {
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = writer
	RenderBranchList([]BranchRow{{Name: "feature-add-users", Source: "myapp_development", Created: "now", SizeMB: "1.0", Status: "ready"}})
	_ = writer.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	if !strings.Contains(string(data), "feature-add-users") {
		t.Fatalf("expected stdout output, got %q", string(data))
	}
}
