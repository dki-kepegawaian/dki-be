# Import .env file
ifneq (,$(wildcard ./.env))
		include .env
		export $(shell sed 's/=.*//' .env)
endif


# Commands
dep: 
	go mod tidy

run: 
	go run cmd/main.go

build: 
	go build -o main cmd/main.go

run-build: build
	./main

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

module:
	@if [ -z "$(name)" ]; then echo "Usage: make module name=<module_name>"; exit 1; fi
	@./create_module.sh $(name)

# Commands (without docker)
migrate:
	go run cmd/main.go --migrate:run

migrate-rollback:
	go run cmd/main.go --migrate:rollback

migrate-rollback-batch:
	@if [ -z "$(batch)" ]; then echo "Usage: make migrate-rollback-batch batch=<batch_number>"; exit 1; fi
	go run cmd/main.go --migrate:rollback $(batch)

migrate-rollback-all:
	go run cmd/main.go --migrate:rollback:all

migrate-status:
	go run cmd/main.go --migrate:status

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	go run cmd/main.go --migrate:create:$(name)

seed: 
	go run cmd/main.go --seed

migrate-seed: 
	go run cmd/main.go --migrate:run --seed

# Docker (image produksi: migrasi + seeder otomatis saat container start)
init-docker:
	docker compose up -d --build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f app

container-postgres:
	docker compose exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

migrate-docker:
	docker compose exec app /app/teralur --migrate:run

migrate-status-docker:
	docker compose exec app /app/teralur --migrate:status

migrate-rollback-docker:
	docker compose exec app /app/teralur --migrate:rollback

seed-docker:
	docker compose exec app /app/teralur --seed
