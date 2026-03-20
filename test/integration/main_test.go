//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
)

var (
	testContainer testcontainers.Container
	testConfig    config.Config
	testConn      *pgx.Conn
	cleanupOnce   sync.Once
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, cfg, conn, err := startContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start integration postgres: %v\n", err)
		os.Exit(1)
	}

	testContainer = container
	testConfig = cfg
	testConn = conn

	cleanup := func() {
		cleanupOnce.Do(func() {
			if testConn != nil {
				testConn.Close(ctx)
				testConn = nil
			}
			if testContainer != nil {
				_ = testContainer.Terminate(ctx)
			}
		})
	}
	defer cleanup()

	if err := loadFixtureSchema(ctx, testConn); err != nil {
		fmt.Fprintf(os.Stderr, "load integration schema: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	code := m.Run()
	cleanup()
	os.Exit(code)
}

func startContainer(ctx context.Context) (testcontainers.Container, config.Config, *pgx.Conn, error) {
	version := os.Getenv("POSTGRES_VERSION")
	if version == "" {
		version = "17"
	}

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("postgres:%s-alpine", version),
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready").WithOccurrence(1).WithStartupTimeout(2 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, config.Config{}, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, config.Config{}, nil, err
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, config.Config{}, nil, err
	}

	cfg := config.Config{
		Host:     host,
		Port:     port.Int(),
		User:     "test",
		Password: "test",
		Database: "testdb",
	}

	conn, err := postgres.ConnectToDatabase(ctx, cfg, cfg.Database)
	if err != nil {
		return nil, config.Config{}, nil, err
	}

	return container, cfg, conn, nil
}

func loadFixtureSchema(ctx context.Context, conn *pgx.Conn) error {
	fixturePath := filepath.Join("..", "fixtures", "schema.sql")
	schemaSQL, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, string(schemaSQL))
	return err
}

func maintenanceConn(t *testing.T) *pgx.Conn {
	t.Helper()

	conn, err := postgres.ConnectToDatabase(context.Background(), testConfig, "postgres")
	if err != nil {
		t.Fatalf("connect maintenance db: %v", err)
	}

	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

func closeSourceConn(t *testing.T) {
	t.Helper()

	if testConn != nil {
		testConn.Close(context.Background())
		testConn = nil
	}
}

func reconnectSourceConn(t *testing.T) {
	t.Helper()

	if testConn != nil {
		return
	}

	conn, err := postgres.ConnectToDatabase(context.Background(), testConfig, testConfig.Database)
	if err != nil {
		t.Fatalf("reconnect source db: %v", err)
	}

	testConn = conn
}

func createTestBranch(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	branchName := uniqueBranchName(t)
	conn := maintenanceConn(t)

	closeSourceConn(t)
	if err := postgres.CreateBranch(ctx, conn, testConfig.Database, branchName); err != nil {
		t.Fatalf("create branch %q: %v", branchName, err)
	}
	reconnectSourceConn(t)

	t.Cleanup(func() {
		cleanupConn, err := postgres.ConnectToDatabase(context.Background(), testConfig, "postgres")
		if err == nil {
			defer cleanupConn.Close(context.Background())
			_ = postgres.DropBranch(context.Background(), cleanupConn, branchName)
		}
	})

	return branchName
}

func branchConn(t *testing.T, branchName string) *pgx.Conn {
	t.Helper()

	conn, err := postgres.ConnectToDatabase(context.Background(), testConfig, branchName)
	if err != nil {
		t.Fatalf("connect branch db %q: %v", branchName, err)
	}

	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

func uniqueBranchName(t *testing.T) string {
	t.Helper()

	base := strings.ToLower(t.Name())
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "-", "_")
	base = replacer.Replace(base)
	base = strings.Trim(base, "_")
	if len(base) > 30 {
		base = base[:30]
	}

	return fmt.Sprintf("dbfork_it_%s_%d", base, time.Now().UnixNano())
}
