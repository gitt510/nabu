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

# Render tasks/ into web/dist/index.html from `nabu task ls --json` and open it
view:
    go run . task ls --json > web/data.json
    cd web && npm ci --silent --no-fund --no-audit && npm run --silent build
    open web/dist/index.html

# Render tasks/ with the installed nabu and upload the page to R2, where the nabu Worker serves it; skipped while another publish runs
publish:
    #!/usr/bin/env bash
    set -euo pipefail
    lock="${TMPDIR:-/tmp}/nabu-publish.lock"
    mkdir "$lock" 2>/dev/null || { echo "skip: another publish is running"; exit 0; }
    trap 'rmdir "$lock"' EXIT
    nabu task ls --json > web/data.json
    cd web
    test -d node_modules || npm ci --silent --no-fund --no-audit
    npm run --silent build
    out=$(npx wrangler r2 object put nabu/index.html --file dist/index.html --content-type "text/html; charset=utf-8" --remote 2>&1) || { echo "$out"; exit 1; }
    echo "ok ${SECONDS}s $(( $(wc -c < dist/index.html) / 1024 ))KB"

# Show the tail of nabu's log, where each post_commit run lands
log lines="40":
    tail -n {{lines}} "${XDG_STATE_HOME:-$HOME/.local/state}/nabu/nabu.log"

# Deploy the Worker that serves what publish uploads
deploy:
    cd web && npx wrangler deploy
