.PHONY: up down db-shell db-reset sync-up secret lint test ci stage-up stage-down stage-reset stage-db-shell backup-up backup-once backup-logs deploy-prod deploy-test check-format check-workflow check-scripts check-compose check-backend check-backend-lint check-backend-errcheck check-frontend check-backup check-local check-push check-docker-build check-e2e check-release

# Current commit, baked into backup snapshot metadata.
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
PROD_ENV_FILE ?= .env.prod
TEST_ENV_FILE ?= .env.test
PROD_PROJECT ?= knowledgeos-prod
TEST_PROJECT ?= knowledgeos-test
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

up:
	GIT_COMMIT=$(GIT_COMMIT) docker compose up --build -d

down:
	docker compose down

db-shell:
	docker compose exec postgres psql -U knowledgeos -d knowledgeos

db-reset:
	docker compose down -v
	GIT_COMMIT=$(GIT_COMMIT) docker compose up --build -d

sync-up:
	docker compose --profile sync up --build -d

# Start the backup service (cron mode) alongside the stack.
backup-up:
	GIT_COMMIT=$(GIT_COMMIT) docker compose --profile backup up --build -d

# Run a single backup cycle now (debug / on-demand).
backup-once:
	docker compose --profile backup run --rm backup-service --once

backup-logs:
	docker compose --profile backup logs -f backup-service

secret:
	@openssl rand -base64 32

lint:
	cd backend && $(GOLANGCI_LINT) run ./...

test:
	cd backend && go test ./...
	cd backup-service && go test ./...
	cd frontend && npm ci && npm run lint && npm run build

ci: lint test

check-format:
	@files="$$(git ls-files 'backend/*.go' 'backend/**/*.go' 'backup-service/*.go' 'backup-service/**/*.go')"; \
	if [ -n "$$files" ]; then \
		unformatted="$$(gofmt -l $$files)"; \
		if [ -n "$$unformatted" ]; then \
			printf 'Go files need gofmt:\n%s\n' "$$unformatted"; \
			exit 1; \
		fi; \
	fi
	git diff --check

check-workflow:
	ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci-cd.yml")'

check-scripts:
	bash -n scripts/deploy-compose.sh scripts/deploy-compose-ssh.sh

check-compose:
	POSTGRES_PASSWORD=ci-postgres JWT_SECRET=ci-jwt SUPERADMIN_EMAIL=admin@example.com SUPERADMIN_PASSWORD=ci-password FRONTEND_IMAGE=ghcr.io/example/frontend:ci BACKEND_IMAGE=ghcr.io/example/backend:ci BACKUP_IMAGE=ghcr.io/example/backup-service:ci IMAGE_TAG=ci FRONTEND_PORT=8080 docker compose -f docker-compose.deploy.yml config >/dev/null

check-backend:
	cd backend && go test ./...

check-backend-lint:
	cd backend && $(GOLANGCI_LINT) run ./...

check-backend-errcheck:
	@files="$$(git diff --cached --name-only --diff-filter=ACMR -- 'backend/*.go' 'backend/**/*.go')"; \
	if [ -z "$$files" ]; then \
		echo "No staged backend Go files for errcheck"; \
		exit 0; \
	fi; \
	packages="$$(printf '%s\n' "$$files" | xargs -n1 dirname | sort -u | sed 's#^backend#.#')"; \
	echo "Running errcheck on: $$packages"; \
	cd backend && $(GOLANGCI_LINT) run --default=none -E errcheck $$packages

check-frontend:
	cd frontend && npm ci && npm run lint && npm run build

check-backup:
	cd backup-service && go test ./...

check-local: check-format check-workflow check-scripts check-backend-errcheck

check-push: check-local check-compose check-backend check-backup check-backend-lint check-frontend

check-docker-build:
	docker build -f backend/Dockerfile backend
	docker build -f frontend/Dockerfile frontend
	docker build -f backup-service/Dockerfile backup-service

check-e2e:
	docker compose up -d --build
	cd frontend && npx playwright install --with-deps chromium && CI=1 npm run test:e2e

check-release: check-push check-docker-build

stage-up:
	GIT_COMMIT=$(GIT_COMMIT) docker compose --env-file .env.staging -f docker-compose.staging.yml -p knowledgeos-staging up --build -d

stage-down:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging down

stage-reset:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging down -v
	GIT_COMMIT=$(GIT_COMMIT) docker compose --env-file .env.staging -f docker-compose.staging.yml -p knowledgeos-staging up --build -d

stage-db-shell:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging exec postgres psql -U knowledgeos_staging -d knowledgeos_staging

deploy-prod:
	docker compose --env-file $(PROD_ENV_FILE) -f docker-compose.deploy.yml -p $(PROD_PROJECT) pull
	docker compose --env-file $(PROD_ENV_FILE) -f docker-compose.deploy.yml -p $(PROD_PROJECT) up -d --remove-orphans

deploy-test:
	docker compose --env-file $(TEST_ENV_FILE) -f docker-compose.deploy.yml -p $(TEST_PROJECT) pull
	docker compose --env-file $(TEST_ENV_FILE) -f docker-compose.deploy.yml -p $(TEST_PROJECT) up -d --remove-orphans
