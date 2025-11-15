# Flox Development Environment Setup

This project uses [Flox](https://flox.dev) for reproducible, containerless development environments. This guide will help you get started.

## What is Flox?

Flox is a next-generation package and environment manager that provides:
- **Reproducible environments** across macOS and Linux (x86 and ARM)
- **No containers required** - uses pre-configured sub-shells
- **Single manifest** for packages, tools, environment variables, and services
- **Built on Nix** for reliability without requiring Nix expertise

## Prerequisites

1. Install Flox:
   ```bash
   # macOS
   brew install flox

   # Linux
   curl -fsSL https://flox.dev/install | bash
   ```

2. Ensure Docker is installed and running (required for kind):
   ```bash
   docker --version
   ```

## Getting Started

### 1. Activate the Environment

```bash
cd external-dns-nextdns
flox activate
```

On first activation, Flox will:
- Install Go 1.23 toolchain
- Install Kubernetes tools (kind, kubectl, helm)
- Install development tools (golangci-lint, delve, air)
- Install Just task runner
- Set up isolated Go cache directories
- Download Go module dependencies

You'll see a welcome message with version information and available commands.

### 2. Start Services (Optional)

The Flox environment includes a service for managing a kind (Kubernetes in Docker) cluster:

```bash
# Start the kind cluster
flox services start

# Check service status
flox services status

# View service logs
flox services logs kind

# Stop the kind cluster
flox services stop
```

The kind cluster is configured with:
- Cluster name: `external-dns-dev`
- Port forwarding for webhook (8888) and health checks (8080)
- Single control-plane node

### 3. Set Your NextDNS Credentials

```bash
export NEXTDNS_API_KEY="your-api-key-here"
export NEXTDNS_PROFILE_ID="your-profile-id-here"
```

**Tip**: Add these to a `.env.local` file (gitignored) for persistence:
```bash
# .env.local
export NEXTDNS_API_KEY="your-api-key-here"
export NEXTDNS_PROFILE_ID="your-profile-id-here"

# Then source it in your shell
source .env.local
```

### 4. Start Developing

Use Just commands for common tasks:

```bash
# Build the webhook
just build

# Run locally
just run

# Run with hot-reload (auto-rebuilds on code changes)
just dev

# Run tests
just test

# Run all checks (format, vet, lint)
just check
```

See all available commands:
```bash
just
# or
just help
```

## What's Included in the Environment?

The Flox manifest (`.flox/env/manifest.toml`) defines everything you need:

### Build Tools
- **Go 1.23**: Latest stable Go toolchain
- **Just**: Modern task runner (replacement for Make)

### Kubernetes Tools
- **kind**: Kubernetes in Docker
- **kubectl**: Kubernetes CLI
- **helm**: Kubernetes package manager

### Container Tools
- **docker**: Container runtime
- **docker-compose**: Multi-container orchestration

### Development Tools
- **golangci-lint**: Fast Go linter
- **delve**: Go debugger
- **air**: Hot-reload for Go applications
- **gotools**: Additional Go tools (godoc, goimports, etc.)

### Utilities
- **jq**: JSON processor
- **yq**: YAML processor
- **git**: Version control
- **curl**: HTTP client

## Environment Variables

The Flox environment automatically sets:

| Variable | Value | Purpose |
|----------|-------|---------|
| `PROJECT_NAME` | `external-dns-nextdns-webhook` | Project identifier |
| `VERSION` | `dev` | Build version |
| `BINARY_NAME` | `webhook` | Output binary name |
| `GO111MODULE` | `on` | Enable Go modules |
| `CGO_ENABLED` | `0` | Disable CGO for static builds |
| `LOG_LEVEL` | `info` | Default log level |
| `KIND_CLUSTER_NAME` | `external-dns-dev` | Kind cluster name |
| `GOENV` | `$FLOX_ENV_CACHE/goenv` | Isolated Go environment |
| `GOCACHE` | `$FLOX_ENV_CACHE/go-build` | Isolated Go build cache |
| `GOMODCACHE` | `$FLOX_ENV_CACHE/go/pkg/mod` | Isolated Go module cache |

## Working with Kind

The Flox environment includes a managed kind service:

### Starting the Kind Cluster

```bash
# Via Flox services (recommended)
flox services start

# Or via Just
just kind-up
```

The cluster is created with:
- Name: `external-dns-dev`
- API server: `https://127.0.0.1:6443`
- Port forwarding: 8080 (health), 8888 (webhook)

### Working with the Cluster

```bash
# Check cluster status
just kind-status

# Load Docker image into kind
just kind-load

# Deploy webhook to kind
just k8s-deploy

# View logs
just k8s-logs

# Clean up
just kind-down
# or
flox services stop
```

### Cluster Configuration

The kind cluster is defined with this configuration:
```yaml
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
```

## Typical Development Workflow

### First Time Setup
```bash
# 1. Activate Flox environment
flox activate

# 2. Set credentials
export NEXTDNS_API_KEY="..."
export NEXTDNS_PROFILE_ID="..."

# 3. Run checks to ensure everything works
just check

# 4. Start kind cluster (if testing Kubernetes integration)
flox services start
```

### Daily Development
```bash
# 1. Activate environment
flox activate

# 2. Start hot-reload development server
just dev

# In another terminal:
# Make code changes - air will auto-rebuild and restart

# 3. Run tests
just test

# 4. Before committing
just check
```

### Testing End-to-End
```bash
# 1. Build Docker image
just docker-build

# 2. Load into kind
just kind-load

# 3. Deploy
just k8s-deploy

# 4. Watch logs
just k8s-logs
```

## Customizing the Environment

### Adding Packages

Edit `.flox/env/manifest.toml`:

```toml
[install]
# Add a new package
my-tool.pkg-path = "my-tool"
my-tool.pkg-group = "dev"
```

Then reload the environment:
```bash
# Exit and re-enter
exit
flox activate
```

### Modifying Environment Variables

Edit the `[vars]` section in `.flox/env/manifest.toml`:

```toml
[vars]
MY_CUSTOM_VAR = "value"
```

### Adding Activation Hooks

Edit the `[hook]` section in `.flox/env/manifest.toml`:

```toml
[hook]
on-activate = '''
echo "Running custom setup..."
# Add your custom commands here
'''
```

## Troubleshooting

### "Docker daemon is not running"
The Flox activation hook checks if Docker is running. Start Docker:
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### "kind cluster already exists"
If the kind service fails to start because the cluster exists:
```bash
# Delete existing cluster
kind delete cluster --name external-dns-dev

# Start service again
flox services start
```

### "Go dependencies not downloading"
Ensure you have internet access and run:
```bash
just deps
# or
go mod download
```

### "Flox environment corrupted"
Reset the environment:
```bash
# Exit the environment
exit

# Remove cache
rm -rf .flox/cache .flox/run

# Re-activate
flox activate
```

### "Package not found"
Search the Flox catalog:
```bash
flox search <package-name>
```

## Flox Commands Reference

### Basic Commands
```bash
flox activate              # Enter the environment
flox deactivate           # Exit the environment (or use 'exit')
flox edit                 # Edit the manifest in $EDITOR
flox list                 # List installed packages
flox search <query>       # Search for packages
```

### Service Management
```bash
flox services start       # Start all services
flox services stop        # Stop all services
flox services restart     # Restart all services
flox services status      # Show service status
flox services logs <name> # View service logs
```

### Package Management
```bash
flox install <pkg>        # Install a package
flox uninstall <pkg>      # Remove a package
flox upgrade              # Upgrade all packages
```

### Environment Management
```bash
flox push                 # Share environment to FloxHub
flox pull                 # Get environment from FloxHub
flox clone <env>          # Clone an environment
```

## Benefits Over Traditional Setup

### Without Flox
```bash
# Install tools manually (different on each OS)
brew install go kubectl kind docker helm golangci-lint just
# or
apt-get install ... # different package names
# or
pacman -S ... # different again

# Set up environment variables manually
export GOPATH=...
export PATH=...
# Hope you remember to source .bashrc

# Install Go tools
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Different versions on different machines
# Conflicts with other projects
# Hard to onboard new developers
```

### With Flox
```bash
flox activate

# Everything is installed
# Same versions everywhere
# Isolated from other projects
# New developers: just "flox activate"
```

## Next Steps

1. **Read the Documentation**
   - [CLAUDE.md](./CLAUDE.md) - AI assistant instructions
   - [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) - Project roadmap
   - [README.md](./README.md) - Project overview

2. **Explore Just Commands**
   ```bash
   just help
   ```

3. **Start Coding**
   ```bash
   just dev
   ```

4. **Run Tests**
   ```bash
   just test
   ```

5. **Deploy to Kind**
   ```bash
   just kind-up
   just docker-build
   just kind-load
   just k8s-deploy
   ```

## Resources

- [Flox Documentation](https://flox.dev/docs)
- [Flox Blog](https://flox.dev/blog)
- [Just Documentation](https://github.com/casey/just)
- [Kind Documentation](https://kind.sigs.k8s.io)

## Questions?

- Check the [Flox Discord](https://flox.dev/discord)
- Review the [Flox GitHub](https://github.com/flox/flox)
- Read the [Flox Tutorials](https://flox.dev/docs/tutorials/)

---

**Happy coding! 🚀**
