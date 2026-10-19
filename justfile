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

# Render tasks/ and the viewer into web/dist/ and serve them on localhost:8000
view:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build
    go run . task ls --json > web/dist/tasks.json
    open http://localhost:8000
    python3 -m http.server -d web/dist 8000

# Build the viewer and deploy it with the Worker that serves the API and what the publish workflow uploads
deploy:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build && pnpm exec wrangler deploy
