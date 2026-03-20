// Package postgres provides PostgreSQL database size helpers for dbfork.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// GetDatabaseSizeMB returns the size of a database in megabytes.
func GetDatabaseSizeMB(ctx context.Context, conn *pgx.Conn, dbName string) (float64, error) {
	var sizeMB float64
	err := conn.QueryRow(ctx, "SELECT pg_database_size($1) / 1024.0 / 1024.0", dbName).Scan(&sizeMB)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
			return 0.0, nil
		}
		return 0.0, err
	}

	return sizeMB, nil
}
