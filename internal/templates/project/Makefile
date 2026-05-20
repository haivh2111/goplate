.PHONY: dev build run tidy clean test coverage lint mock swag migrate gen-feature gen-adapter gen-event help

# ── Server & Build ─────────────────────────────────────────────
dev:
	@command -v air >/dev/null || { echo "air not installed: go install github.com/air-verse/air@latest"; exit 1; }
	air

build:
	@mkdir -p bin
	go build -o bin/api ./cmd

run: build
	./bin/api

tidy:
	go mod tidy

clean:
	rm -rf bin coverage.out coverage.html

# ── Testing & Quality ──────────────────────────────────────────
test:
	go test -race -count=1 ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

mock:
	@command -v mockery >/dev/null || { echo "mockery not installed"; exit 1; }
	mockery --all --output mocks

# ── Docs & Database ────────────────────────────────────────────
swag:
	@command -v swag >/dev/null || { echo "swag not installed: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init -g cmd/main.go -o swagger

migrate:
	go run ./cmd migrate

# ── Generation shortcuts (delegate to goplate CLI) ─────────────
gen-feature:
	@test -n "$(name)" || { echo "usage: make gen-feature name=<name> [fields=\"...\"]"; exit 1; }
	goplate new-feature $(name) --fields "$(fields)"

gen-adapter:
	@test -n "$(service)" || { echo "usage: make gen-adapter service=<svc> provider=<prov> methods=\"...\""; exit 1; }
	@test -n "$(provider)" || { echo "usage: make gen-adapter service=<svc> provider=<prov> methods=\"...\""; exit 1; }
	goplate new-adapter $(service) $(provider) --methods "$(methods)"

gen-event:
	@test -n "$(name)" || { echo "usage: make gen-event name=<EventName> [payload=\"...\"] [subscriber=<feature>]"; exit 1; }
	goplate new-event $(name) --payload "$(payload)" --subscriber "$(subscriber)"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
