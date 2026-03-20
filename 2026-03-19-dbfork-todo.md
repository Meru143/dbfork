# 2026-03-19-dbfork-todo.md
# dbfork — Detailed TODO List

---

## Phase 1: Project Setup

### 1.1 Repository Initialization
- [x] Create `dbfork/` directory
- [x] Run `go mod init github.com/<owner>/dbfork`
- [x] Create `README.md` with project name, one-line description
- [x] Create `LICENSE` (MIT)
- [x] Create `.gitignore` (binaries, `*.db`, `.env`, `dist/`)
- [x] Create `CHANGELOG.md` with `## [Unreleased]`
- [x] Run `git init && git add -A && git commit -m "chore: initial scaffold"`

### 1.2 Directory Structure
- [x] Create `cmd/dbfork/` directory
- [x] Create `cmd/dbfork/main.go` with `func main() { cli.Execute() }`
- [x] Create `internal/cli/` directory
- [x] Create `internal/postgres/` directory
- [x] Create `internal/state/` directory
- [x] Create `internal/config/` directory
- [x] Create `internal/output/` directory
- [x] Create `test/integration/` directory
- [x] Create `test/fixtures/` directory
- [ ] Create `test/fixtures/schema.sql` with tables, indexes, and constraints for tests
- [x] Create `.github/workflows/` directory

### 1.3 Install Dependencies
- [ ] Run `go get github.com/spf13/cobra@v1.9.0`
- [ ] Run `go get github.com/spf13/viper@v1.20.0`
- [ ] Run `go get github.com/jackc/pgx/v5@v5.7.0`
- [ ] Run `go get github.com/olekukonko/tablewriter@v0.0.5`
- [ ] Run `go get github.com/charmbracelet/lipgloss@v1.0.0`
- [ ] Run `go get github.com/stretchr/testify@v1.10.0`
- [ ] Run `go get github.com/testcontainers/testcontainers-go@v0.35.0`
- [ ] Run `go mod tidy`

### 1.4 Root Cobra Command
- [ ] Create `internal/cli/root.go`
- [ ] Declare `var rootCmd = &cobra.Command{ Use: "dbfork", Short: "Git-style branching for local Postgres databases" }`
- [ ] Add `Execute()` function that calls `rootCmd.Execute()` and exits on error
- [ ] Add persistent flags: `--host`, `--port`, `--user`, `--password`, `--source`, `--verbose`
- [ ] Add `PersistentPreRun` that loads config via viper and overrides with flags

### 1.5 Config System
- [ ] Create `internal/config/loader.go`
- [ ] Define `Config` struct with fields: `Host`, `Port`, `User`, `Password`, `Database`
- [ ] In `LoadConfig()`: set viper config name to `config`, type to `toml`, path to `~/.dbfork/`
- [ ] Bind env vars: `viper.BindEnv("host", "DBFORK_HOST")`, repeat for all fields
- [ ] Set defaults: `viper.SetDefault("host", "localhost")`, `viper.SetDefault("port", 5432)`
- [ ] Unmarshal into `Config` struct using `viper.Unmarshal(&cfg)`
- [ ] Return `Config` and any error from `LoadConfig()`

### 1.6 State Store
- [ ] Create `internal/state/types.go` with `Branch` and `State` structs
- [ ] Add `Version int` field to `State` struct, set to `1`
- [ ] Create `internal/state/store.go`
- [ ] Implement `statePath() string` returning `filepath.Join(os.UserHomeDir(), ".dbfork", "state.json")`
- [ ] Implement `Load() (*State, error)` — read and `json.Unmarshal` state file, return empty `State{}` if file not found
- [ ] Implement `Save(s *State) error` — `json.MarshalIndent` with 2-space indent, write with `os.WriteFile`
- [ ] Create `~/.dbfork/` dir with `os.MkdirAll` if it doesn't exist in `Save()`
- [ ] Implement `AddBranch(s *State, b Branch)` — append to `s.Branches`
- [ ] Implement `RemoveBranch(s *State, name string)` — filter out by name
- [ ] Implement `FindBranch(s *State, name string) (*Branch, bool)` — return pointer and ok

