## [Unreleased]

## [v0.1.0] - 2026-03-21

### Added

- Core `dbfork` CLI commands: `init`, `create`, `list`, `connect`, `diff`, `use`, and `drop`
- PostgreSQL branch management built on `CREATE DATABASE ... TEMPLATE`
- Schema diff support for tables, columns, indexes, and constraints
- Unit tests, PostgreSQL integration tests, CI, and release automation
- Community docs including contributing, security, and code of conduct

### Improved

- Published repository metadata, README positioning, and discoverability assets
- Release packaging for Linux, macOS, and Windows via GoReleaser

### Fixed

- Cross-platform path handling in CLI tests
- GitHub Actions lint and integration stability
- Published module path and README install commands for the live repository
