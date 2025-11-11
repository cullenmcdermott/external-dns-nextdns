.PHONY: help build run test clean docker-build docker-run lint fmt vet

# Variables
BINARY_NAME=webhook
DOCKER_IMAGE=external-dns-nextdns-webhook
VERSION?=dev

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the webhook binary
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BINARY_NAME) ./cmd/webhook

run: ## Run the webhook locally
	@echo "Running $(BINARY_NAME)..."
	@if [ -z "$$NEXTDNS_API_KEY" ]; then echo "Error: NEXTDNS_API_KEY not set"; exit 1; fi
	@if [ -z "$$NEXTDNS_PROFILE_ID" ]; then echo "Error: NEXTDNS_PROFILE_ID not set"; exit 1; fi
	./$(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@if [ -z "$$NEXTDNS_API_KEY" ]; then echo "Error: NEXTDNS_API_KEY not set"; exit 1; fi
	@if [ -z "$$NEXTDNS_PROFILE_ID" ]; then echo "Error: NEXTDNS_PROFILE_ID not set"; exit 1; fi
	docker run --rm \
		-e NEXTDNS_API_KEY=$$NEXTDNS_API_KEY \
		-e NEXTDNS_PROFILE_ID=$$NEXTDNS_PROFILE_ID \
		-e DRY_RUN=true \
		-p 8080:8080 \
		$(DOCKER_IMAGE):latest

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

check: fmt vet ## Run format and vet checks

dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	@echo "Development setup complete!"

# Example environment setup
example-env: ## Print example environment variables
	@echo "# NextDNS Configuration"
	@echo "export NEXTDNS_API_KEY=\"your-api-key-here\""
	@echo "export NEXTDNS_PROFILE_ID=\"your-profile-id-here\""
	@echo ""
	@echo "# Optional Configuration"
	@echo "export DRY_RUN=true"
	@echo "export ALLOW_OVERWRITE=false"
	@echo "export LOG_LEVEL=info"
	@echo "export DOMAIN_FILTER=\"example.com,example.org\""
