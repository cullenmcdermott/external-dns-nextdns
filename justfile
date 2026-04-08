# Justfile for external-dns-nextdns-webhook
# Run 'just' or 'just --list' to see all available commands
# https://github.com/casey/just

# Variables (can be overridden with environment variables)
binary_name := env_var_or_default('BINARY_NAME', 'webhook')
version := env_var_or_default('VERSION', 'dev')
docker_image := env_var_or_default('DOCKER_IMAGE', 'external-dns-nextdns-webhook')

# Default recipe - show help
default:
    @just --list

# Display help information with detailed descriptions
help:
    @echo "external-dns-nextdns-webhook - Development Commands"
    @echo ""
    @echo "Dev Environment (Flox + Tilt):"
    @echo "  just dev-start          Start development environment (Flox services)"
    @echo "  just dev-stop           Stop development environment and clean up"
    @echo "  just dev-restart        Restart development environment"
    @echo "  just dev-logs           Tail Flox service logs"
    @echo "  just dev-status         Show Flox service status"
    @echo ""
    @echo "Build Commands:"
    @echo "  just build              Build the webhook binary"
    @echo "  just docker-build       Build Docker image"
    @echo "  just clean              Clean build artifacts"
    @echo ""
    @echo "Testing Commands:"
    @echo "  just test               Run all tests"
    @echo "  just test-unit          Run unit tests only (fast)"
    @echo "  just test-integration   Run integration tests only"
    @echo "  just test-coverage      Run tests and generate coverage report"
    @echo ""
    @echo "Code Quality:"
    @echo "  just check              Format, vet, and lint code"
    @echo "  just fmt                Format code"
    @echo "  just lint               Run golangci-lint"
    @echo ""
    @echo "Dependencies:"
    @echo "  just deps               Download dependencies"
    @echo "  just tidy               Tidy dependencies"
    @echo ""

# ==========================================
# Development Environment (Flox)
# ==========================================

# Start development environment (kind cluster + Tilt)
dev-start:
    @echo "🚀 Starting development environment..."
    @echo "This will start the kind cluster and Tilt..."
    flox services start dev

# Stop development environment and clean up
dev-stop:
    @echo "🛑 Stopping development environment..."
    flox services stop dev

# Restart development environment
dev-restart: dev-stop dev-start
    @echo "♻️  Development environment restarted"

# Tail Flox service logs
dev-logs:
    @echo "📜 Tailing development environment logs..."
    @echo "(Ctrl+C to stop)"
    flox services logs dev --follow

# Show Flox service status
dev-status:
    @echo "📊 Flox service status:"
    flox services status

# ==========================================
# Build Commands
# ==========================================

# Build the webhook binary
build:
    @echo "🔨 Building {{binary_name}}..."
    go build -ldflags="-s -w -X main.Version={{version}}" -o {{binary_name}} ./cmd/webhook
    @echo "✅ Build complete: {{binary_name}}"

# Build container image with ko
image-build:
    @echo "🐳 Building container image with ko..."
    ko build ./cmd/webhook --local
    @echo "✅ Container image built"

# Clean build artifacts and caches
clean:
    @echo "🧹 Cleaning build artifacts..."
    rm -f {{binary_name}}
    rm -f {{binary_name}}-linux-amd64
    rm -f coverage.out coverage.html
    find . -name "*.go.*" -type f -delete 2>/dev/null || true
    @echo "✅ Clean complete"

# ==========================================
# Testing Commands
# ==========================================

# Run all tests
test:
    @echo "🧪 Running all tests..."
    go test -v -race ./...

# Run unit tests only (fast, no external dependencies)
test-unit:
    @echo "🧪 Running unit tests..."
    go test -v -race -short ./...

# Run integration tests only (may require test NextDNS profile)
test-integration:
    @echo "🧪 Running integration tests..."
    go test -v -race -run Integration ./...

# Run tests with coverage report
test-coverage:
    @echo "🧪 Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...
    @echo "📊 Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    @echo "✅ Coverage report generated: coverage.html"

# ==========================================
# Code Quality
# ==========================================

# Run all checks (fmt, vet, lint)
check: fmt vet lint
    @echo "✅ All checks passed"

# Format code
fmt:
    @echo "📝 Formatting code..."
    go fmt ./...
    @echo "✅ Format complete"

# Run go vet
vet:
    @echo "🔍 Running go vet..."
    go vet ./...
    @echo "✅ Vet complete"

# Run golangci-lint
lint:
    @echo "🔍 Running golangci-lint..."
    golangci-lint run
    @echo "✅ Lint complete"

# ==========================================
# Dependencies
# ==========================================

# Download dependencies
deps:
    @echo "📦 Downloading dependencies..."
    go mod download
    @echo "✅ Dependencies downloaded"

# Tidy dependencies
tidy:
    @echo "📦 Tidying dependencies..."
    go mod tidy
    @echo "✅ Dependencies tidied"

