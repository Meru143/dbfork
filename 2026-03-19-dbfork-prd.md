# 2026-03-19-dbfork-prd.md
# dbfork — Local Database Branching CLI

---

## Section 1 — Project Overview

**Name:** dbfork  
**Type:** CLI Developer Tool  
**Language/Runtime:** Go 1.24  
**License:** MIT  

`dbfork` brings git-style database branching to your local development environment — no cloud account, no SaaS dependency, no modified Postgres. You run `dbfork create feature-x` and get an instant, fully isolated copy of your local Postgres database that your app can connect to and write to freely. When done, `dbfork diff` shows you a schema-level and data-level diff between your branch and main, and `dbfork drop` tears it down. Built on Postgres's native `CREATE DATABASE ... TEMPLATE` mechanism, branches are created in seconds regardless of database size. Works with standard Postgres 14–17 running locally or in Docker — zero client-side storage layers required.

---

## Section 2 — Problem Statement

- Developers testing database migrations must either nuke-and-reseed their development database, maintain multiple parallel database instances manually, or rely on a staging database shared with teammates.
- Neon, Xata, Supabase, and Vela all provide database branching but only as cloud SaaS with network latency, account requirements, and cost.
- Neon's OSS repo implements copy-on-write at the storage engine level — it is not a simple local tool, it is an entirely custom Postgres fork.
- `pg_dump` + `pg_restore` is the only local alternative and it is prohibitively slow for databases over a few hundred MB.
- Developers working on migrations with real-world data volumes (millions of rows) have no fast, local, offline branching option.
- Teams that use Docker Compose for local development cannot easily integrate cloud-only branching tools.

---

## Section 3 — Solution

1. Use Postgres's built-in `CREATE DATABASE <branch_name> TEMPLATE <source_db>` DDL to create an instant, full copy of the source database. This copies all objects, schema, and data in a single server-side operation — no data transfer over the network.
2. Store branch metadata (source DB, created_at, connection string) in a local JSON config at `~/.dbfork/state.json`.
3. Provide `dbfork create <name>`, `dbfork list`, `dbfork connect <name>`, `dbfork diff <name>`, `dbfork drop <name>`, and `dbfork use <name>` commands.
4. `dbfork use <name>` writes the branch database name into a `.dbfork` file in the current directory that tools can read to auto-switch connection strings (similar to `.nvmrc` or `.ruby-version`).
5. `dbfork diff <name>` compares schema between the branch and its source using `pg_catalog` queries and reports structural differences.

---

## Section 4 — Target Users

**Primary:** Backend developers working on Postgres-backed applications locally who run database migrations as part of their development workflow. They want to test a migration on real data without risking their primary local database.

**Secondary:** Engineers working in teams that practice trunk-based development with feature flags — they want an isolated DB per feature branch, mirroring git branch workflows.

**Tertiary:** Developers writing and testing complex data transformations who need to iterate rapidly without reseeding.

---

## Section 5 — Tech Stack

| Component | Library | Version | Purpose |
|---|---|---|---|
| Language | Go | 1.24 | Compiled binary, fast startup |
| CLI Framework | cobra | 1.9.x | Subcommands, flags, help generation |
| Postgres Driver | pgx/v5 | 5.7.x | Connect to Postgres, execute DDL |
| Config | viper | 1.20.x | Load config from `~/.dbfork/config.toml` |
| State Store | encoding/json (stdlib) | — | Branch metadata in `~/.dbfork/state.json` |
| Table Output | tablewriter | 0.0.5 | Render branch listing as CLI table |
| Color | lipgloss | 2.x | Terminal styling |
| Testing | testify | 1.10.x | Unit test assertions |
| Integration Test DB | testcontainers-go | 0.35.x | Spin up real Postgres for integration tests |
| Release | goreleaser | 2.x | Cross-platform binary releases |
| Lint | golangci-lint | 1.64.x | Code quality |

