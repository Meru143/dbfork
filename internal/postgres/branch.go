// Package postgres provides PostgreSQL database branching helpers for dbfork.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrBranchExists indicates the target branch database already exists.
	ErrBranchExists = errors.New("branch already exists")
	// ErrSourceBusy indicates the source database still has blocking sessions.
	ErrSourceBusy = errors.New("source database is busy")
)

// CreateBranch clones a PostgreSQL database using TEMPLATE.
func CreateBranch(ctx context.Context, conn *pgx.Conn, sourceName, branchName string) error {
	// CREATE DATABASE cannot run inside a transaction, so session-level SET is
	// required here even though the TODO originally called for SET LOCAL.
	if _, err := conn.Exec(ctx, "SET statement_timeout = '60s'"); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "RESET statement_timeout")
	}()

	query := fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{branchName}.Sanitize(),
		pgx.Identifier{sourceName}.Sanitize(),
	)

	if _, err := conn.Exec(ctx, query); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "42P04":
				return ErrBranchExists
			case "55006":
				return ErrSourceBusy
			}
		}
		return err
	}

	return nil
}

// DropBranch drops a cloned PostgreSQL database.
func DropBranch(ctx context.Context, conn *pgx.Conn, branchName string) error {
	if _, err := TerminateIdleConnections(ctx, conn, branchName); err != nil {
		return err
	}

	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{branchName}.Sanitize())
	_, err := conn.Exec(ctx, query)
	return err
}

// DatabaseExists reports whether a PostgreSQL database exists.
func DatabaseExists(ctx context.Context, conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
