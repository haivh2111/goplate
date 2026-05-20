# goplate

> A code generator for production-ready Go REST APIs built on Echo + GORM + hexagonal architecture.

`goplate` does two things:

1. **Bootstraps** a new Go service from an embedded boilerplate (Echo, GORM v2, JWT, structured errors, event bus).
2. **Scaffolds** features, adapters, and domain events inside an existing project — with every file pre-wired, validated, and Swagger-annotated, so you can ship business logic instead of plumbing.

Generated code follows **Folder-by-Feature + Ports & Adapters**. The generator refuses to create code that would cross dependency boundaries.

## Install

### `go install` (recommended)

```sh
go install github.com/haivh2111/goplate/cmd/goplate@latest
```

Binary lands in `$GOPATH/bin/goplate`. Add it to your `$PATH` if you haven't.

### Pre-built binaries

Each release on the [Releases page](https://github.com/haivh2111/goplate/releases) ships binaries for macOS, Linux, and Windows on both `amd64` and `arm64`. Download the archive for your OS/arch, extract, and put `goplate` somewhere on your `$PATH`.

```sh
# Example for darwin/arm64 (Apple Silicon)
curl -L https://github.com/haivh2111/goplate/releases/latest/download/goplate_v1.0.0_darwin_arm64.tar.gz | tar xz
sudo mv goplate /usr/local/bin/
goplate --version
```

### Build from source

```sh
git clone https://github.com/haivh2111/goplate.git
cd goplate
make install
```

### Verify the toolchain

```sh
goplate doctor
```

Checks for `go`, `git`, `air`, `swag`, `mockery`, `golangci-lint`. Add `--fix` to install any missing tools that ship as `go install`-able binaries.

## Quick start

```sh
# 1. Scaffold a new service
goplate new payment-service --module github.com/acme/payment-service --db postgres
cd payment-service

# 2. Edit .env (db credentials, JWT secret, ...) then start the DB
docker compose up -d db

# 3. Add a feature — model, repo, service, handler, tests, all wired up
goplate new-feature product --fields "name:string,price:float64,active:bool"

# 4. Add an external integration via the ports & adapters pattern
goplate new-adapter payment stripe \
    --methods "CreateCharge(req ChargeRequest) (*ChargeResponse, error); RefundCharge(id string) error"

# 5. Wire a domain event + subscriber
goplate new-event OrderPlaced \
    --payload "OrderID:uint,UserID:uint,TotalAmount:float64" \
    --subscriber product

# 6. Start the server with hot reload
make dev
```

Open <http://localhost:8080/api/health> — you should see `{"data":{"status":"ok"}}`. The Swagger UI lives at `/api/docs` once you've run `make swag`.

## Commands

| Command | What it does |
|---|---|
| **`goplate new`** | Scaffolds a fresh service (Echo + GORM + JWT + event bus + Makefile shims). |
| **`goplate new-feature`** | Generates 11 files for a feature: model, dto, repository (+ MySQL impl), service (+ impl), handler (with Swagger annotations), module (DI + routes), and three `*_test.go` skeletons. |
| **`goplate new-adapter`** | Generates a port interface + a concrete provider implementation. When a second provider is added, AST-merges new methods into the existing port (`--stub-siblings` auto-stubs older providers so the project keeps compiling). |
| **`goplate new-event`** | Registers a domain event constant + payload struct, optionally inserting a `RegisterSubscribers` block into a target feature. Idempotent: re-running with the same event is a no-op. |
| **`goplate doctor`** | Verifies the host toolchain. `--fix` auto-installs missing `go install`-able tools. |

Every generator command accepts `--dry-run` (preview without writing). Every command's `--help` lists the supported flags.

Inside a generated project, Makefile shims provide shorthand: `make gen-feature name=...`, `make gen-adapter service=... provider=... methods="..."`, `make gen-event name=...`.

## Generated architecture

```
your-service/
├── cmd/main.go                       # entry: load config → open DB → start Echo
├── config/                           # env-loaded Config
├── internal/
│   ├── server/                       # Echo wiring, Providers, route + subscriber registry
│   ├── middleware/                   # JWT + error handler
│   ├── shared/                       # AppError, response helpers, validator
│   ├── events/                       # event bus + event types
│   ├── infra/database/               # GORM open + migrate
│   ├── features/<name>/              # ← `goplate new-feature` writes here
│   └── adapters/<service>/           # ← `goplate new-adapter` writes here
└── Makefile                          # dev / build / test / swag / migrate / gen-*
```

**The golden rule the generators enforce:** a feature package may depend on adapter *ports* (interfaces) but never on a concrete provider package. Concrete wiring lives only in `cmd/main.go` + the `Providers` struct.

## Development

```sh
git clone https://github.com/haivh2111/goplate.git
cd goplate
make test           # full race-enabled suite, including compile-harness
make test-short     # fast unit-only pass (skips compile-harness)
make lint           # golangci-lint
make build          # local binary into ./bin/goplate
make snapshot       # build all 6 platform archives into ./dist/ for testing
```

### Releasing

```sh
git tag v1.2.3 && git push origin v1.2.3   # GitHub Actions runs goreleaser
# or for an unsigned local build:
make release VERSION=v1.2.3
```

The CI workflow ([`.github/workflows/release.yml`](.github/workflows/release.yml)) uses [goreleaser](https://goreleaser.com) to build the 6-platform matrix and upload it to a GitHub Release.

## Documentation

- **[USER_GUIDE.html](USER_GUIDE.html)** — full reference: every command, every flag, the field/methods DSL, end-to-end workflows, troubleshooting.
- **[CLAUDE.md](CLAUDE.md)** — orientation for future AI assistants working in this repo.
- **[CLI_SPEC.html](CLI_SPEC.html)** — the original design spec (historical).

## Status

Round 4 (current): all four CLI commands fully implemented, cross-platform release builds via Makefile and goreleaser, CI on every push. Round 5 (this PR): README + user guide + repo `.gitignore`.

Deferred (not blocking spec compliance): code signing for macOS/Windows binaries, Homebrew tap, deb/rpm packaging.

## License

[MIT](LICENSE) — © 2026 haivh2111.
