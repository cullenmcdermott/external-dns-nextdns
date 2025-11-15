# External-DNS NextDNS Webhook Provider

A webhook provider for [external-dns](https://github.com/kubernetes-sigs/external-dns) that manages DNS records using [NextDNS](https://nextdns.io) DNS Rewrites API.

## Status

✅ **NO-OP PROVIDER READY** - The service structure is complete and deployable. Currently implements a no-op provider (logs operations without making actual API calls). See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for the roadmap to full implementation.

## Features

- **Webhook Architecture**: Follows the external-dns webhook provider standard (2025)
- **DNS Rewrite Management**: Uses NextDNS DNS Rewrites API for dynamic DNS management
- **Smart Overwrite Protection**: Warns before overwriting existing records (configurable)
- **Supported Record Types**: A, AAAA, and CNAME records
- **Domain Filtering**: Optional domain filtering for multi-tenant environments
- **Dry Run Mode**: Test changes without applying them
- **Cloud Native**: Designed to run as a sidecar container in Kubernetes

## Architecture

This provider implements the external-dns webhook interface as a separate HTTP service:

```
┌─────────────────┐         ┌──────────────────────┐         ┌─────────────┐
│                 │ HTTP    │  NextDNS Webhook     │  API    │             │
│  External-DNS   │────────▶│  Provider            │────────▶│  NextDNS    │
│                 │         │  (This Project)      │         │  API        │
└─────────────────┘         └──────────────────────┘         └─────────────┘
```

## Prerequisites

- Kubernetes cluster (for deployment)
- NextDNS account with API access
- NextDNS Profile ID
- NextDNS API Key

## Configuration

The provider is configured via environment variables:

### Required

| Variable | Description |
|----------|-------------|
| `NEXTDNS_API_KEY` | Your NextDNS API key (found at bottom of account page) |
| `NEXTDNS_PROFILE_ID` | Your NextDNS profile ID |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8888` | Port for webhook API (should be localhost only) |
| `HEALTH_PORT` | `8080` | Port for health checks (exposed externally) |
| `DRY_RUN` | `false` | If true, log changes without applying them |
| `ALLOW_OVERWRITE` | `false` | If false, warn before overwriting existing records |
| `LOG_LEVEL` | `info` | Log level (trace, debug, info, warn, error) |
| `DOMAIN_FILTER` | `` | Comma-separated list of domains to manage |
| `SUPPORTED_RECORDS` | `A,AAAA,CNAME` | Comma-separated list of supported record types |
| `DEFAULT_TTL` | `300` | Default TTL for DNS records (in seconds) |
| `NEXTDNS_BASE_URL` | `https://api.nextdns.io` | NextDNS API base URL |

## Installation

### Quick Start (Development with Flox)

**Recommended**: Use Flox for a fully reproducible development environment:

```bash
# Install Flox (if not already installed)
# See: https://flox.dev/docs

# Activate the development environment
flox activate

# This automatically:
# - Installs Go 1.23, kind, kubectl, docker, just, and dev tools
# - Sets up environment variables
# - Downloads Go dependencies
# - Creates isolated Go cache

# Set your NextDNS credentials
export NEXTDNS_API_KEY="your-api-key"
export NEXTDNS_PROFILE_ID="your-profile-id"

# Build and run using Just
just build
just run

# Or run with hot-reload
just dev
```

### Quick Start (Manual Development)

```bash
# Set environment variables
export NEXTDNS_API_KEY="your-api-key"
export NEXTDNS_PROFILE_ID="your-profile-id"

# Build and run
go build -o webhook ./cmd/webhook
./webhook
```

### Docker

```bash
docker build -t external-dns-nextdns-webhook:latest .

docker run -d \
  -e NEXTDNS_API_KEY="your-api-key" \
  -e NEXTDNS_PROFILE_ID="your-profile-id" \
  -p 8080:8080 \
  external-dns-nextdns-webhook:latest
```

### Kubernetes (Sidecar Pattern)

The recommended way to deploy is using the sidecar pattern with external-dns:

```bash
# 1. Create the namespace
kubectl apply -f deploy/kubernetes/namespace.yaml

# 2. Create your secret with NextDNS credentials
cp deploy/kubernetes/secret.yaml.example deploy/kubernetes/secret.yaml
# Edit secret.yaml with your actual credentials
kubectl apply -f deploy/kubernetes/secret.yaml

# 3. Deploy RBAC and other resources
kubectl apply -f deploy/kubernetes/serviceaccount.yaml
kubectl apply -f deploy/kubernetes/clusterrole.yaml
kubectl apply -f deploy/kubernetes/clusterrolebinding.yaml
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/service.yaml

# 4. Deploy the webhook + external-dns
kubectl apply -f deploy/kubernetes/deployment.yaml

# 5. Verify it's running
kubectl get pods -n external-dns
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c nextdns-webhook
```

Or using Kustomize:

```bash
cd deploy/kubernetes
cp secret.yaml.example secret.yaml
# Edit secret.yaml with your credentials
kubectl apply -k .
```

See [deploy/kubernetes/README.md](./deploy/kubernetes/README.md) for detailed deployment instructions and troubleshooting.

## How It Works

### Record Management

1. **External-DNS** watches Kubernetes resources (Ingress, Service, etc.)
2. Sends DNS record changes to this webhook provider via HTTP
3. This provider translates changes to **NextDNS Rewrites API** calls
4. NextDNS updates DNS records in your profile

### Overwrite Protection

When `ALLOW_OVERWRITE=false` (default):

```
⚠️  WARNING: Record already exists
    DNS Name: app.example.com
    Current:  A -> 192.168.1.100
    Planned:  A -> 192.168.1.200
    Set ALLOW_OVERWRITE=true to enable automatic overwrites
```

### Supported Record Types

NextDNS Rewrites API supports:
- **A records**: IPv4 addresses
- **AAAA records**: IPv6 addresses
- **CNAME records**: Canonical name aliases

## Development

See [CLAUDE.md](./CLAUDE.md) for important notes about working on this codebase.

### Development Environment Setup

This project uses **Flox** for reproducible development environments and **Just** for task automation.

#### Using Flox (Recommended)

```bash
# Activate the Flox environment
flox activate

# Start services (e.g., kind cluster for testing)
flox services start

# Check service status
flox services status

# View logs
flox services logs kind
```

The Flox environment includes:
- Go 1.23 toolchain
- Kubernetes tools (kind, kubectl, helm)
- Container tools (docker, docker-compose)
- Development tools (golangci-lint, delve debugger, air for hot-reload)
- Task runner (just)
- Utilities (jq, yq, git, curl)

#### Available Just Commands

```bash
# Show all commands
just

# Build commands
just build              # Build the webhook binary
just build-linux        # Build for Linux AMD64
just clean              # Clean build artifacts

# Development commands
just run                # Run the webhook locally
just dev                # Run with hot-reload
just fmt                # Format code
just lint               # Run linter
just check              # Run all checks (fmt, vet, lint)

# Testing commands
just test               # Run tests
just test-coverage      # Run tests with coverage
just test-race          # Run tests with race detector

# Docker commands
just docker-build       # Build Docker image
just docker-run         # Run Docker container

# Kubernetes commands
just kind-up            # Create kind cluster
just kind-down          # Delete kind cluster
just kind-status        # Show cluster status
just k8s-deploy         # Deploy to kind
just k8s-logs           # View webhook logs

# Utility commands
just env-example        # Print example env vars
just version            # Show version info
```

### Project Structure

```
.
├── cmd/
│   └── webhook/          # Main entry point
│       └── main.go
├── internal/
│   └── nextdns/          # NextDNS provider implementation
│       ├── config.go     # Configuration management
│       └── provider.go   # Provider interface implementation
├── pkg/
│   └── webhook/          # HTTP server implementation
│       └── server.go
├── .flox/
│   └── env/
│       └── manifest.toml # Flox environment definition
├── justfile              # Task automation commands
├── IMPLEMENTATION_PLAN.md  # Detailed implementation roadmap
├── CLAUDE.md              # Instructions for AI assistants
└── README.md              # This file
```

### Building

```bash
# With Just
just build

# Or manually
go build -o webhook ./cmd/webhook
```

### Testing

```bash
# With Just
just test

# Or manually
go test ./...

# With coverage
just test-coverage
```

## Contributing

Contributions are welcome! Please see [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for current priorities.

## Resources

- [External-DNS Webhook Provider Documentation](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/)
- [NextDNS API Documentation](https://nextdns.github.io/api/)
- [NextDNS Go SDK](https://github.com/amalucelli/nextdns-go)
- [External-DNS Issue #3709](https://github.com/kubernetes-sigs/external-dns/issues/3709) - Original feature request

## License

[Add License Here]

## Acknowledgments

- Built using the [external-dns webhook provider interface](https://github.com/kubernetes-sigs/external-dns)
- Uses [amalucelli/nextdns-go](https://github.com/amalucelli/nextdns-go) SDK
- Inspired by other webhook providers in the external-dns ecosystem
