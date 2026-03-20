# Contributing

## Local Development

1. Start PostgreSQL for local development:

```bash
docker-compose up -d
```

2. Download Go dependencies:

```bash
go mod download
```

3. Run the verification commands before opening a pull request:

```bash
go test ./...
go test ./test/integration/... -tags integration
golangci-lint run ./...
go build -o bin/dbfork ./cmd/dbfork/
```

## Project Layout

- `cmd/dbfork`: CLI entry point
- `internal/cli`: Cobra commands
- `internal/postgres`: PostgreSQL operations and schema diff helpers
- `internal/state`: Local branch metadata store
- `internal/config`: Config loading
- `internal/output`: Terminal renderers
- `test/integration`: Tagged testcontainers-based integration suite

## Workflow

- Keep changes focused and small.
- Add or update tests with behavior changes.
- Use conventional commits for local history.
- Do not commit secrets, `.env` files, or generated binaries.

## Releases

- Dry-run a release locally with `make release-dry`.
- Tag releases with `v*` to trigger the GitHub Actions release workflow.