### 1.7 Postgres Client
- [ ] Create `internal/postgres/client.go`
- [ ] Implement `BuildDSN(cfg config.Config) string` returning `postgres://user:pass@host:port/dbname`
- [ ] Implement `Connect(ctx context.Context, dsn string) (*pgx.Conn, error)` wrapping `pgx.Connect()`
- [ ] Implement `ConnectToDatabase(ctx context.Context, cfg config.Config, dbName string) (*pgx.Conn, error)` — builds DSN with given dbName, calls `Connect()`
- [ ] Implement `Ping(ctx context.Context, conn *pgx.Conn) error` with a simple `SELECT 1` query

### 1.8 Build and Release
- [ ] Create `.goreleaser.yml` with builds for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- [ ] Set archive format to `.tar.gz` for Unix, `.zip` for Windows
- [ ] Add `checksum: name_template: "checksums.txt"` to goreleaser config
- [ ] Create `Makefile` with: `build`, `install`, `test`, `test-integration`, `lint`, `release-dry`
- [ ] Add `make build` target: `go build -o bin/dbfork ./cmd/dbfork/`
- [ ] Add `make install` target: `go install ./cmd/dbfork/`

---

## Phase 2: Postgres Operations

### 2.1 Branch Create (Database Template)
- [ ] Create `internal/postgres/branch.go`
- [ ] Implement `CreateBranch(ctx context.Context, conn *pgx.Conn, sourceName, branchName string) error`
- [ ] Execute: `conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", pgx.Identifier{branchName}.Sanitize(), pgx.Identifier{sourceName}.Sanitize()))`
- [ ] Set statement timeout of 60s using `SET LOCAL statement_timeout = '60s'` before DDL
- [ ] Wrap error: detect `pgerror code "42P04"` (duplicate database) and return named error `ErrBranchExists`
- [ ] Detect `pgerror code "55006"` (object in use) and return named error `ErrSourceBusy`

### 2.2 Terminate Connections
- [ ] Create `internal/postgres/connections.go`
- [ ] Implement `TerminateIdleConnections(ctx context.Context, conn *pgx.Conn, dbName string) (int, error)`
- [ ] Execute: `SELECT COUNT(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid() AND state = 'idle'`
- [ ] Return count of terminated connections
- [ ] Implement `CountActiveConnections(ctx context.Context, conn *pgx.Conn, dbName string) (int, error)`
- [ ] Execute: `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`

### 2.3 Drop Database
- [ ] In `internal/postgres/branch.go`
- [ ] Implement `DropBranch(ctx context.Context, conn *pgx.Conn, branchName string) error`
- [ ] First call `TerminateIdleConnections()` for the branch database
- [ ] Execute: `conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgx.Identifier{branchName}.Sanitize()))`

### 2.4 Database Size
- [ ] Create `internal/postgres/size.go`
- [ ] Implement `GetDatabaseSizeMB(ctx context.Context, conn *pgx.Conn, dbName string) (float64, error)`
- [ ] Execute: `SELECT pg_database_size($1) / 1024.0 / 1024.0`
- [ ] Scan result into `float64`
- [ ] Return `0.0` if database not found (catch `pgconn.PgError` code `3D000`)

