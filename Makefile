lint:
	@echo "Running linter checks"
	golangci-lint run

test:
	@echo "Running UNIT tests"
	@go clean -testcache
	go test -cover -race -short ./... | { grep -v 'no test files'; true; }

cover:
	@echo "Running test coverage"
	@go clean -testcache
	go test -cover -coverprofile=coverage.out -race -short ./internal/app/handler/... ./internal/app/storage/... | grep -v 'no test files'
	go tool cover -html=coverage.out

generate:
	@echo "Generating mocks"
	go generate ./...

build:
	@echo "Building the app to the .build dir"
	go build -o .build/gophermart ./cmd/gophermart/*.go
