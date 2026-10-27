# AGENTS.md

Go URL-shortener REST service: create short aliases for URLs (POST /url), redirect GET /{alias} to the original URL, delete them (DELETE /url/{alias}), and read click stats (GET /url/{alias}/stats) — everything under /url is Basic-auth protected. Stack: chi, log/slog, sqlx + SQLite via `modernc.org/sqlite` (pure Go, no CGO).

## Module path

The module is named `url-shortener` (see go.mod), **not** the GitHub repo path. All internal imports use `url-shortener/internal/...`.

## Commands

```bash
go build ./...            # build
go vet ./...              # vet
golangci-lint run         # lint (config in .golangci.yml; CI uses v2.15.0)
go test ./internal/...    # unit tests
go test ./...             # also runs ./tests integration tests (need a live server)
go generate ./...         # regenerate mockery mocks
go run cmd/url-shortener/main.go   # requires CONFIG_PATH, see below
```

## Layout

- `cmd/url-shortener/main.go` — entry point: config, logger setup, chi router wiring, graceful shutdown.
- `internal/config/` — YAML config via cleanenv. Requires `CONFIG_PATH` env var pointing to a YAML file (e.g. `./config/local.yaml`); `HTTP_SERVER_PASSWORD` env var overrides the password.
- `internal/http-server/handlers/` — one package per handler. Each handler exposes `New(log, deps...) http.HandlerFunc` and declares a small consumer-side interface (`URLSaver`, `URLGetter`, `URLDeleter`, `URLStatsGetter`) next to itself.
- `internal/http-server/handlers/*/mocks/` — generated with `//go:generate go run github.com/vektra/mockery/v2@v2.53.8 --name=<Interface>`. Do not edit by hand.
- `internal/lib/api/response/` — shared JSON envelope (`status`/`error`) and `ValidationError` formatting; use `resp.OK()` / `resp.Error(...)` / `render.JSON` in handlers.
- `internal/lib/logger/sl/` — `sl.Err(err)` slog attribute helper; `slogpretty`/`slogdiscard` handlers.
- `internal/storage/` — sentinel errors `ErrURLNotFound`, `ErrURLExists`; handlers match them with `errors.Is`.
- `internal/storage/sqlite/` — `Storage` implementation; `Connect(path)` opens the DB and runs migrations.
- `tests/` — httpexpect integration tests against a live server at `localhost:8082`.

## Conventions

- Error wrapping: declare `const op = "package.Function"` at the top of a function and wrap with `fmt.Errorf("%s: %w", op, err)`.
- Logging: structured slog with `sl.Err(err)` for errors; log messages are lowercase short phrases.
- Requests/responses: decode with `render.DecodeJSON`, validate with `go-playground/validator` struct tags, respond via the shared envelope from `internal/lib/api/response`.
- Handlers respond with the shared JSON envelope; error responses set a real HTTP status code via `render.Status(r, ...)` before `render.JSON` (400 validation/decode, 404 not found, 409 alias exists, 500 internal).
- New storage behavior: add the method to the sqlite `Storage`, and declare the matching consumer-side interface in the handler package + mockery mock for tests.
- Commits use gitmoji prefixes (`:tada:`, `:fire:`, `:zap:`, …).

## Migrations

Hand-rolled versioned migrations, no external tool: `internal/storage/sqlite/migrations/` holds embedded numbered `NNNN_name.sql` files; `migrations.Up(db)` runs on every `sqlite.Connect`, tracks applied file names in the `migrations` table, and applies new ones in order, each in its own transaction. To change the schema, add a new `NNNN_description.sql` file with idempotent SQL (`IF NOT EXISTS`) — never edit an already-shipped migration file. The runner itself creates the `migrations` table; DBs created before version tracking have the schema but an empty table, so all files must stay idempotent.

## Tooling

- Lint: golangci-lint v2 (`.golangci.yml`, standard set + funlen). CI installs the pinned version from the workflow.
- CI: `.github/workflows/ci.yml` — build, vet, lint, unit tests with `-race`, a check that mocks are regenerated (`go generate ./... && git diff --exit-code`), and integration tests against a server started in the job (SQLite needs no external services).

## Gotchas

- `tests/` integration tests hit a live server at `localhost:8082` with the credentials from `config/local.yaml` (`admin:admin`) — start the server first or they fail with connection errors. CI does this automatically.
- `.env.example` is empty and `.env` files are not read by the app; configuration comes only from `CONFIG_PATH` YAML + env overrides.