### 2.5 Check Database Exists
- [ ] In `internal/postgres/branch.go`
- [ ] Implement `DatabaseExists(ctx context.Context, conn *pgx.Conn, dbName string) (bool, error)`
- [ ] Execute: `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
- [ ] Scan into `bool` result

### 2.6 Check CREATEDB Privilege
- [ ] In `internal/postgres/client.go`
- [ ] Implement `HasCreateDBPrivilege(ctx context.Context, conn *pgx.Conn) (bool, error)`
- [ ] Execute: `SELECT rolcreatedb FROM pg_roles WHERE rolname = current_user`
- [ ] Scan into `bool`

---

## Phase 3: Schema Diff

### 3.1 Table Diff
- [ ] Create `internal/postgres/diff.go`
- [ ] Define `SchemaDiff` struct: `{ AddedTables []string; DroppedTables []string; ChangedTables []TableDiff }`
- [ ] Implement `GetTableNames(ctx context.Context, conn *pgx.Conn, schema string) ([]string, error)`
- [ ] Execute: `SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE' ORDER BY table_name`
- [ ] Implement `DiffTables(source, branch []string) (added, dropped []string)`
- [ ] Compute added tables: in branch but not in source
- [ ] Compute dropped tables: in source but not in branch

### 3.2 Column Diff
- [ ] In `internal/postgres/diff.go`
- [ ] Define `ColumnInfo` struct: `{ Name string; DataType string; IsNullable string; ColumnDefault string }`
- [ ] Implement `GetColumns(ctx context.Context, conn *pgx.Conn, schema, tableName string) ([]ColumnInfo, error)`
- [ ] Execute: `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`
- [ ] Implement `DiffColumns(sourceCols, branchCols []ColumnInfo) []ColumnDiff`
- [ ] For each column: detect added, dropped, type changed, nullability changed

### 3.3 Index Diff
- [ ] In `internal/postgres/diff.go`
- [ ] Implement `GetIndexes(ctx context.Context, conn *pgx.Conn, tableName string) ([]string, error)`
- [ ] Execute: `SELECT indexname FROM pg_indexes WHERE schemaname='public' AND tablename=$1 ORDER BY indexname`
- [ ] Implement `DiffIndexes(sourceIdxs, branchIdxs []string) (added, dropped []string)`

### 3.4 Full Diff Orchestrator
- [ ] In `internal/postgres/diff.go`
- [ ] Implement `ComputeSchemaDiff(ctx context.Context, sourceConn, branchConn *pgx.Conn) (*SchemaDiff, error)`
- [ ] Get table names from both DBs
- [ ] Diff tables: record added/dropped
- [ ] For each shared table: get columns from both, diff them, add to `ChangedTables` if any diffs found
- [ ] For each shared table: get indexes from both, diff them

---

## Phase 4: Branch Name Validation

### 4.1 Name Sanitizer
- [ ] Create `internal/cli/names.go`
- [ ] Implement `SanitizeBranchName(input string) (string, error)`
- [ ] Return error if input is empty
- [ ] Return error if input length > 50 chars (leave room for `dbfork_` prefix + Postgres 63 char limit)
- [ ] Convert to lowercase using `strings.ToLower`
- [ ] Replace spaces and hyphens with underscores
- [ ] Strip any characters not matching `[a-z0-9_]`
- [ ] Prepend `dbfork_` prefix
- [ ] Return error if sanitized name is `dbfork_` (input was entirely invalid chars)
- [ ] Implement `DisplayName(dbName string) string` — strips `dbfork_` prefix for display

---

## Phase 5: CLI Commands

### 5.1 `dbfork init`
- [ ] Create `internal/cli/init.go`
- [ ] Register `initCmd` with cobra: `Use: "init"`
- [ ] Create `~/.dbfork/` directory with `os.MkdirAll`
- [ ] If `~/.dbfork/config.toml` exists, print "Config already exists" and exit 0
- [ ] Prompt for host (default: `localhost`), port (default: `5432`), user (default: `$USER`), password, database name
- [ ] Test connection using the entered values: call `Connect()` and `Ping()`
- [ ] If connection succeeds: write TOML config to `~/.dbfork/config.toml` using `viper.WriteConfigAs()`
- [ ] Print success message with config file path

