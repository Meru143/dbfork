#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
media_dir="$repo_root/media"
output_path="${1:-$media_dir/dbfork-demo.gif}"
user_home="${HOME:-$(getent passwd "$(id -un)" | cut -d: -f6)}"
cache_dir="${XDG_CACHE_HOME:-$user_home/.cache}/dbfork-demo"
bin_dir="$cache_dir/bin"

ensure_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

ensure_cached_binary() {
  local name="$1"
  local url="$2"

  if command -v "$name" >/dev/null 2>&1; then
    return
  fi

  ensure_command wget
  mkdir -p "$bin_dir"

  if [[ ! -x "$bin_dir/$name" ]]; then
    echo "downloading $name into $bin_dir"
    wget -q -O "$bin_dir/$name" "$url"
    chmod +x "$bin_dir/$name"
  fi

  export PATH="$bin_dir:$PATH"
  ensure_command "$name"
}

ensure_asciinema() {
  ensure_cached_binary \
    "asciinema" \
    "https://github.com/asciinema/asciinema/releases/download/v3.2.0/asciinema-x86_64-unknown-linux-gnu"
}

ensure_agg() {
  ensure_cached_binary \
    "agg" \
    "https://github.com/asciinema/agg/releases/download/v1.7.0/agg-x86_64-unknown-linux-gnu"
}

demo_home="$(mktemp -d /tmp/dbfork-demo-home-XXXXXX)"
demo_work="$(mktemp -d /tmp/dbfork-demo-work-XXXXXX)"
demo_bin="$demo_work/dbfork"
cast_path="$demo_work/dbfork-demo.cast"
session_script="$demo_work/demo-session.sh"

cleanup() {
  if [[ -f "$repo_root/docker-compose.yml" ]]; then
    docker compose -f "$repo_root/docker-compose.yml" down -v >/dev/null 2>&1 || true
  fi
  chmod -R u+w "$demo_home" >/dev/null 2>&1 || true
  rm -rf "$demo_home" >/dev/null 2>&1 || true
  rm -rf "$demo_work" >/dev/null 2>&1 || true
}
trap cleanup EXIT

ensure_command docker
ensure_command go
mkdir -p "$media_dir"
ensure_asciinema
ensure_agg

go build -o "$demo_bin" "$repo_root/cmd/dbfork"
docker compose -f "$repo_root/docker-compose.yml" down -v >/dev/null 2>&1 || true

cat >"$session_script" <<EOF
#!/usr/bin/env bash
set -euo pipefail

repo_root="$repo_root"
demo_home="$demo_home"
demo_bin="$demo_bin"

type_command() {
  local command_text="\$1"
  local delay="\${2:-0.03}"

  printf '\$ '
  for ((i = 0; i < \${#command_text}; i++)); do
    printf '%s' "\${command_text:i:1}"
    sleep "\$delay"
  done
  printf '\n'
}

run_command() {
  local command_text="\$1"
  local pause_after="\${2:-1}"

  type_command "\$command_text"
  eval "\$command_text"
  sleep "\$pause_after"
}

section() {
  printf '\n'
  printf '# %s\n' "\$1"
  sleep 1
}

intro() {
  printf '# dbfork demo\n'
  printf '# local PostgreSQL database branching\n'
  sleep 1
}

export HOME="\$demo_home"
export GOPATH="\$demo_home/go"
export GOMODCACHE="\$demo_home/go/pkg/mod"
export PATH="$demo_work:\$PATH"
mkdir -p "\$GOMODCACHE"

cd "\$repo_root"
clear
intro

section "Start PostgreSQL"
run_command "docker compose up -d" 1
run_command "until docker exec dbfork-postgres pg_isready -U postgres -d myapp_development >/dev/null 2>&1; do sleep 1; done" 0.5

section "Initialize dbfork"
run_command "printf 'localhost\\\\n5432\\\\npostgres\\\\npostgres\\\\nmyapp_development\\\\n' | dbfork init" 1.5

section "Seed the source database"
run_command "docker exec dbfork-postgres psql -v ON_ERROR_STOP=1 -U postgres -d myapp_development -c \\\"DROP TABLE IF EXISTS users; CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);\\\"" 1.5

section "Create and inspect a branch"
run_command "dbfork create feature-add-users" 1.5
run_command "dbfork list" 1.5

section "Change the branch schema"
run_command "docker exec dbfork-postgres psql -v ON_ERROR_STOP=1 -U postgres -d dbfork_feature_add_users -c \\\"ALTER TABLE users ADD COLUMN email TEXT;\\\"" 1.5
run_command "dbfork diff feature-add-users" 2.5

section "Clean up"
run_command "dbfork drop feature-add-users --force" 2
EOF

chmod +x "$session_script"
rm -f "$output_path"

echo "recording demo cast at $cast_path"
asciinema rec \
  --quiet \
  --overwrite \
  --headless \
  --window-size 110x32 \
  --idle-time-limit 1.5 \
  --title "dbfork demo" \
  --command "$session_script" \
  "$cast_path"

echo "rendering gif to $output_path"
agg \
  --quiet \
  --theme github-dark \
  --cols 110 \
  --rows 32 \
  --font-size 18 \
  --idle-time-limit 1.5 \
  --speed 1.15 \
  "$cast_path" \
  "$output_path"

echo "demo gif written to $output_path"
