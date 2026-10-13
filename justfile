_default:
    @just --list --unsorted

# Run from source; extra args pass through (e.g. just run task ls)
run *args:
    go run . {{args}}

# Run all tests; extra flags pass through (e.g. just test -v)
test *args:
    go test {{args}} ./...

# Verify formatting and run static analysis
check:
    test -z "$(gofmt -l .)"
    go vet ./...
    golangci-lint run

# Build and install the binary into GOBIN
install:
    go install .

# Render tasks/ and the viewer into web/dist/ and serve them on localhost:8000
view:
    go run . task ls --json > web/data.json
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build && pnpm --silent data
    open http://localhost:8000
    python3 -m http.server -d web/dist 8000

# Render tasks/ with the installed nabu and upload the data to R2, where the nabu Worker serves it; skipped while another publish runs
publish:
    #!/usr/bin/env bash
    set -euo pipefail
    lock="${TMPDIR:-/tmp}/nabu-publish.lock"
    mkdir "$lock" 2>/dev/null || { echo "skip: another publish is running"; exit 0; }
    trap 'rmdir "$lock"' EXIT
    nabu task ls --json > web/data.json
    cd web
    test -d node_modules || pnpm install --frozen-lockfile --silent
    pnpm --silent data
    out=$(pnpm exec wrangler r2 object put nabu/tasks.json --file dist/tasks.json --content-type "application/json; charset=utf-8" --remote 2>&1) || { echo "$out"; exit 1; }
    echo "ok ${SECONDS}s $(( $(wc -c < dist/tasks.json) / 1024 ))KB"

# Show the tail of nabu's log, where each post_commit run lands
log lines="40":
    tail -n {{lines}} "${XDG_STATE_HOME:-$HOME/.local/state}/nabu/nabu.log"

# Build the viewer and deploy it with the Worker that serves what publish uploads
deploy:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build && pnpm exec wrangler deploy
