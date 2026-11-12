_default:
    @just --list --unsorted

# Run from source; extra args pass through (e.g. just run task ls)
run *args:
    go run ./cmd/nabu {{args}}

# Run all tests; extra flags pass through (e.g. just test -v)
test *args:
    go test {{args}} ./...
    pnpm install --frozen-lockfile --silent
    pnpm --silent test

# Format the Go and TypeScript sources in place
fmt:
    gofmt -w .
    pnpm install --frozen-lockfile --silent
    pnpm --silent fmt

# Verify formatting and run static analysis
check:
    test -z "$(gofmt -l .)"
    go generate ./... && git diff --exit-code -- internal/api/openapi.gen.go
    go vet ./...
    golangci-lint run
    pnpm install --frozen-lockfile --silent
    pnpm --silent check

# Build and install the binary into GOBIN
install:
    go install ./cmd/nabu

# Build the viewer and deploy it with the Worker that serves the API
deploy:
    pnpm install --frozen-lockfile --silent
    pnpm --silent build && pnpm exec wrangler deploy

# Serve the viewer and the Worker from the working tree at http://localhost:8787 on the local D1
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    pnpm install --frozen-lockfile --silent
    pnpm exec wrangler d1 migrations apply nabu --local
    # The Worker serves dist/ as its assets; vite serves the page in dev, so an empty one is enough
    mkdir -p dist
    trap 'kill 0' EXIT
    pnpm exec wrangler dev --local --port 8788 &
    pnpm --silent dev

# Copy the production D1 into the local D1 that just dev reads (stop just dev first)
db-pull:
    #!/usr/bin/env bash
    set -euo pipefail
    # The dump holds every task, so it lives only until the copy is done
    dir=$(mktemp -d)
    trap 'rm -rf "$dir"' EXIT
    dump="$dir/nabu.sql"
    pnpm exec wrangler d1 export nabu --remote --output "$dump"
    rm -rf .wrangler/state/v3/d1
    pnpm exec wrangler d1 execute nabu --local --file "$dump"

# Upload the working tree as a preview at https://preview-nabu.gitt510.workers.dev; production keeps serving, but writes reach the production D1
preview:
    pnpm install --frozen-lockfile --silent
    pnpm --silent build && pnpm exec wrangler versions upload --preview-alias preview
