.PHONY: build run test test-one lint fmt migrate-up migrate-down docker-up docker-down

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

test:
	go test ./...

# make test-one PKG=./internal/tenant RUN=TestFromContext
test-one:
	go test -run $(RUN) $(PKG)

lint:
	golangci-lint run ./...

fmt:
	gofmt -l -w .

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

docker-up:
	docker compose up --build

docker-down:
	docker compose down
