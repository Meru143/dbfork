package output

import (
	"bytes"
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
