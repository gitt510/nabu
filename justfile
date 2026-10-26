_default:
    @just --list --unsorted

# Run from source; extra args pass through (e.g. just run task ls)
run *args:
    go run . {{args}}

# Run all tests; extra flags pass through (e.g. just test -v)
test *args:
    go test {{args}} ./...
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent test

# Verify formatting and run static analysis
check:
    test -z "$(gofmt -l .)"
    go vet ./...
    golangci-lint run
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent check

# Build and install the binary into GOBIN
install:
    go install .

# Build the viewer and deploy it with the Worker that serves the API
deploy:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build && pnpm exec wrangler deploy

# Serve the viewer from the working tree at http://localhost:8787, rebuilt on each page load; writes reach the production D1
dev:
    pnpm install --frozen-lockfile --silent
    cd web && node dev.mjs

# Upload the working tree as a preview at https://preview-nabu.gitt510.workers.dev; production keeps serving, but writes reach the production D1
preview:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build && pnpm exec wrangler versions upload --preview-alias preview
