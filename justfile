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

# Serve the viewer and the Worker from the working tree at http://localhost:8787 on a local D1 made anew from the seed
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    pnpm install --frozen-lockfile --silent
    rm -rf .wrangler/state/v3/d1
    pnpm exec wrangler d1 migrations apply nabu --local
    pnpm exec wrangler d1 execute nabu --local --file src/api/dev/seed.sql
    # The Worker serves dist/ as its assets; vite serves the page in dev, so an empty one is enough
    mkdir -p dist
    trap 'kill 0' EXIT
    # No Access stands in front of localhost, so the Worker skips its check there
    pnpm exec wrangler dev --local --port 8788 --var DEV_SKIP_ACCESS:1 &
    pnpm --silent dev

# Upload the working tree as a preview at https://preview-nabu.gitt510.workers.dev; production keeps serving, but writes reach the production D1
preview:
    pnpm install --frozen-lockfile --silent
    pnpm --silent build && pnpm exec wrangler versions upload --preview-alias preview