**Why `CREATE DATABASE ... TEMPLATE` over pg_dump?** The `TEMPLATE` mechanism is a server-side operation — Postgres copies the source database's file blocks directly on disk. It does not transfer data through the client. For a 1GB database, `pg_dump | pg_restore` takes minutes; `CREATE DATABASE ... TEMPLATE` takes seconds. This is the same technique that made Neon's branching fast, but using a standard Postgres feature available since Postgres 8.

**Why Go over TypeScript?** Single compiled binary with no runtime dependency. Users install by downloading one file. No `npm install`, no Node.js version management. Go's `pgx/v5` is the most complete Postgres driver available.

**Why cobra over urfave/cli?** Cobra is the standard for complex CLIs in the Go ecosystem (used by kubectl, hugo, helm). It has better shell completion generation and a more active community.

**Limitation to document clearly:** `CREATE DATABASE ... TEMPLATE` requires that no other session is connected to the template database at the moment of cloning. `dbfork create` must terminate idle connections to the source database before running the template command (using `pg_terminate_backend`).

---

## Section 6 — Core Features (v1)

**1. Branch Creation (`dbfork create <name>`)**
- Terminate idle connections to source DB to allow template cloning
- Execute `CREATE DATABASE <name> TEMPLATE <source>` via pgx
- Store branch metadata in `~/.dbfork/state.json`: name, source, created_at, connection DSN
- Print connection string for the new branch database
- Support `--source <dbname>` flag to specify a non-default source database

**2. Branch Listing (`dbfork list`)**
- Read state file and display all branches as a formatted table
- Columns: `Name`, `Source`, `Created`, `Size (MB)`, `Status`
- Query Postgres `pg_database` to get current size using `pg_database_size()`
- Mark the currently active branch (from `.dbfork` file) with `*`

**3. Branch Connection (`dbfork connect <name>`)**
- Print the DSN for the branch database to stdout
- Support `--format url` (DSN URL) and `--format env` (exports `DATABASE_URL=...`)
- Support `--psql` flag: execute `psql -d <dsn>` directly

**4. Branch Dropping (`dbfork drop <name>`)**
- Terminate all connections to the branch database using `pg_terminate_backend`
- Execute `DROP DATABASE <name>`
- Remove branch record from `~/.dbfork/state.json`
- Require explicit `--force` flag or interactive confirmation

**5. Schema Diff (`dbfork diff <name>`)**
- Compare schema between branch and its source by querying `information_schema.tables`, `information_schema.columns`, and `pg_indexes` on both databases
- Report: new tables, dropped tables, added columns, dropped columns, changed column types, added/dropped indexes
- Output as human-readable diff or `--format json`

**6. Active Branch (`dbfork use <name>`)**
- Write `<name>` to `.dbfork` file in current directory
- Print the full DSN of the activated branch
- Applications can read `.dbfork` to auto-select their connection string (documented convention)

**7. Config (`~/.dbfork/config.toml`)**
- `default_host` (default: `localhost`)
- `default_port` (default: `5432`)
- `default_user` (default: OS username)
- `default_password`
- `default_database` (source database used when `--source` not specified)

---

## Section 7 — Interface Spec

### CLI Commands

```bash
# Create a branch of the current default DB
dbfork create feature-add-users

# Create from a specific source database
dbfork create experiment --source myapp_development

# List all branches
dbfork list

# Print connection string
dbfork connect feature-add-users
dbfork connect feature-add-users --format env
# → export DATABASE_URL=postgres://user:pass@localhost:5432/dbfork_feature_add_users

# Open psql against a branch
dbfork connect feature-add-users --psql

# Show schema diff
dbfork diff feature-add-users

# Mark a branch as active (writes .dbfork)
dbfork use feature-add-users

# Drop a branch (with confirmation)
dbfork drop feature-add-users

# Drop without confirmation prompt
dbfork drop feature-add-users --force

# Initialize config file
dbfork init
```

### Flags Table

