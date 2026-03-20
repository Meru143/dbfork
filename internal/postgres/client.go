// Package postgres provides PostgreSQL connection helpers for dbfork.
package postgres

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"

	"github.com/Meru143/dbfork/internal/config"
)

// BuildDSN creates a PostgreSQL connection string from config values.
func BuildDSN(cfg config.Config) string {
	userInfo := url.User(cfg.User)
	if cfg.Password != "" {
		userInfo = url.UserPassword(cfg.User, cfg.Password)
	}

	return (&url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   cfg.Database,
	}).String()
}

// Connect opens a PostgreSQL connection.
func Connect(ctx context.Context, dsn string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, dsn)
}

// ConnectToDatabase connects to a specific PostgreSQL database.
func ConnectToDatabase(ctx context.Context, cfg config.Config, dbName string) (*pgx.Conn, error) {
	cfg.Database = dbName
	return Connect(ctx, BuildDSN(cfg))
}

// Ping verifies that a PostgreSQL connection is alive.
func Ping(ctx context.Context, conn *pgx.Conn) error {
	var value int
	return conn.QueryRow(ctx, "SELECT 1").Scan(&value)
}

// HasCreateDBPrivilege reports whether the current user can create databases.
func HasCreateDBPrivilege(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var allowed bool
	err := conn.QueryRow(ctx, "SELECT rolcreatedb FROM pg_roles WHERE rolname = current_user").Scan(&allowed)
	if err != nil {
		return false, err
	}

	return allowed, nil
}