### 5.2 `dbfork create`
- [ ] Create `internal/cli/create.go`
- [ ] Register `createCmd`: `Use: "create <name>"`
- [ ] Mark `args` as `cobra.ExactArgs(1)`
- [ ] Add `--source` flag (string, defaults to config `default_database`)
- [ ] Load config and state
- [ ] Call `SanitizeBranchName(args[0])`
- [ ] Check state for existing branch with same name — error `E001` if found
- [ ] Connect to `postgres` (maintenance DB) using `ConnectToDatabase(ctx, cfg, "postgres")`
- [ ] Call `HasCreateDBPrivilege()` — error `E004` if false
- [ ] Call `DatabaseExists()` for branch name in Postgres — error `E001` if already exists in PG
- [ ] Call `TerminateIdleConnections()` for source DB — print count of terminated connections
- [ ] Call `CountActiveConnections()` — warn if > 0 active connections remain
- [ ] Show spinner using `lipgloss` animation during `CreateBranch()`
- [ ] Call `CreateBranch(ctx, conn, source, branchName)`
- [ ] On success: build `Branch` struct, call `AddBranch()`, call `state.Save()`
- [ ] Print: `✓ Created branch 'feature-add-users'`
- [ ] Print: `  Connect: dbfork connect feature-add-users`

### 5.3 `dbfork list`
- [ ] Create `internal/cli/list.go`
- [ ] Register `listCmd`: `Use: "list"`, aliases `["ls"]`
- [ ] Load state and config
- [ ] If no branches, print "No branches. Run 'dbfork create <name>' to get started."
- [ ] Connect to `postgres` maintenance DB
- [ ] For each branch in state: call `GetDatabaseSizeMB()` and `DatabaseExists()` (mark as "orphaned" if exists in state but not in PG)
- [ ] Read `.dbfork` file in current dir to identify active branch
- [ ] Render with `tablewriter`: columns `Active`, `Name`, `Source`, `Created`, `Size (MB)`, `Status`
- [ ] Mark active branch row with `*` in Active column

### 5.4 `dbfork connect`
- [ ] Create `internal/cli/connect.go`
- [ ] Register `connectCmd`: `Use: "connect <name>"`
- [ ] Mark args as `cobra.ExactArgs(1)`
- [ ] Add `--format` flag with choices `url`, `env`, `psql`
- [ ] Add `--psql` bool flag shortcut
- [ ] Load state, find branch by display name — error if not found
- [ ] Build DSN from branch fields
- [ ] If `--format url`: print DSN URL to stdout
- [ ] If `--format env`: print `export DATABASE_URL=<dsn>`
- [ ] If `--psql` or `--format psql`: call `exec.Command("psql", "-d", dsn).Run()` and pass stdin/stdout

### 5.5 `dbfork drop`
- [ ] Create `internal/cli/drop.go`
- [ ] Register `dropCmd`: `Use: "drop <name>"`
- [ ] Mark args as `cobra.ExactArgs(1)`
- [ ] Add `--force` bool flag
- [ ] Load state, find branch — error if not found
- [ ] If not `--force`: prompt `"Drop branch 'feature-x'? This cannot be undone. [y/N]: "`
- [ ] Read from `bufio.NewReader(os.Stdin)` — exit if not "y"
- [ ] Connect to maintenance DB
- [ ] Call `DropBranch()` — this terminates connections first, then drops DB
- [ ] Call `RemoveBranch()` and `state.Save()`
- [ ] Print: `✓ Dropped branch 'feature-add-users'`

### 5.6 `dbfork diff`
- [ ] Create `internal/cli/diff.go`
- [ ] Register `diffCmd`: `Use: "diff <name>"`
- [ ] Mark args as `cobra.ExactArgs(1)`
- [ ] Add `--format` flag with choices `text`, `json`
- [ ] Load state, find branch and its source
- [ ] Connect to both source and branch databases (two separate `pgx.Conn` instances)
- [ ] Call `ComputeSchemaDiff(ctx, sourceConn, branchConn)`
- [ ] If `--format json`: marshal `SchemaDiff` to JSON and print
- [ ] If `--format text` (default): call `output.RenderDiff(diff)` for colored output