| Flag | Type | Default | Description |
|---|---|---|---|
| `--source` | `string` | config `default_database` | Source database to branch from |
| `--host` | `string` | `localhost` | Postgres host |
| `--port` | `int` | `5432` | Postgres port |
| `--user` | `string` | OS username | Postgres user |
| `--password` | `string` | `""` | Postgres password |
| `--format` | `string` | `table` | Output format: `table`, `json`, `env` |
| `--force` | `bool` | `false` | Skip confirmation prompt |
| `--psql` | `bool` | `false` | Open psql after connect |
| `--verbose` | `bool` | `false` | Show SQL queries executed |

### Config File (`~/.dbfork/config.toml`)

```toml
default_host     = "localhost"
default_port     = 5432
default_user     = "postgres"
default_password = ""
default_database = "myapp_development"
```

---

## Section 8 — Data Flow Diagram

```
  ┌────────────────────────────────────────┐
  │         dbfork create <name>           │
  └─────────────────┬──────────────────────┘
                    │
      ┌─────────────▼─────────────┐
      │  Read config & state      │
      │  ~/.dbfork/config.toml    │
      │  ~/.dbfork/state.json     │
      └─────────────┬─────────────┘
                    │
      ┌─────────────▼─────────────┐
      │  Validate name is unique  │──► already exists? → Error E001
      │  Check Postgres connection│──► connection fail? → Error E002
      └─────────────┬─────────────┘
                    │
      ┌─────────────▼──────────────────────────────┐
      │  Terminate idle connections to source DB   │
      │  SELECT pg_terminate_backend(pid) FROM     │
      │  pg_stat_activity WHERE datname=$1          │
      │  AND pid <> pg_backend_pid()               │
      └─────────────┬──────────────────────────────┘
                    │
      ┌─────────────▼─────────────┐
      │  CREATE DATABASE <name>   │
      │  TEMPLATE <source>        │──► timeout? → Error E003
      └─────────────┬─────────────┘
                    │
      ┌─────────────▼─────────────┐
      │  Write to state.json      │
      │  { name, source, dsn,     │
      │    created_at }            │
      └─────────────┬─────────────┘
                    │
      ┌─────────────▼─────────────┐
      │  Print connection string  │
      │  and success message      │
      └───────────────────────────┘
```

---

## Section 9 — Architecture / Package Structure

```
dbfork/
├── cmd/
│   └── dbfork/
│       └── main.go             # Entry point: creates root cobra command, calls Execute()
├── internal/
│   ├── cli/
│   │   ├── root.go             # Root command, persistent flags, --verbose, --config
│   │   ├── create.go           # `dbfork create` subcommand
│   │   ├── list.go             # `dbfork list` subcommand
│   │   ├── connect.go          # `dbfork connect` subcommand
│   │   ├── drop.go             # `dbfork drop` subcommand
│   │   ├── diff.go             # `dbfork diff` subcommand
│   │   ├── use.go              # `dbfork use` subcommand
│   │   └── init.go             # `dbfork init` subcommand
│   ├── postgres/
│   │   ├── client.go           # pgx.Connect wrapper, DSN builder
│   │   ├── branch.go           # CREATE/DROP DATABASE, TEMPLATE operations
│   │   ├── connections.go      # pg_terminate_backend, pg_stat_activity queries
│   │   ├── diff.go             # information_schema queries for schema diff
│   │   └── size.go             # pg_database_size() query
│   ├── state/
│   │   ├── store.go            # Read/write ~/.dbfork/state.json
│   │   └── types.go            # Branch struct, State struct
│   ├── config/
│   │   └── loader.go           # viper config loading from ~/.dbfork/config.toml
│   └── output/
│       ├── table.go            # tablewriter formatted output
│       ├── json.go             # JSON marshaling for --format json
│       └── diff.go             # Colored diff rendering with lipgloss
├── test/
│   ├── integration/
│   │   ├── create_test.go      # testcontainers-go integration tests
│   │   └── diff_test.go
│   └── fixtures/
│       └── schema.sql          # Test schema for integration tests
├── .goreleaser.yml
├── Makefile
└── go.mod
```

