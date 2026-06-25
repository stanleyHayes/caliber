MODULE := github.com/xcreativs/caliber
GOBIN  := $(shell go env GOPATH)/bin

.PHONY: help mocks tools proto sqlc lint test cover build run-api run-worker tidy
help: ## list targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

mocks: ## (re)generate gomock mocks (go.uber.org/mock)
	go generate ./...

tools: ## install codegen plugins (latest stable)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

sqlc: ## generate type-safe db access from SQL (sqlc)
	sqlc generate

proto: ## resolve deps, lint, and generate from proto (needs PATH=$$PATH:$(GOBIN))
	buf dep update
	buf lint
	buf generate

lint: ## run golangci-lint (enforces hexagonal import boundaries)
	golangci-lint run ./...

test: ## run tests with race + coverage
	go test -race -coverprofile=coverage.out ./...

cover: test ## show total coverage
	go tool cover -func=coverage.out | tail -1

build: ## compile everything
	go build ./...

run-api: ## run the API + REST gateway
	go run ./cmd/api

run-worker: ## run the background worker
	go run ./cmd/worker

tidy: ## sync go.mod/go.sum
	go mod tidy
