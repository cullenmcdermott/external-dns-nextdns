# CLAUDE.md

## Project

NextDNS webhook provider for [external-dns](https://github.com/kubernetes-sigs/external-dns). Manages DNS records via the NextDNS Rewrites API. Fully functional.

## Architecture

```
cmd/webhook/main.go          # Entry point, signal handling, config loading
internal/nextdns/
  config.go                  # Env var parsing (getEnv, getEnvInt, getEnvBool, getEnvList)
  client.go                  # NextDNS API client with retry/backoff
  provider.go                # provider.Provider implementation (Records, ApplyChanges, etc.)
pkg/webhook/server.go        # HTTP servers: API on 127.0.0.1:8888, health on 0.0.0.0:8080
deploy/kubernetes/           # Kustomize-based k8s manifests (sidecar pattern)
```

## Key Concepts

- **Webhook interface**: external-dns calls us via HTTP (`GET /`, `GET /records`, `POST /records`, `POST /adjustendpoints`)
- **NextDNS Rewrites**: A, AAAA, CNAME only. No native update — uses delete + create. SDK: `github.com/amalucelli/nextdns-go`
- **Overwrite protection**: Per-record via annotation `external-dns.alpha.kubernetes.io/nextdns-allow-overwrite: "true"`. Default: blocked.
- **Dry-run mode**: `DRY_RUN=true` previews changes without API calls
- **Retry**: Exponential backoff (3 retries) for transient/5xx/429 errors

## Development

Use `flox activate` for the dev environment. Use `flox install` for new tools (not `go install`).

```bash
just build          # Build binary
just test           # Run tests
just check          # Format + vet
just dev            # Hot-reload
just docker-build   # Docker image
```

## Code Style

- Follow existing patterns in each file
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Structured logging with `log/slog`
- Conventional commits: `feat:`, `fix:`, `test:`, `docs:`, `chore:`

## Non-Goals

- No support for record types beyond A/AAAA/CNAME
- Not a general-purpose NextDNS client
- Only manages DNS rewrites, not other NextDNS features
