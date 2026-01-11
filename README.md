# External-DNS NextDNS Webhook Provider

A webhook provider for [external-dns](https://github.com/kubernetes-sigs/external-dns) that manages DNS records using [NextDNS](https://nextdns.io) DNS Rewrites API.

## Status

**FULLY FUNCTIONAL** - The provider is complete and ready for production use. It includes:
- Full NextDNS API integration for DNS record management
- Per-record overwrite control via Kubernetes annotations
- Automatic retry with exponential backoff for transient API errors
- Enhanced dry-run mode with detailed change preview
- Comprehensive logging for all operations

## Features

- **Webhook Architecture**: Follows the external-dns webhook provider standard (2025)
- **DNS Rewrite Management**: Uses NextDNS DNS Rewrites API for dynamic DNS management
- **Per-Record Overwrite Protection**: Control overwrites on a per-Ingress basis using annotations
- **Automatic Retry**: Handles transient API errors with exponential backoff (3 retries: 1s, 2s, 4s delays)
- **Supported Record Types**: A, AAAA, and CNAME records
- **Domain Filtering**: Optional domain filtering for multi-tenant environments
- **Enhanced Dry Run Mode**: Preview mode shows detailed diffs of planned changes
- **Cloud Native**: Designed to run as a sidecar container in Kubernetes

## Architecture

This provider implements the external-dns webhook interface as a separate HTTP service:

```
+-----------------+         +----------------------+         +-------------+
|                 | HTTP    |  NextDNS Webhook     |  API    |             |
|  External-DNS   |-------->|  Provider            |-------->|  NextDNS    |
|                 |         |  (This Project)      |         |  API        |
+-----------------+         +----------------------+         +-------------+
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
| `DRY_RUN` | `false` | If true, preview changes without applying them |
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

### Per-Record Overwrite Protection

By default, the provider will **not overwrite existing DNS records**. This protects against accidentally overwriting important records.

To allow overwriting an existing record, add the following annotation to your Ingress or Service:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  annotations:
    external-dns.alpha.kubernetes.io/nextdns-allow-overwrite: "true"
spec:
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-service
                port:
                  number: 80
```

**Annotation Details:**
- **Key**: `external-dns.alpha.kubernetes.io/nextdns-allow-overwrite`
- **Value**: `"true"` (case-insensitive) to allow overwrite
- **Default**: If annotation is absent or set to `"false"`, overwrite is blocked

When an overwrite is blocked, a warning is logged:

```
WARNING: Record already exists and will NOT be overwritten.
    DNS Name: app.example.com
    Record Type: A
    Current Value: 192.168.1.100
    Planned Value: 192.168.1.200
    To allow overwrite, add annotation: external-dns.alpha.kubernetes.io/nextdns-allow-overwrite: "true"
```

### Automatic Retry with Exponential Backoff

The provider automatically retries failed API calls for transient errors:

- **Retry attempts**: 3 (plus initial attempt = 4 total attempts)
- **Backoff delays**: 1 second, 2 seconds, 4 seconds
- **Maximum total delay**: ~7 seconds

**Retried errors:**
- Network timeouts
- Server errors (500, 502, 503, 504)
- Rate limit errors (429)

**Not retried (fail immediately):**
- Client errors (400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found)

### Enhanced Dry-Run Mode

When `DRY_RUN=true`, the provider enters preview mode:

1. Fetches current DNS records from NextDNS (read-only API call)
2. Compares current state with planned changes
3. Logs detailed diff for each operation:

```
=== DRY RUN PREVIEW ===
INFO Would create record  action=CREATE dns_name=new.example.com record_type=A target=1.2.3.4
INFO Would update record  action=UPDATE dns_name=existing.example.com current=10.0.0.1 planned=10.0.0.2
INFO Would delete record  action=DELETE dns_name=old.example.com record_type=A target=192.168.1.99
=== END DRY RUN PREVIEW ===
```

For conflicting creates (record already exists), the preview also shows overwrite protection status:
- `overwrite="allowed (annotation present)"` - Will overwrite
- `overwrite="blocked (annotation not present)"` - Will skip

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
│       ├── client.go     # NextDNS API client with retry logic
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

### Dependency Management

This project uses automated dependency updates to keep all dependencies current and secure.

#### Renovate Bot

[Renovate](https://docs.renovatebot.com/) automatically creates pull requests for dependency updates:

- **Go modules** (go.mod): Updated automatically with semantic versioning
- **Docker base images** (Dockerfile): Updated to latest stable versions
- **GitHub Actions**: Updated when new versions are available

Renovate runs weekly on Mondays at 6am UTC and groups related updates together.

**Configuration**: See [renovate.json](./renovate.json) for the full configuration.

#### Flox Environment Updates

**Important**: Renovate does not currently support Flox manifests ([feature request](https://discourse.flox.dev/t/enable-renovate-to-manage-versions-in-manifest-toml/1093)).

For Flox package updates, we use a scheduled GitHub Actions workflow:

- **Workflow**: `.github/workflows/flox-update.yml`
- **Schedule**: Weekly on Mondays at 6am UTC (after Renovate)
- **Behavior**: Automatically creates a PR when Flox packages have updates

**Manual update**:
```bash
flox activate
flox update
```

This updates all packages in `.flox/env/manifest.toml` to their latest versions.

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