**Key Go Structs:**

```go
// internal/state/types.go

type Branch struct {
    Name      string    `json:"name"`
    Source    string    `json:"source"`
    Host      string    `json:"host"`
    Port      int       `json:"port"`
    User      string    `json:"user"`
    Database  string    `json:"database"`   // actual DB name (prefixed)
    CreatedAt time.Time `json:"created_at"`
}

type State struct {
    Branches []Branch `json:"branches"`
}

// internal/postgres/diff.go

type TableDiff struct {
    Name    string
    Status  string   // "added" | "dropped" | "changed"
    Columns []ColumnDiff
}

type ColumnDiff struct {
    Name     string
    Status   string   // "added" | "dropped" | "type_changed"
    OldType  string
    NewType  string
}
```

---

## Section 10 — Error Handling

- **Branch name already exists:** Check state.json before attempting `CREATE DATABASE`. Exit 1 with clear message.
- **Postgres connection failure:** Wrap pgx dial error, suggest checking host/port/credentials.
- **Source database has active connections:** Warn user, attempt `pg_terminate_backend`, retry `CREATE DATABASE` once. If still blocked, exit with message to close active connections manually.
- **`CREATE DATABASE` timeout:** postgres-side DDL timeout. Print timeout error and suggest the source DB may be locked.
- **State file corrupted (invalid JSON):** Print error, suggest `dbfork repair` (v2 feature), exit 1.
- **Insufficient Postgres privileges:** `CREATE DATABASE` requires superuser or `CREATEDB` role. Print role-specific error message.

| Code | Meaning | User-Facing Message |
|---|---|---|
| `E001` | Branch name already exists | `"Branch 'feature-x' already exists. Use 'dbfork drop feature-x' to remove it."` |
| `E002` | Cannot connect to Postgres | `"Cannot connect to postgres://user@host:5432. Check your config with 'dbfork init'."` |
| `E003` | CREATE DATABASE timed out | `"Timed out waiting for database template operation. Close all connections to source DB."` |
| `E004` | Insufficient privileges | `"User '<user>' does not have CREATEDB privilege. Grant with: ALTER USER <user> CREATEDB;"` |
| `E005` | State file invalid | `"State file at ~/.dbfork/state.json is corrupted. Backup and delete it to reset."` |

---

## Section 11 — Edge Cases

1. **Source database has active connections** — `CREATE DATABASE ... TEMPLATE` requires no other sessions connected to the template. Must terminate idle connections via `pg_terminate_backend` and handle the case where active (non-idle) connections refuse to terminate.
2. **Branch name conflicts with existing Postgres database** — the branch database may already exist in Postgres but not in state.json (manual creation, or state was reset). Must check `pg_database` before attempting creation.
3. **Source database is being modified during template clone** — Postgres's TEMPLATE clone is instantaneous at the WAL level, but issuing modifications during the clone window causes an error. Advise user to pause writes or use `--wait-for-idle` flag (v2).
4. **Special characters in branch name** — branch name becomes a Postgres identifier. Must sanitize: only lowercase letters, digits, underscores. Prefix with `dbfork_` to avoid collisions with user databases.
5. **Postgres running in Docker** — host may be `127.0.0.1` vs `localhost` depending on Docker network config. `dbfork init` should test connection and surface the correct host.
6. **Multiple Postgres instances** — developer may have Postgres 14 and 17 running on different ports. Config must support per-instance settings (v2: named profiles).
7. **Large databases (>10GB)** — `CREATE DATABASE ... TEMPLATE` is server-side but still allocates disk space. Must warn user when available disk space is less than 2× source DB size.
8. **Source database not in `template1` lineage** — by default, Postgres only allows templating from `template0` (no active connections ever) or a database with no active sessions. The "terminate connections" step is critical.
9. **`dbfork drop` while app is connected** — must handle `ERROR: database "X" is being accessed by other users` by terminating connections first, like in `create`.
10. **State file version mismatch** — future versions of dbfork may change the state schema. Include a `version` field in `state.json` for forward compatibility.

