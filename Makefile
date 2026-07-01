MODULE := github.com/xcreativs/caliber
GOBIN  := $(shell go env GOPATH)/bin

.PHONY: help mocks tools proto sqlc lint vet test test-short cover build ci scan scan-go scan-web scan-containers run-api run-worker run-of-show run-of-show-keep-alive backup-capture tidy offline-build offline-pull offline-demo offline-stop offline-check
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

vet: ## run go vet
	go vet ./...

COVERAGE_EXCLUDES := 'node_modules|internal/gen/|internal/mocks/|internal/platform/migrate/|internal/adapters/outbound/postgres/sqlcdb/'

test: ## run tests with race + coverage (Docker-gated integration tests skip fast if Docker is down)
	go test -race -coverprofile=coverage.out ./...

test-short: ## run tests fast, skipping the testcontainers integration tests
	go test -short ./...

cover: test ## show app-code coverage (excludes generated/vendor packages)
	grep -vE $(COVERAGE_EXCLUDES) coverage.out > coverage.app.out
	go tool cover -func=coverage.app.out | tail -1

cover-check: test ## enforce per-package Go coverage >= 80% (excludes integration/cmd/generated)
	scripts/check-go-coverage.sh 80

cover-report: test ## write a JSON coverage trend report (coverage-report.json)
	grep -vE $(COVERAGE_EXCLUDES) coverage.out > coverage.app.out
	@echo "{\"total_app_coverage\":\"$$(go tool cover -func=coverage.app.out | awk '/^total:/ {print $$3}')\",\"generated_at\":\"$$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > coverage-report.json
	@echo "Coverage report written to coverage-report.json"

build: ## compile everything
	go build ./...

ci: build vet lint test ## run the full local CI (build, vet, lint, race tests) — run this before pushing
	@echo "local CI passed — safe to push"

scan: scan-go scan-web scan-containers ## run dependency and container vulnerability scans
	@echo "supply-chain scans passed"

scan-go: ## run govulncheck over Go packages
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

scan-web: ## run npm audit for high/critical frontend vulnerabilities
	cd web && npm audit --audit-level=high

scan-containers: ## build and scan api/worker/migrate images with Trivy
	@command -v trivy >/dev/null || { echo "trivy is required for container scans: https://trivy.dev/latest/getting-started/installation/"; exit 127; }
	docker build -f deploy/Dockerfile.api -t caliber-api:scan .
	trivy image --vuln-type os,library --severity HIGH,CRITICAL --exit-code 1 caliber-api:scan
	docker build -f deploy/Dockerfile.worker -t caliber-worker:scan .
	trivy image --vuln-type os,library --severity HIGH,CRITICAL --exit-code 1 caliber-worker:scan
	docker build -f deploy/Dockerfile.migrate -t caliber-migrate:scan .
	trivy image --vuln-type os,library --severity HIGH,CRITICAL --exit-code 1 caliber-migrate:scan

run-api: ## run the API + REST gateway
	go run ./cmd/api

run-worker: ## run the background worker
	go run ./cmd/worker

run-of-show: ## drive the full demo narrative end-to-end (CAL-105)
	scripts/run-of-show.sh

run-of-show-keep-alive: ## drive the demo and keep the API running for UI exploration
	scripts/run-of-show.sh --keep-alive

backup-capture: ## record a clean Flow B transcript + report card to web/public/interview-backup.json (CAL-106)
	go run ./cmd/backup-capture -out web/public/interview-backup.json

tidy: ## sync go.mod/go.sum
	go mod tidy

offline-pull: ## pull base images needed by the offline stack (requires network once)
	docker pull pgvector/pgvector:pg17
	docker pull redis:7-alpine
	docker pull node:24-alpine
	docker pull nginx:1.27-alpine
	docker pull golang:1.26.4
	docker pull gcr.io/distroless/static-debian12:nonroot

offline-build: ## build all images for the self-contained offline demo stack
	docker compose -f docker-compose.offline.yml build

offline-demo: ## start the self-contained offline demo stack (no external network needed after build)
	scripts/offline-demo.sh

offline-stop: ## stop the offline demo stack
	scripts/offline-demo.sh --stop

offline-check: ## verify images and compose config for the offline stack
	scripts/offline-demo.sh --check