### 5.7 `dbfork use`
- [ ] Create `internal/cli/use.go`
- [ ] Register `useCmd`: `Use: "use <name>"`
- [ ] Mark args as `cobra.ExactArgs(1)`
- [ ] Load state, find branch by display name
- [ ] Write display name to `.dbfork` in `os.Getwd()`
- [ ] Print: `✓ Switched to branch 'feature-add-users'`
- [ ] Print: `  DATABASE_URL=<dsn>`

---

## Phase 6: Output Formatters

### 6.1 Table Output
- [ ] Create `internal/output/table.go`
- [ ] Implement `RenderBranchList(branches []BranchRow)` using `tablewriter.NewWriter(os.Stdout)`
- [ ] Set `table.SetBorder(false)` and `table.SetColumnSeparator("  ")`
- [ ] Set header: `["", "BRANCH", "SOURCE", "CREATED", "SIZE (MB)", "STATUS"]`
- [ ] Set alignment: all LEFT except SIZE (RIGHT)
- [ ] Call `table.Render()`

### 6.2 Diff Renderer
- [ ] Create `internal/output/diff.go`
- [ ] Import `lipgloss` for color styling
- [ ] Define styles: `addedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))` (green)
- [ ] Define: `droppedStyle` (red, color "9"), `changedStyle` (yellow, color "11")
- [ ] Implement `RenderDiff(diff *postgres.SchemaDiff)`
- [ ] Print section "Tables" with added/dropped table names
- [ ] For each `ChangedTable`: print table name, then each column diff with +/- prefix and color
- [ ] Print "No schema differences found." when diff is empty

---

## Phase 7: Unit Tests

### 7.1 State Store Tests
- [ ] Create `internal/state/store_test.go`
- [ ] Test `Load()` returns empty state when file not found
- [ ] Test `Save()` + `Load()` round-trip
- [ ] Test `AddBranch()` appends correctly
- [ ] Test `RemoveBranch()` removes by name, leaves others intact
- [ ] Test `FindBranch()` returns correct branch and true
- [ ] Test `FindBranch()` returns nil and false for missing name

### 7.2 Config Tests
- [ ] Create `internal/config/loader_test.go`
- [ ] Test `LoadConfig()` with a TOML fixture file returns correct `Config`
- [ ] Test env var `DBFORK_HOST` overrides config file value
- [ ] Test defaults applied when no config file present

### 7.3 Name Sanitizer Tests
- [ ] Create `internal/cli/names_test.go`
- [ ] Test `SanitizeBranchName("feature-add-users")` → `"dbfork_feature_add_users"`
- [ ] Test `SanitizeBranchName("")` → error
- [ ] Test `SanitizeBranchName("MY BRANCH!")` → `"dbfork_my_branch"` (stripped special chars)
- [ ] Test `SanitizeBranchName("a" * 60)` → error (too long)
- [ ] Test `DisplayName("dbfork_feature_add_users")` → `"feature_add_users"`

### 7.4 DSN Builder Tests
- [ ] Create `internal/postgres/client_test.go`
- [ ] Test `BuildDSN()` with all fields → correct URL format `postgres://user:pass@host:5432/db`
- [ ] Test `BuildDSN()` with empty password → URL omits password `postgres://user@host:5432/db`

---

## Phase 8: Integration Tests

### 8.1 Test Container Setup
- [ ] Create `test/integration/main_test.go`
- [ ] In `TestMain(m *testing.M)`: call `testcontainers.GenericContainer()` with `postgres:17-alpine` image
- [ ] Set env vars: `POSTGRES_PASSWORD=test`, `POSTGRES_USER=test`, `POSTGRES_DB=testdb`
- [ ] Wait for container using `wait.ForLog("database system is ready")`
- [ ] Get mapped port: `container.MappedPort(ctx, "5432")`
- [ ] Set global `testConn` connection for all tests
- [ ] Execute `test/fixtures/schema.sql` to set up test tables
- [ ] Call `defer container.Terminate(ctx)` to clean up