---

## Section 12 — Testing Strategy

**Unit Tests:**
- Test `state/store.go` — read/write/update cycle with temp file
- Test branch name sanitization — special chars converted to underscores, prefix added
- Test DSN builder — correct URL format from config values
- Test `config/loader.go` — loads TOML, applies defaults for missing fields

**Integration Tests (using testcontainers-go):**
- Spin up `postgres:17-alpine` container
- Test `CREATE DATABASE ... TEMPLATE` produces exact copy of schema
- Test `dbfork diff` detects an added column after `ALTER TABLE` on the branch
- Test `pg_terminate_backend` terminates idle connections before clone
- Test `DROP DATABASE` succeeds after connections terminated

**Mocking Strategy:**
- No HTTP mocking needed (direct Postgres communication)
- Use real Postgres via testcontainers — do not mock pgx

---

## Section 13 — Distribution

```bash
# Install via Homebrew (post-1.0)
brew install dbfork

# Install via go install
go install github.com/<owner>/dbfork/cmd/dbfork@latest

# Download binary directly (goreleaser)
curl -sSL https://github.com/<owner>/dbfork/releases/latest/download/dbfork_linux_amd64.tar.gz | tar xz
```

**Platforms via goreleaser:** Linux amd64/arm64, macOS amd64/arm64 (Universal Binary), Windows amd64.

**CI/CD:** GitHub Actions with `goreleaser/goreleaser-action@v6` on tag push. Publishes to GitHub Releases with SHA256 checksums.

---

## Section 14 — Differentiators

1. **vs Neon/Xata/Supabase (cloud branching):** All cloud-only, require accounts, add network latency to every query. `dbfork` is fully local, works offline, requires no account.
2. **vs `pg_dump + pg_restore`:** On a 1GB database, dump+restore takes 2–5 minutes. `CREATE DATABASE ... TEMPLATE` takes 1–3 seconds regardless of data size.
3. **vs Neon OSS repo:** Neon's OSS code is an entire custom Postgres storage engine — not installable as a tool. `dbfork` uses standard Postgres with no modifications.
4. **vs running multiple Docker containers:** Each additional Postgres container uses ~80MB+ RAM and requires manual connection string management. `dbfork` creates lightweight named databases within a single Postgres instance.

---

## Section 15 — Future Scope (v2+)

- [ ] Named connection profiles (support multiple Postgres instances)
- [ ] `dbfork merge` — generate a migration file representing diff between branch and source
- [ ] `dbfork sync` — pull latest data from source into an existing branch
- [ ] MySQL/MariaDB support via `CREATE DATABASE ... DEFAULT CHARACTER SET`
- [ ] Automatic `.env` file update when switching branches
- [ ] VS Code extension for branch switching
- [ ] `dbfork snapshot <name>` — named point-in-time snapshots within a branch
- [ ] Git hook integration: auto-create branch on `git checkout -b`, auto-drop on `git branch -d`

---

## Section 16 — Success Metrics

- [ ] `dbfork create` completes in under 5 seconds for a 500MB database
- [ ] Zero modification to Postgres installation required
- [ ] Works with Postgres 14, 15, 16, and 17
- [ ] Works with Postgres in Docker and native installations
- [ ] Single binary install with no dependencies
- [ ] macOS, Linux, and Windows binaries published on each release
- [ ] 80%+ unit test coverage (excluding integration tests)
- [ ] Integration tests pass against Postgres 14 and 17 simultaneously

---

## Section 17 — Additional Deliverables

