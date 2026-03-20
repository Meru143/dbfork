package cli

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "hyphens become underscores", input: "feature-add-users", want: "dbfork_feature_add_users"},
		{name: "empty input errors", input: "", wantError: true},
		{name: "uppercase spaces and punctuation sanitized", input: "MY BRANCH!", want: "dbfork_my_branch"},
		{name: "over max length errors", input: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", wantError: true},
		{name: "entirely invalid chars error", input: "!!!", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SanitizeBranchName(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}

			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("dbfork_feature_add_users"); got != "feature_add_users" {
		t.Fatalf("expected feature_add_users, got %q", got)
	}
}
