# Justfile for external-dns-nextdns-webhook
# Run 'just' or 'just --list' to see all available commands
# https://github.com/casey/just

# Variables (can be overridden with environment variables)
binary_name := env_var_or_default('BINARY_NAME', 'webhook')
version := env_var_or_default('VERSION', 'dev')
docker_image := env_var_or_default('DOCKER_IMAGE', 'external-dns-nextdns-webhook')
kind_cluster := env_var_or_default('KIND_CLUSTER_NAME', 'external-dns-dev')

# Default recipe - show help
default:
    @just --list

# Display help information with detailed descriptions
help:
    @echo "external-dns-nextdns-webhook - Development Commands"
    @echo ""
    @echo "Build Commands:"
    @echo "  just build              Build the webhook binary"
    @echo "  just build-linux        Build for Linux AMD64"
    @echo "  just clean              Clean build artifacts"
    @echo ""
    @echo "Development Commands:"
    @echo "  just run                Run the webhook locally"
    @echo "  just dev                Run with hot-reload (using air)"
    @echo "  just fmt                Format code"
    @echo "  just vet                Run go vet"
    @echo "  just lint               Run golangci-lint"
    @echo "  just check              Run fmt, vet, and lint"
    @echo ""
    @echo "Testing Commands:"
    @echo "  just test               Run tests"
    @echo "  just test-race          Run tests with race detector"
    @echo "  just test-coverage      Run tests and generate coverage report"
    @echo "  just test-verbose       Run tests with verbose output"
    @echo ""
    @echo "Dependency Commands:"
    @echo "  just deps               Download dependencies"
    @echo "  just tidy               Tidy dependencies"
    @echo "  just vendor             Vendor dependencies"
    @echo ""
    @echo "Docker Commands:"
    @echo "  just docker-build       Build Docker image"
    @echo "  just docker-run         Run Docker container"
    @echo "  just docker-push        Push Docker image"
    @echo ""
    @echo "Kubernetes Commands:"
    @echo "  just kind-up            Create kind cluster"
    @echo "  just kind-down          Delete kind cluster"
    @echo "  just kind-status        Show kind cluster status"
    @echo "  just kind-load          Load Docker image into kind"
    @echo "  just k8s-deploy         Deploy to kind cluster"
    @echo "  just k8s-undeploy       Remove from kind cluster"
    @echo "  just k8s-logs           Show webhook logs in kind"
    @echo ""
    @echo "Environment Commands:"
    @echo "  just env-example        Print example environment variables"
    @echo "  just version            Show version information"
    @echo ""
    @echo "Environment Variables:"
    @echo "  NEXTDNS_API_KEY         NextDNS API key (required for run)"
    @echo "  NEXTDNS_PROFILE_ID      NextDNS Profile ID (required for run)"
    @echo "  DRY_RUN                 Enable dry-run mode (default: false)"
    @echo "  ALLOW_OVERWRITE         Allow overwriting existing records (default: false)"
    @echo "  LOG_LEVEL               Log level (default: info)"

# Build the webhook binary
build:
    @echo "🔨 Building {{binary_name}}..."
    go build -ldflags="-s -w -X main.Version={{version}}" -o {{binary_name}} ./cmd/webhook
    @echo "✅ Build complete: {{binary_name}}"

# Build for Linux AMD64 (useful for Docker)
build-linux:
    @echo "🔨 Building {{binary_name}} for Linux AMD64..."
    GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version={{version}}" -o {{binary_name}}-linux-amd64 ./cmd/webhook
    @echo "✅ Build complete: {{binary_name}}-linux-amd64"

# Clean build artifacts and caches
clean:
    @echo "🧹 Cleaning build artifacts..."
    rm -f {{binary_name}}
    rm -f {{binary_name}}-linux-amd64
    rm -f coverage.out coverage.html
    @echo "✅ Clean complete"

# Run the webhook locally (requires NEXTDNS_API_KEY and NEXTDNS_PROFILE_ID)
run: build
    @echo "🚀 Running {{binary_name}}..."
    @if [ -z "${NEXTDNS_API_KEY}" ]; then echo "❌ Error: NEXTDNS_API_KEY not set"; exit 1; fi
    @if [ -z "${NEXTDNS_PROFILE_ID}" ]; then echo "❌ Error: NEXTDNS_PROFILE_ID not set"; exit 1; fi
    ./{{binary_name}}

# Run with hot-reload using air
dev:
    @echo "🔥 Starting development server with hot-reload..."
    @if [ -z "${NEXTDNS_API_KEY}" ]; then echo "❌ Error: NEXTDNS_API_KEY not set"; exit 1; fi
    @if [ -z "${NEXTDNS_PROFILE_ID}" ]; then echo "❌ Error: NEXTDNS_PROFILE_ID not set"; exit 1; fi
    air

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

# Run all checks (fmt, vet, lint)
check: fmt vet lint
    @echo "✅ All checks passed"

# Run tests
test:
    @echo "🧪 Running tests..."
    go test -v ./...

# Run tests with race detector
test-race:
    @echo "🧪 Running tests with race detector..."
    go test -v -race ./...

# Run tests with coverage
test-coverage:
    @echo "🧪 Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...
    @echo "📊 Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    @echo "✅ Coverage report generated: coverage.html"

# Run tests with verbose output
test-verbose:
    @echo "🧪 Running tests (verbose)..."
    go test -v -race -coverprofile=coverage.out ./...

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

# Vendor dependencies
vendor:
    @echo "📦 Vendoring dependencies..."
    go mod vendor
    @echo "✅ Dependencies vendored"