### 8.2 Create Branch Tests
- [ ] Create `test/integration/create_test.go`
- [ ] `TestCreateBranch`: call `CreateBranch()` with unique name, assert `DatabaseExists()` returns true
- [ ] `TestCreateBranchHasSameSchema`: after create, connect to branch DB, query `information_schema.tables`, assert same table names as source
- [ ] `TestCreateBranchDuplicateName`: create same branch twice, assert second call returns `ErrBranchExists`
- [ ] `TestDropBranch`: create branch, call `DropBranch()`, assert `DatabaseExists()` returns false

### 8.3 Diff Tests
- [ ] Create `test/integration/diff_test.go`
- [ ] `TestDiffEmpty`: create branch, call `ComputeSchemaDiff()`, assert all arrays empty
- [ ] `TestDiffAddedColumn`: create branch, execute `ALTER TABLE users ADD COLUMN bio TEXT` on branch, call diff, assert `bio` in `ColumnDiff` with status `added`
- [ ] `TestDiffDroppedTable`: create branch, `DROP TABLE sessions` on branch, call diff, assert `sessions` in `DroppedTables`

### 8.4 Connections Tests
- [ ] Create `test/integration/connections_test.go`
- [ ] `TestTerminateIdleConnections`: open idle connection to source DB in goroutine, call `TerminateIdleConnections()`, assert count = 1
- [ ] `TestGetDatabaseSizeMB`: call `GetDatabaseSizeMB()` on test DB, assert returned value > 0

---

## Phase 9: CI/CD Pipeline

### 9.1 GitHub Actions CI
- [ ] Create `.github/workflows/ci.yml` with `on: [push, pull_request]`
- [ ] Job `lint`: run `golangci-lint run ./...`
- [ ] Job `test`: run `go test ./internal/...`
- [ ] Job `integration`: run `go test ./test/integration/... -tags integration` (requires Docker)
- [ ] Matrix for integration: `postgres_version: [14, 15, 16, 17]` — pass as env var to testcontainers
- [ ] Job `build`: run `go build ./cmd/dbfork/`

### 9.2 Release Workflow
- [ ] Create `.github/workflows/release.yml` triggered on tag `v*`
- [ ] Use `goreleaser/goreleaser-action@v6`
- [ ] Pass `GITHUB_TOKEN` secret

### 9.3 Code Quality
- [ ] Create `.golangci.yml` enabling: `errcheck`, `govet`, `staticcheck`, `unused`, `gosec`
- [ ] Add `govulncheck` step to CI

---

## Phase 10: Documentation

### 10.1 README.md
- [ ] Add Go version badge, CI badge, license badge
- [ ] Add "Why dbfork?" section with `CREATE DATABASE TEMPLATE` vs `pg_dump` timing comparison
- [ ] Add Installation section: `go install`, Homebrew (upcoming), binary download
- [ ] Add Quick Start (5 commands to get running)
- [ ] Add Command Reference table
- [ ] Add Config Reference section
- [ ] Add "How it works" section explaining `CREATE DATABASE TEMPLATE` mechanism
- [ ] Add Limitations section noting template connection restriction

### 10.2 Community Files
- [ ] Create `CONTRIBUTING.md` with dev setup (`docker-compose up` for Postgres)
- [ ] Create `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1)
- [ ] Create `SECURITY.md`
- [ ] Create `.github/ISSUE_TEMPLATE/bug_report.md`
- [ ] Create `.github/PULL_REQUEST_TEMPLATE.md`
- [ ] Create `.editorconfig` with `indent_style=tab` (Go convention)
- [ ] Create `docker-compose.yml` with `postgres:17-alpine` service for local dev
