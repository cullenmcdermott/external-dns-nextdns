// Dagger CI/CD pipeline for external-dns-nextdns-webhook
//
// This module provides CI/CD functions for building, testing, and releasing
// the external-dns NextDNS webhook provider. Functions can be run locally
// with `dagger call` or from GitHub Actions using the dagger-for-github action.

package main

import (
	"context"
	"fmt"

	"dagger/external-dns-nextdns/internal/dagger"
)

// ExternalDnsNextdns provides CI/CD pipeline functions for the webhook
type ExternalDnsNextdns struct{}

// goBase returns a Go container with the source mounted and dependencies cached
func (m *ExternalDnsNextdns) goBase(source *dagger.Directory) *dagger.Container {
	goModCache := dag.CacheVolume("go-mod-cache")
	goBuildCache := dag.CacheVolume("go-build-cache")

	return dag.Container().
		From("golang:1.25").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", goBuildCache).
		WithEnvVariable("CGO_ENABLED", "0").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"go", "mod", "download"})
}

// Test runs Go tests with race detection
func (m *ExternalDnsNextdns) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	return m.goBase(source).
		WithEnvVariable("CGO_ENABLED", "1"). // Race detection requires CGO
		WithExec([]string{"go", "test", "-v", "-race", "./..."}).
		Stdout(ctx)
}

// TestCoverage runs tests with coverage and returns the coverage file
func (m *ExternalDnsNextdns) TestCoverage(ctx context.Context, source *dagger.Directory) *dagger.File {
	return m.goBase(source).
		WithExec([]string{"go", "test", "-v", "-coverprofile=coverage.out", "./..."}).
		File("/src/coverage.out")
}

// Lint runs golangci-lint on the source code
func (m *ExternalDnsNextdns) Lint(ctx context.Context, source *dagger.Directory) error {
	goModCache := dag.CacheVolume("go-mod-cache")
	goBuildCache := dag.CacheVolume("go-build-cache")
	lintCache := dag.CacheVolume("golangci-lint-cache")

	_, err := dag.Container().
		From("golangci/golangci-lint:v2.8.0-alpine").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", goBuildCache).
		WithMountedCache("/root/.cache/golangci-lint", lintCache).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "--timeout", "5m", "./..."}).
		Sync(ctx)
	return err
}

// Build compiles the webhook binary
func (m *ExternalDnsNextdns) Build(
	ctx context.Context,
	source *dagger.Directory,
	// Version string to embed in the binary
	// +optional
	// +default="dev"
	version string,
) *dagger.File {
	ldflags := fmt.Sprintf("-s -w -X main.Version=%s", version)

	return m.goBase(source).
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "amd64").
		WithExec([]string{
			"go", "build",
			"-ldflags", ldflags,
			"-o", "webhook",
			"./cmd/webhook",
		}).
		File("/src/webhook")
}

// BuildDocker builds the Docker image for a single platform
func (m *ExternalDnsNextdns) BuildDocker(
	ctx context.Context,
	source *dagger.Directory,
	// Version string to embed in the binary
	// +optional
	// +default="dev"
	version string,
) *dagger.Container {
	return source.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Dockerfile: "Dockerfile",
		BuildArgs: []dagger.BuildArg{
			{Name: "VERSION", Value: version},
		},
	})
}

// BuildDockerMultiPlatform builds Docker images for linux/amd64 and linux/arm64
func (m *ExternalDnsNextdns) BuildDockerMultiPlatform(
	ctx context.Context,
	source *dagger.Directory,
	// Version string to embed in the binary
	version string,
) []*dagger.Container {
	platforms := []dagger.Platform{"linux/amd64", "linux/arm64"}
	containers := make([]*dagger.Container, len(platforms))

	for i, platform := range platforms {
		containers[i] = source.DockerBuild(dagger.DirectoryDockerBuildOpts{
			Dockerfile: "Dockerfile",
			Platform:   platform,
			BuildArgs: []dagger.BuildArg{
				{Name: "VERSION", Value: version},
			},
		})
	}

	return containers
}

// PublishDocker builds and publishes multi-platform Docker images to a registry
func (m *ExternalDnsNextdns) PublishDocker(
	ctx context.Context,
	source *dagger.Directory,
	// Version string for tagging
	version string,
	// Registry address (e.g., ghcr.io)
	registryAddress string,
	// Registry username
	registryUsername string,
	// Registry password as a secret
	registryPassword *dagger.Secret,
) ([]string, error) {
	containers := m.BuildDockerMultiPlatform(ctx, source, version)

	// Generate tags based on version
	imageName := fmt.Sprintf("%s/%s/external-dns-nextdns-webhook", registryAddress, registryUsername)
	tags := []string{
		fmt.Sprintf("%s:%s", imageName, version),
		fmt.Sprintf("%s:latest", imageName),
	}

	// Publish all platform variants together as a manifest list
	var publishedAddrs []string
	for _, tag := range tags {
		addr, err := dag.Container().
			WithRegistryAuth(registryAddress, registryUsername, registryPassword).
			Publish(ctx, tag, dagger.ContainerPublishOpts{
				PlatformVariants: containers,
			})
		if err != nil {
			return nil, fmt.Errorf("failed to publish %s: %w", tag, err)
		}
		publishedAddrs = append(publishedAddrs, addr)
	}

	return publishedAddrs, nil
}

// Changelog generates a changelog using git-cliff
func (m *ExternalDnsNextdns) Changelog(
	ctx context.Context,
	source *dagger.Directory,
) *dagger.File {
	return dag.Container().
		From("orhunp/git-cliff:latest").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"git-cliff", "--config", "cliff.toml", "--latest", "--strip", "header", "-o", "CHANGELOG.md"}).
		File("/src/CHANGELOG.md")
}

// CI runs the complete CI pipeline (lint, test, build, docker build)
func (m *ExternalDnsNextdns) CI(ctx context.Context, source *dagger.Directory) error {
	// Run lint
	if err := m.Lint(ctx, source); err != nil {
		return fmt.Errorf("lint failed: %w", err)
	}

	// Run tests
	if _, err := m.Test(ctx, source); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	// Build binary
	binary := m.Build(ctx, source, "dev")
	if _, err := binary.Sync(ctx); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Build Docker image
	container := m.BuildDocker(ctx, source, "dev")
	if _, err := container.Sync(ctx); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

// Release runs the complete release pipeline (test, multi-platform build, publish)
func (m *ExternalDnsNextdns) Release(
	ctx context.Context,
	source *dagger.Directory,
	// Version string (e.g., v1.0.0)
	version string,
	// Registry address (e.g., ghcr.io)
	registryAddress string,
	// Registry username
	registryUsername string,
	// Registry password as a secret
	registryPassword *dagger.Secret,
) ([]string, error) {
	// Run tests first
	if _, err := m.Test(ctx, source); err != nil {
		return nil, fmt.Errorf("tests failed: %w", err)
	}

	// Build and publish multi-platform images
	addrs, err := m.PublishDocker(ctx, source, version, registryAddress, registryUsername, registryPassword)
	if err != nil {
		return nil, fmt.Errorf("publish failed: %w", err)
	}

	return addrs, nil
}
