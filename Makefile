.PHONY: up down db-shell db-reset sync-up secret lint

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