**Documentation:**
- [ ] README.md with demo GIF, install options, and quick start
- [ ] CONTRIBUTING.md
- [ ] SECURITY.md
- [ ] CODE_OF_CONDUCT.md
- [ ] `.github/ISSUE_TEMPLATE/bug_report.md`

**Dev Environment:**
- [ ] `docker-compose.yml` with Postgres 17 for local dev
- [ ] `.devcontainer/devcontainer.json` with Go 1.24 image and Postgres service
- [ ] `.env.example`

**Environment Variables:**
- [ ] `DBFORK_HOST` — override default host
- [ ] `DBFORK_PORT` — override default port
- [ ] `DBFORK_USER` — override default user
- [ ] `DBFORK_PASSWORD` — postgres password
- [ ] `DBFORK_DATABASE` — override default source database
- [ ] `DBFORK_CONFIG` — override config file path

---

## Section 18 — Expanded Testing

**Unit Tests (target: 80%+ coverage):**
- [ ] `state/store.go` — read state from temp file
- [ ] `state/store.go` — write new branch to state
- [ ] `state/store.go` — update existing branch in state
- [ ] `state/store.go` — delete branch from state
- [ ] `state/store.go` — returns empty state when file does not exist
- [ ] `config/loader.go` — loads all TOML fields correctly
- [ ] `config/loader.go` — returns defaults when config file missing
- [ ] `config/loader.go` — `DBFORK_HOST` env var overrides config
- [ ] `postgres/client.go` — builds correct DSN URL from Config struct
- [ ] Branch name sanitizer — rejects name with spaces → error
- [ ] Branch name sanitizer — adds `dbfork_` prefix when not present
- [ ] Branch name sanitizer — truncates to 63 chars (Postgres identifier limit)

**Integration Tests:**
- [ ] `testcontainers.GenericContainer` with `postgres:17-alpine` image
- [ ] Create source DB with fixture schema via `schema.sql`
- [ ] `CREATE DATABASE ... TEMPLATE` — assert branch DB has same tables
- [ ] `pg_database_size()` — assert branch has non-zero size
- [ ] `ALTER TABLE` on branch — assert source unaffected
- [ ] `information_schema.columns` diff — assert added column detected
- [ ] `pg_terminate_backend` — create idle connection to source, assert terminate works
- [ ] `DROP DATABASE` — assert database no longer exists in `pg_database`

**E2E Tests:**
- [ ] Full `dbfork create` → `dbfork list` → `dbfork diff` → `dbfork drop` workflow
- [ ] `dbfork use` writes correct name to `.dbfork` file

**Test Infrastructure:**
- [ ] `testcontainers-go` container setup in `TestMain`
- [ ] Shared connection pool for integration test suite
- [ ] Unique database name per test (using `t.Name()` hash)
- [ ] Container teardown with `defer container.Terminate(ctx)`

---

## Section 19 — CI/CD Pipeline

**CI:**
- [ ] `.github/workflows/ci.yml` — push and PR triggers
- [ ] Job: `golangci-lint` with `.golangci.yml` config
- [ ] Job: `go test ./internal/...` (unit tests only, no Docker required)
- [ ] Job: `go test ./test/integration/...` (requires Docker, uses testcontainers)
- [ ] Matrix: Postgres 14, 15, 16, 17 (testcontainers image tag matrix)
- [ ] Job: `go build ./cmd/dbfork/...`

**Release:**
- [ ] `.github/workflows/release.yml` — trigger on tag `v*`
- [ ] Run `goreleaser release --clean` with `GITHUB_TOKEN`
- [ ] `.goreleaser.yml`: builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- [ ] Archive format: `.tar.gz` for Linux/macOS, `.zip` for Windows
- [ ] Include SHA256SUMS file in release

**Makefile:**
- [ ] `make lint`
- [ ] `make test`
- [ ] `make test-integration`
- [ ] `make build`
- [ ] `make install` (copies binary to `$GOPATH/bin`)
- [ ] `make release-dry` (`goreleaser release --snapshot --clean`)
