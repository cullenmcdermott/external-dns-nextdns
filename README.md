# external-dns-nextdns-webhook

An [external-dns](https://github.com/kubernetes-sigs/external-dns) webhook provider that manages DNS records through the [NextDNS](https://nextdns.io) Rewrites API.

```
External-DNS  --HTTP-->  This webhook  --API-->  NextDNS
```

## What it does

- Syncs A, AAAA, and CNAME records from Kubernetes to NextDNS
- Blocks overwrites of existing records unless you opt in per-Ingress via annotation
- Retries transient API failures (5xx, 429) with exponential backoff
- Supports dry-run mode to preview changes before applying them
- Runs as a sidecar alongside external-dns

## Prerequisites

- Kubernetes cluster
- NextDNS account with API access
- NextDNS Profile ID and API Key

## Configuration

All configuration is through environment variables.

### Required

| Variable | Description |
|----------|-------------|
| `NEXTDNS_API_KEY` | Your NextDNS API key (found at bottom of account page) |
| `NEXTDNS_PROFILE_ID` | Your NextDNS profile ID |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8888` | Webhook API port (localhost only) |
| `HEALTH_PORT` | `8080` | Health check port (exposed for k8s probes) |
| `DRY_RUN` | `false` | Preview changes without applying them |
| `LOG_LEVEL` | `info` | One of: trace, debug, info, warn, error |
| `DOMAIN_FILTER` | | Comma-separated list of domains to manage |
| `SUPPORTED_RECORDS` | `A,AAAA,CNAME` | Record types to handle |
| `DEFAULT_TTL` | `300` | Default TTL in seconds |
| `NEXTDNS_BASE_URL` | `https://api.nextdns.io` | API base URL |

## Installation

### Development (with Flox)

```bash
flox activate
export NEXTDNS_API_KEY="your-api-key"
export NEXTDNS_PROFILE_ID="your-profile-id"
just build && just run
```

### Development (manual)

```bash
export NEXTDNS_API_KEY="your-api-key"
export NEXTDNS_PROFILE_ID="your-profile-id"
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

### Kubernetes

Deploy as a sidecar with external-dns using the manifests in `deploy/kubernetes/`:

```bash
# Using kustomize
cd deploy/kubernetes
cp secret.yaml.example secret.yaml
# Edit secret.yaml with your credentials
kubectl apply -k .
```

Or apply the manifests individually:

```bash
kubectl apply -f deploy/kubernetes/namespace.yaml
cp deploy/kubernetes/secret.yaml.example deploy/kubernetes/secret.yaml
# Edit secret.yaml
kubectl apply -f deploy/kubernetes/secret.yaml
kubectl apply -f deploy/kubernetes/serviceaccount.yaml
kubectl apply -f deploy/kubernetes/clusterrole.yaml
kubectl apply -f deploy/kubernetes/clusterrolebinding.yaml
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/service.yaml
kubectl apply -f deploy/kubernetes/deployment.yaml
```

See [deploy/kubernetes/README.md](./deploy/kubernetes/README.md) for troubleshooting.

## Overwrite protection

By default, the provider won't overwrite a DNS record that already exists in NextDNS. To allow it for a specific Ingress, add this annotation:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/nextdns-allow-overwrite: "true"
```

When an overwrite is blocked, you'll see a log like:

```
WARNING: Record already exists and will NOT be overwritten.
    DNS Name: app.example.com
    Record Type: A
    Current Value: 192.168.1.100
    Planned Value: 192.168.1.200
    To allow overwrite, add annotation: external-dns.alpha.kubernetes.io/nextdns-allow-overwrite: "true"
```

## Dry-run mode

Set `DRY_RUN=true` to preview what would change without touching NextDNS. It fetches current records (read-only) and logs what it would do:

```
=== DRY RUN PREVIEW ===
INFO Would create record  action=CREATE dns_name=new.example.com record_type=A target=1.2.3.4
INFO Would update record  action=UPDATE dns_name=existing.example.com current=10.0.0.1 planned=10.0.0.2
INFO Would delete record  action=DELETE dns_name=old.example.com record_type=A target=192.168.1.99
=== END DRY RUN PREVIEW ===
```

## Retry behavior

Failed API calls are retried 3 times with backoff delays of 1s, 2s, 4s. Only transient errors are retried (network timeouts, 5xx, 429). Client errors like 401 or 404 fail immediately.

## Development

Run `just` to see all available commands. The main ones:

```bash
just build          # Build binary
just test           # Run tests
just check          # Format + vet + lint
just dev            # Run with hot-reload
just docker-build   # Build Docker image
```

This project uses [Flox](https://flox.dev) for the dev environment and [Just](https://github.com/casey/just) as a task runner.

### Project structure

```
cmd/webhook/main.go           Entry point
internal/nextdns/
  config.go                   Configuration from env vars
  client.go                   NextDNS API client with retry
  provider.go                 external-dns provider interface
pkg/webhook/server.go         HTTP servers (API + health)
deploy/kubernetes/            Kustomize manifests
```

### Dependency updates

[Renovate](https://docs.renovatebot.com/) handles Go modules, Docker base images, and GitHub Actions automatically. It runs weekly on Mondays at 6am UTC. See [renovate.json](./renovate.json).

Flox packages aren't supported by Renovate yet ([tracking issue](https://discourse.flox.dev/t/enable-renovate-to-manage-versions-in-manifest-toml/1093)), so there's a separate GitHub Actions workflow (`.github/workflows/flox-update.yml`) that runs weekly to update them.

## Resources

- [external-dns webhook provider docs](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/)
- [NextDNS API docs](https://nextdns.github.io/api/)
- [nextdns-go SDK](https://github.com/amalucelli/nextdns-go)

## License

TODO
