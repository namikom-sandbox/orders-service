# AGENTS.md

## Build and test
- Build: `go build ./...`
- Unit tests: `go test -short ./...`
- Component or integration tests: `go test ./...` (needs Docker: yes; the store tests start Postgres through testcontainers)
- Lint and format: `gofmt -l .` must print nothing; `go vet ./...`
- CI workflow that runs these: `.github/workflows/ci.yml`

## Conventions
- Error handling: wrap with `fmt.Errorf("...: %w", err)`. The HTTP layer maps `orders.ErrNotFound` to 404 and `orders.ErrInvalid` to 400; anything else is a 500.
- Logging: `log/slog`, JSON to stdout. Never log request bodies or customer IDs.
- Tests live next to the code they test. Store tests skip under `-short`.

## Do not touch
- Paths the agent must not change without a person: `internal/orders/schema.sql`, `.github/workflows/`

## Owners
- Default reviewers and code-owner teams: `@namikom-sandbox/orders-team`
- Who answers product questions for this repo: `@namikom-sandbox/orders-team`

## Known traps
- `schema.sql` runs on every start, so every statement must be safe to run twice (`IF NOT EXISTS`).
