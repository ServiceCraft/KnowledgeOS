.PHONY: up down db-shell db-reset sync-up secret lint test ci stage-up stage-down stage-reset stage-db-shell backup-up backup-once backup-logs deploy-prod deploy-test

# Current commit, baked into backup snapshot metadata.
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
PROD_ENV_FILE ?= .env.prod
TEST_ENV_FILE ?= .env.test
PROD_PROJECT ?= knowledgeos-prod
TEST_PROJECT ?= knowledgeos-test

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
	cd backend && golangci-lint run ./...

test:
	cd backend && go test ./...
	cd backup-service && go test ./...
	cd frontend && npm ci && npm run lint && npm run build

ci: lint test

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
