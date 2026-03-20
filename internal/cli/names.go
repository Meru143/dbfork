// Package cli provides branch naming helpers for dbfork commands.
package cli

import (
	"errors"
	"strings"
)

const branchPrefix = "dbfork_"

// SanitizeBranchName converts user input into a safe PostgreSQL database name.
func SanitizeBranchName(input string) (string, error) {
	if input == "" {
		return "", errors.New("branch name cannot be empty")
	}
	if len(input) > 50 {
		return "", errors.New("branch name must be 50 characters or fewer")
	}

	lowered := strings.ToLower(input)
	replaced := strings.NewReplacer(" ", "_", "-", "_").Replace(lowered)

	var builder strings.Builder
	for _, r := range replaced {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}

	sanitized := branchPrefix + builder.String()
	if sanitized == branchPrefix {
		return "", errors.New("branch name must contain at least one valid character")
	}

	return sanitized, nil
}

// DisplayName removes the dbfork prefix for user-facing output.
func DisplayName(dbName string) string {
	return strings.TrimPrefix(dbName, branchPrefix)
}
