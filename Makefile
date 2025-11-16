.PHONY: build up down test test-unit test-integration fmt lint clean migrate logs test-db

build:
	go build -o pr-manager ./cmd/pr-manager

up:
	docker-compose up --build

down:
	docker-compose down

test: test-unit test-integration

test-unit:
	go test ./internal/... -v

test-integration: test-db
	DB_CONN=postgres://postgres:postgres@localhost:5432/pr_manager_test?sslmode=disable go test ./tests/... -v -tags=integration

test-db:
	docker-compose exec db psql -U postgres -c "CREATE DATABASE pr_manager_test;" || true

fmt:
	gofmt -s -w .

lint:
	go vet ./...

clean:
	rm -f pr-manager
	docker-compose down -v

migrate:
	docker-compose exec db psql -U postgres -d pr_manager -c "SELECT * FROM pg_tables WHERE schemaname = 'public';"

logs:
	docker-compose logs -f app