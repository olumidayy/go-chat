.PHONY: hooks lint test build

## Install git hooks from scripts/
hooks:
	cp scripts/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	@echo "✅ Git hooks installed."

## Run linter
lint:
	golangci-lint run

## Run tests
test:
	go test ./...

## Build
build:
	go build ./...
