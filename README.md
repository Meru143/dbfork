# dbfork

[![Go Version](https://img.shields.io/badge/go-1.25.8-00ADD8?logo=go)](https://go.dev/)
[![CI](https://github.com/Meru143/dbfork/actions/workflows/ci.yml/badge.svg)](https://github.com/Meru143/dbfork/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

Fast local PostgreSQL database branching for migrations, schema testing, and isolated development workflows.

`dbfork` is a Go CLI for local Postgres development. It creates instant database branches with PostgreSQL's native `CREATE DATABASE ... TEMPLATE` feature, so you can clone your development database in seconds, test migrations safely, inspect schema drift, and tear branches down when you are done.

- Clone a local PostgreSQL database in seconds instead of waiting on `pg_dump | pg_restore`
- Give each feature or migration its own disposable database branch
- Diff branch schema against the source database before you merge changes

## Why dbfork?

- Fast local branching: `CREATE DATABASE ... TEMPLATE` is a server-side copy, so PostgreSQL duplicates the database on disk without moving rows through the client.
- Faster than `pg_dump | pg_restore`: for a 1 GB development database, `pg_dump` plus restore usually takes 2 to 5 minutes. Template cloning typically finishes in 1 to 3 seconds on the same machine.
- No extra infrastructure: no cloud account, no custom storage layer, and no fleet of extra Docker containers.

## Installation

### Go install

```bash
go install github.com/Meru143/dbfork/cmd/dbfork@latest
```

### Homebrew

Homebrew support is planned and will be published after the first tagged release.

### Binary download

Download a release archive from the GitHub Releases page and unpack the binary for your platform.

## Quick Start

```bash
docker-compose up -d
go install github.com/Meru143/dbfork/cmd/dbfork@latest
dbfork init
dbfork create feature-add-users
dbfork list
```

Typical next steps:

```bash
dbfork use feature-add-users
dbfork diff feature-add-users
dbfork drop feature-add-users --force
```

## Command Reference

| Command | Description | Example |
|---|---|---|
| `dbfork init` | Create `~/.dbfork/config.toml` after validating a Postgres connection | `dbfork init` |
| `dbfork create <name>` | Clone the source database into a new branch database | `dbfork create feature-add-users` |
| `dbfork list` | List tracked branches, size, active marker, and orphaned state | `dbfork list` |
| `dbfork connect <name>` | Print a branch DSN, export statement, or open `psql` | `dbfork connect feature-add-users --format env` |
| `dbfork diff <name>` | Compare branch schema against its source database | `dbfork diff feature-add-users --format json` |
| `dbfork use <name>` | Write the active branch name to `.dbfork` in the current directory | `dbfork use feature-add-users` |
| `dbfork drop <name>` | Drop a branch database and remove it from local state | `dbfork drop feature-add-users --force` |

## Config Reference

`dbfork` reads configuration from `~/.dbfork/config.toml` by default:

```toml
default_host     = "localhost"
default_port     = 5432
default_user     = "postgres"
default_password = ""
default_database = "myapp_development"
```

Environment variables override file values:

- `DBFORK_HOST`
- `DBFORK_PORT`
- `DBFORK_USER`
- `DBFORK_PASSWORD`
- `DBFORK_DATABASE`
- `DBFORK_CONFIG` to point at an alternate TOML file

## How It Works

1. `dbfork create <name>` connects to the maintenance database (`postgres`).
2. It terminates idle sessions on the source database so PostgreSQL can use the source as a template.
3. It runs `CREATE DATABASE <branch> TEMPLATE <source>`.
4. It stores branch metadata in `~/.dbfork/state.json`.
5. Later commands reuse that local state to list, connect, diff, switch, and drop branches.

## Limitations

- PostgreSQL refuses template cloning while other sessions are connected to the source database. `dbfork` terminates idle sessions automatically, but active writers can still block the clone.
- Branching uses whole-database copies inside one PostgreSQL instance, so disk usage grows with each branch.
- `dbfork diff` is schema-focused in v1. It reports table, column, and index changes rather than row-level data drift.

## Development

See [CONTRIBUTING.md](./CONTRIBUTING.md) for local setup, test commands, and release workflow details.
