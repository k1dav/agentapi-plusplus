# justfile for agentapi-plusplus
# Multi-target: Go root + Bun/Node chat frontend. See Taskfile.yml for full task graph.

set shell := ["bash", "-uc"]

# Default: list available recipes.
default:
    @just --list

# Start dev mode — Go server on :8080 (override AGENTAPI_PORT) and chat frontend watch.
dev:
    go run ./main.go &
    cd chat && bun run dev

# Produce release artifacts (Go binary + chat frontend bundle).
build:
    go build -o ./agentapi ./main.go
    cd chat && bun run build

# Run the test suite (go test ./... + chat vitest).
test:
    go test -short ./...
    cd chat && bun run test

# Run the linter (golangci-lint).
lint:
    golangci-lint run ./...

# Apply the formatter (gofmt + chat prettier).
fmt:
    gofmt -w .
    cd chat && bun run format 2>/dev/null || true

# Remove build artifacts.
clean:
    rm -rf ./agentapi ./dist .cache chat/dist chat/node_modules/.cache coverage.out
