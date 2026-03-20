// Package postgres provides PostgreSQL connection management helpers for dbfork.
package postgres

import "context"

import "github.com/jackc/pgx/v5"

// TerminateIdleConnections closes idle sessions for a database.
func TerminateIdleConnections(ctx context.Context, conn *pgx.Conn, dbName string) (int, error) {
	var count int
	err := conn.QueryRow(
		ctx,
		"SELECT COUNT(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid() AND state = 'idle'",
		dbName,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountActiveConnections counts non-current sessions for a database.
func CountActiveConnections(ctx context.Context, conn *pgx.Conn, dbName string) (int, error) {
	var count int
	err := conn.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
		dbName,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
