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

# Build the viewer and deploy it with the Worker that serves the API; refuses a branch other than main or uncommitted changes
deploy:
    @test "$(git branch --show-current)" = main || { echo "deploy runs on main" >&2; exit 1; }
    @test -z "$(git status --porcelain --untracked-files=no)" || { echo "commit or stash the changes first" >&2; exit 1; }
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build
    @just _wrangler deploy

# Serve the viewer from the working tree at http://localhost:8787, rebuilt on each page load; writes reach the production D1
dev:
    pnpm install --frozen-lockfile --silent
    cd web && node dev.mjs

# Upload the working tree as a preview at https://preview-nabu.gitt510.workers.dev; production keeps serving, but writes reach the production D1
preview:
    pnpm install --frozen-lockfile --silent
    cd web && pnpm --silent build
    @just _wrangler versions upload --preview-alias preview

# Run wrangler with the Worker's vars, read from 1Password through .env.refs
_wrangler *args:
    #!/usr/bin/env bash
    set -euo pipefail
    source .env.refs
    team_domain=$(op read "${CF_ACCESS_TEAM_DOMAIN:?missing in .env.refs}")
    aud=$(op read "${CF_ACCESS_AUD:?missing in .env.refs}")
    cd web
    pnpm exec wrangler {{ args }} --var CF_ACCESS_TEAM_DOMAIN:"$team_domain" --var CF_ACCESS_AUD:"$aud"
