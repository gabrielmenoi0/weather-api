.PHONY: run build swagger tidy docker-up docker-down test lint

## run: start API with hot-reload (requires Air)
run:
	air -c .air.toml

## build: compile the binary
build:
	go build -o bin/weather-api ./cmd/api

## swagger: generate Swagger docs
swagger:
	swag init -g cmd/api/main.go -o docs

## tidy: tidy go modules
tidy:
	go mod tidy

## docker-up: start full stack (API + Prometheus + Grafana)
docker-up:
	docker compose up --build -d

## docker-down: stop all services
docker-down:
	docker compose down

## test: run tests
test:
	go test ./... -v -race

## lint: run golangci-lint
lint:
	golangci-lint run ./...
