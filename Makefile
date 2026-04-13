.PHONY: up down db-shell db-reset sync-up secret lint stage-up stage-down stage-reset stage-db-shell

up:
	docker compose up --build -d

down:
	docker compose down

db-shell:
	docker compose exec postgres psql -U knowledgeos -d knowledgeos

db-reset:
	docker compose down -v
	docker compose up --build -d

sync-up:
	docker compose --profile sync up --build -d

secret:
	@openssl rand -base64 32

lint:
	cd backend && golangci-lint run ./...

stage-up:
	docker compose --env-file .env.staging -f docker-compose.staging.yml -p knowledgeos-staging up --build -d

stage-down:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging down

stage-reset:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging down -v
	docker compose --env-file .env.staging -f docker-compose.staging.yml -p knowledgeos-staging up --build -d

stage-db-shell:
	docker compose -f docker-compose.staging.yml -p knowledgeos-staging exec postgres psql -U knowledgeos_staging -d knowledgeos_staging