# Build Docker image
docker-build:
    @echo "🐳 Building Docker image..."
    docker build -t {{docker_image}}:{{version}} .
    docker tag {{docker_image}}:{{version}} {{docker_image}}:latest
    @echo "✅ Docker image built: {{docker_image}}:{{version}}"

# Run Docker container
docker-run:
    @echo "🐳 Running Docker container..."
    @if [ -z "${NEXTDNS_API_KEY}" ]; then echo "❌ Error: NEXTDNS_API_KEY not set"; exit 1; fi
    @if [ -z "${NEXTDNS_PROFILE_ID}" ]; then echo "❌ Error: NEXTDNS_PROFILE_ID not set"; exit 1; fi
    docker run --rm \
        -e NEXTDNS_API_KEY=${NEXTDNS_API_KEY} \
        -e NEXTDNS_PROFILE_ID=${NEXTDNS_PROFILE_ID} \
        -e DRY_RUN=true \
        -p 8080:8080 \
        {{docker_image}}:latest

# Push Docker image (requires Docker registry login)
docker-push:
    @echo "🐳 Pushing Docker image..."
    docker push {{docker_image}}:{{version}}
    docker push {{docker_image}}:latest
    @echo "✅ Docker image pushed"

# Create kind cluster
kind-up:
    @echo "🎡 Creating kind cluster..."
    @if kind get clusters 2>/dev/null | grep -q "^{{kind_cluster}}$$"; then \
        echo "✅ Kind cluster '{{kind_cluster}}' already exists"; \
    else \
        cat <<EOF | kind create cluster --name {{kind_cluster}} --config=- && \
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 8080
        hostPort: 8080
        protocol: TCP
      - containerPort: 8888
        hostPort: 8888
        protocol: TCP
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
EOF
        echo "⏳ Waiting for cluster to be ready..." && \
        kubectl wait --for=condition=Ready nodes --all --timeout=60s && \
        echo "✅ Kind cluster '{{kind_cluster}}' created and ready"; \
    fi

# Delete kind cluster
kind-down:
    @echo "🎡 Deleting kind cluster..."
    kind delete cluster --name {{kind_cluster}}
    @echo "✅ Kind cluster deleted"

# Show kind cluster status
kind-status:
    @echo "🎡 Kind cluster status:"
    @if kind get clusters 2>/dev/null | grep -q "^{{kind_cluster}}$$"; then \
        echo "✅ Cluster '{{kind_cluster}}' exists"; \
        kubectl cluster-info --context kind-{{kind_cluster}}; \
        echo ""; \
        echo "Nodes:"; \
        kubectl get nodes; \
    else \
        echo "❌ Cluster '{{kind_cluster}}' does not exist"; \
        echo "Run 'just kind-up' to create it"; \
    fi

# Load Docker image into kind cluster
kind-load: docker-build
    @echo "📦 Loading image into kind cluster..."
    kind load docker-image {{docker_image}}:{{version}} --name {{kind_cluster}}
    @echo "✅ Image loaded into kind cluster"

# Deploy to kind cluster (requires manifests in deploy/)
k8s-deploy:
    @echo "☸️  Deploying to kind cluster..."
    @if [ ! -d "deploy" ]; then \
        echo "❌ Error: deploy/ directory not found"; \
        echo "Create Kubernetes manifests in deploy/ first"; \
        exit 1; \
    fi
    kubectl apply -f deploy/
    @echo "✅ Deployed to kind cluster"

# Remove from kind cluster
k8s-undeploy:
    @echo "☸️  Removing from kind cluster..."
    @if [ -d "deploy" ]; then \
        kubectl delete -f deploy/ --ignore-not-found=true; \
        echo "✅ Removed from kind cluster"; \
    else \
        echo "⚠️  No deploy/ directory found, nothing to undeploy"; \
    fi

# Show webhook logs in kind
k8s-logs:
    @echo "📜 Showing webhook logs..."
    kubectl logs -l app=external-dns-nextdns-webhook -f

# Print example environment variables
env-example:
    @echo "# NextDNS Configuration"
    @echo "export NEXTDNS_API_KEY=\"your-api-key-here\""
    @echo "export NEXTDNS_PROFILE_ID=\"your-profile-id-here\""
    @echo ""
    @echo "# Optional Configuration"
    @echo "export DRY_RUN=true"
    @echo "export ALLOW_OVERWRITE=false"
    @echo "export LOG_LEVEL=info"
    @echo "export DOMAIN_FILTER=\"example.com,example.org\""
    @echo ""
    @echo "# Server Configuration"
    @echo "export WEBHOOK_HOST=localhost"
    @echo "export WEBHOOK_PORT=8888"
    @echo "export HEALTH_HOST=0.0.0.0"
    @echo "export HEALTH_PORT=8080"

# Show version information
version:
    @echo "external-dns-nextdns-webhook"
    @echo "Version: {{version}}"
    @echo ""
    @echo "Build info:"
    @go version
    @echo ""
    @echo "Dependencies:"
    @go list -m all | head -10

# Initialize air for hot-reload (creates .air.toml if it doesn't exist)
init-air:
    @if [ ! -f .air.toml ]; then \
        echo "📝 Creating .air.toml..."; \
        air init; \
        echo "✅ .air.toml created"; \
    else \
        echo "✅ .air.toml already exists"; \
    fi

# CI/CD command - runs all checks and tests
ci: check test-coverage
    @echo "✅ CI checks passed"

# Full local test - build, test, and verify
verify: clean check test-coverage build
    @echo "✅ Full verification passed"

# Development setup - download deps and run checks
setup: deps tidy check
    @echo "✅ Development environment setup complete"
