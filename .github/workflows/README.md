# GitHub Actions Workflows

This directory contains GitHub Actions workflows for CI/CD automation.

## Workflows Overview

### CI Workflow (`ci.yml`)

Runs on every push to `main` or `claude/**` branches and on all pull requests.

**Jobs:**
- **check**: Runs code quality checks (fmt, vet, lint) using `just check`
- **test**: Runs tests with coverage using `just test-coverage`
- **build**: Builds the binary using `just build`
- **docker**: Builds Docker image using `just docker-build`
- **security**: Runs Trivy security scanner
- **ci-success**: Summary job that ensures all checks pass

**Features:**
- ✅ Automated coverage reporting with PR comments
- ✅ Uses Go version from go.mod (1.25, matching Flox environment)
- ✅ Security scanning with Trivy
- ✅ Consistent with local development (uses Just)

### Release Workflow (`release.yml`)

Triggers on version tags (`v*.*.*`) or manual dispatch.

**Jobs:**
- **release**: Runs tests and builds multi-platform binaries
- **docker**: Builds and pushes multi-arch Docker images
- **manifests**: Generates Kubernetes manifests

**Features:**
- ✅ Multi-platform binary builds (linux/darwin, amd64/arm64)
- ✅ Automated GitHub releases with checksums
- ✅ Docker image publishing to GHCR (and optionally Docker Hub)
- ✅ Kubernetes manifest generation

## Why Just?

All workflows use [Just](https://github.com/casey/just) commands instead of raw Go commands for several important reasons:

### 1. **Single Source of Truth**
Commands are defined once in `justfile` and used everywhere:
- Local development: `just test`
- CI/CD: same `just test` command
- No drift between local and CI environments

### 2. **Consistency**
Developers and CI run exactly the same commands with the same flags:
```bash
# Local
just check

# CI (same command)
just check
```

### 3. **Maintainability**
Update a command once in `justfile`, it propagates everywhere:
```justfile
# Change test flags here, affects both local dev and CI
test:
    go test -v -race ./...
```

### 4. **Discoverability**
New contributors can see all available commands:
```bash
just --list
```

### 5. **Complexity Abstraction**
Complex commands are simplified:
```bash
# Instead of:
go test -v -race -coverprofile=coverage.out ./... && \
  go tool cover -html=coverage.out -o coverage.html

# Use:
just test-coverage
```

## What About Flox?

[Flox](https://flox.dev) provides reproducible development environments and is used for local development in this project. However, we don't use Flox in CI for these reasons:

### Local Development (Uses Flox)
- **Environment Consistency**: Same Go version, tools, and dependencies for all developers
- **Service Management**: Manages kind cluster and Tilt for local testing
- **Isolated**: Doesn't interfere with system-installed tools

### CI/CD (Uses GitHub Actions directly)
- **Simplicity**: GitHub Actions already provides consistent environments
- **Speed**: No need to bootstrap Flox in CI
- **Native Integration**: Better caching and artifact handling
- **Tool Availability**: GitHub-hosted runners have most tools we need

### Best of Both Worlds
- **Flox** ensures developer environment consistency
- **Just** ensures command consistency between local and CI
- **GitHub Actions** provides native CI/CD features

This approach gives us:
1. Reproducible local environments (Flox)
2. Consistent commands everywhere (Just)
3. Fast, native CI/CD (GitHub Actions)

## Adding Flox to CI (Optional)

If you want exact environment parity between local and CI, you can add Flox:

```yaml
- name: Install Flox
  run: |
    curl -fsSL https://downloads.flox.dev/by-env/stable/install.sh | bash
    echo "$HOME/.flox/bin" >> $GITHUB_PATH

- name: Activate Flox environment
  run: flox activate -- just test
```

However, this adds overhead and is usually unnecessary since:
- Go version is already specified in `go.mod`
- Tools are installed via GitHub Actions (Just, golangci-lint)
- Commands are standardized via Just

## Command Reference

All commands are defined in the root `justfile`:

| Command | Description | Used In |
|---------|-------------|---------|
| `just check` | Format, vet, and lint | CI |
| `just test` | Run all tests | Release |
| `just test-coverage` | Run tests with coverage | CI |
| `just build` | Build binary | CI |
| `just docker-build` | Build Docker image | CI |
| `just deps` | Download dependencies | CI |

## Triggering Workflows

### CI Workflow
- **Automatic**: Triggers on every push to `main` or `claude/**` branches
- **Automatic**: Triggers on all pull requests to `main`

### Release Workflow
- **Tag-based**: Push a tag matching `v*.*.*`:
  ```bash
  git tag v1.0.0
  git push origin v1.0.0
  ```
- **Manual**: Use GitHub UI's "Run workflow" button

## Secrets Configuration

### Required Secrets
None (uses `GITHUB_TOKEN` automatically)

### Optional Secrets
- `DOCKERHUB_USERNAME`: For publishing to Docker Hub
- `DOCKERHUB_TOKEN`: For Docker Hub authentication

Configure secrets at: `Settings > Secrets and variables > Actions`

## Coverage Reports

Coverage reports are:
1. **Generated** on every test run (Go 1.23 only)
2. **Uploaded** as workflow artifacts
3. **Commented** on pull requests automatically

Coverage thresholds:
- 🟢 **≥80%**: Great coverage!
- 🟡 **≥60%**: Coverage could be improved
- 🔴 **<60%**: Below recommended threshold

## Security Scanning

Trivy scans for:
- Vulnerabilities in Go dependencies
- Misconfigurations in code
- Security issues in Docker images

Results are uploaded to GitHub Security tab.

## Dependabot

Dependabot is configured to automatically update:
- **Go modules**: Weekly on Mondays
- **GitHub Actions**: Weekly on Mondays
- **Docker base images**: Weekly on Mondays

See `../.github/dependabot.yml` for configuration.

## Troubleshooting

### Workflow fails but passes locally
1. Check that `justfile` commands match what CI runs
2. Verify Go version matches (`go.mod` vs local)
3. Check for uncommitted changes
4. Review workflow logs for specific errors

### Coverage reporting fails
1. Ensure tests generate `coverage.out` file
2. Check that tests actually run (not skipped)
3. Verify Go 1.23 is used (coverage only on latest)

### Docker build fails
1. Check Dockerfile syntax
2. Verify build context (must be repo root)
3. Check Docker Hub credentials (if using)

### Release workflow not triggering
1. Verify tag format matches `v*.*.*` (e.g., `v1.0.0`)
2. Check that tag is pushed to remote
3. Verify GitHub Actions permissions

## Best Practices

1. **Always use Just commands** in workflows for consistency
2. **Test locally first** with the same Just commands
3. **Keep justfile updated** when changing CI commands
4. **Document new workflows** in this README
5. **Use semantic versioning** for releases (`v1.0.0`, not `1.0.0`)

## Further Reading

- [Just Documentation](https://github.com/casey/just)
- [Flox Documentation](https://flox.dev/docs)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
