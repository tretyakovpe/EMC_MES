.PHONY: help fmt lint check-secrets lint-all test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go code
	@echo "Formatting Go code..."
	@gofumpt -w .
	@goimports -w .
	@echo "✅ Code formatted!"

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run ./...
	@echo "✅ Go linting passed!"

check-secrets: ## Check for secrets in staged files
	@echo "Checking for secrets..."
	@bash scripts/check-secrets.sh

lint-all: fmt lint ## Run all linting (format + golangci-lint)
	@echo "✅ All linting passed!"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

pre-commit: lint-all check-secrets test ## Run all checks before committing (includes secrets scan)
	@echo "✅ All pre-commit checks passed!"
