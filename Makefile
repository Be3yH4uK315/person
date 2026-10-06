APP_NAME := person-api
GO := go
GOFMT := gofmt
GOLANGCI_LINT_VERSION := 2.14.0
GOVULNCHECK_VERSION := v1.8.0
GOFILES := $(shell find . -type f -name '*.go' -not -path './vendor/*')

-include .env
export

.PHONY: help fmt fmt-check mod-tidy mod-verify vet test test-race test-integration govulncheck build run lint lint-fix fix check docker-build compose-config compose-up compose-down migrate-up test-db-create migrate-test-up migrate-down migrate-version clean

help:
	@printf '%s\n' \
		'make fmt             - format all Go source files in place' \
		'make fmt-check       - fail if any Go source file is not gofmt-ed' \
		'make mod-tidy        - normalize go.mod/go.sum' \
		'make mod-verify      - verify downloaded module content' \
		'make vet             - run go vet' \
		'make test            - run unit tests' \
		'make test-race       - run tests with race detector' \
		'make test-integration - run PostgreSQL integration tests (requires TEST_DATABASE_URL and migrations)' \
		'make lint            - run pinned golangci-lint version' \
		'make lint-fix        - apply golangci-lint supported fixes' \
		'make build           - build API binary' \
		'make fix             - apply automatic fixes (fmt + mod tidy + lint fixes)' \
		'make govulncheck     - scan Go dependencies and reachable code for known vulnerabilities' \
		'make check           - run the full local quality gate without modifying the tree' \
		'make run             - run API locally using variables from .env' \
		'make docker-build    - build API image' \
		'make compose-config  - validate rendered Docker Compose configuration' \
		'make compose-up      - start PostgreSQL, migrations and API' \
		'make compose-down    - stop compose stack' \
		'make migrate-up      - apply database migrations' \
		'make test-db-create  - create the dedicated person_test database in the compose PostgreSQL' \
		'make migrate-test-up - create person_test (if needed) and apply migrations to it' \
		'make migrate-down    - rollback one migration' \
		'make migrate-version - show current migration version' \
		'make clean           - remove local build artifacts'

fmt:
	@$(GOFMT) -w $(GOFILES)

fmt-check:
	@unformatted="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

mod-tidy:
	$(GO) mod tidy

mod-verify:
	$(GO) mod verify

vet:
	$(GO) vet ./...
	$(GO) vet -tags=integration ./...

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

test-integration:
	@test -n "$(TEST_DATABASE_URL)" || { echo 'TEST_DATABASE_URL is required'; exit 1; }
	$(GO) test -tags=integration -count=1 ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo 'golangci-lint is not installed'; exit 1; }
	golangci-lint version | grep -Eq "(^|[[:space:]])$(GOLANGCI_LINT_VERSION)([[:space:]]|$$)" || { echo 'golangci-lint $(GOLANGCI_LINT_VERSION) is required'; exit 1; }
	golangci-lint run ./...

lint-fix:
	@command -v golangci-lint >/dev/null 2>&1 || { echo 'golangci-lint is not installed'; exit 1; }
	golangci-lint version | grep -Eq "(^|[[:space:]])$(GOLANGCI_LINT_VERSION)([[:space:]]|$$)" || { echo 'golangci-lint $(GOLANGCI_LINT_VERSION) is required'; exit 1; }
	golangci-lint run --fix ./...

govulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

build:
	@mkdir -p bin
	$(GO) build -trimpath -o bin/$(APP_NAME) ./cmd/person-api

run:
	@test -n "$(DATABASE_URL)" || { echo 'DATABASE_URL is required; copy .env.example to .env and configure it'; exit 1; }
	$(GO) run ./cmd/person-api

fix: fmt mod-tidy lint-fix

check: fmt-check mod-verify vet test test-race lint govulncheck build

docker-build:
	docker build -t $(APP_NAME):local .

compose-config:
	docker compose config --quiet

compose-up: compose-config
	docker compose up -d --build

compose-down:
	docker compose down

migrate-up:
	@test -n "$(MIGRATE_DATABASE_URL)" || { echo 'MIGRATE_DATABASE_URL is required'; exit 1; }
	@docker compose run --rm migrate -path=/migrations -database="$(MIGRATE_DATABASE_URL)" up

test-db-create:
	@docker compose up -d --wait postgres
	@docker compose exec -T postgres sh -c 'createdb -U "$$POSTGRES_USER" person_test' 2>&1 | grep -v 'already exists' || true

migrate-test-up: test-db-create
	@test -n "$(MIGRATE_TEST_DATABASE_URL)" || { echo 'MIGRATE_TEST_DATABASE_URL is required'; exit 1; }
	@docker compose run --rm migrate -path=/migrations -database="$(MIGRATE_TEST_DATABASE_URL)" up

migrate-down:
	@test -n "$(MIGRATE_DATABASE_URL)" || { echo 'MIGRATE_DATABASE_URL is required'; exit 1; }
	@docker compose run --rm migrate -path=/migrations -database="$(MIGRATE_DATABASE_URL)" down 1

migrate-version:
	@test -n "$(MIGRATE_DATABASE_URL)" || { echo 'MIGRATE_DATABASE_URL is required'; exit 1; }
	@docker compose run --rm migrate -path=/migrations -database="$(MIGRATE_DATABASE_URL)" version

clean:
	rm -rf bin
