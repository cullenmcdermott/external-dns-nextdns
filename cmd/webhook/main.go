package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cullenmcdermott/external-dns-nextdns-webhook/internal/nextdns"
	"github.com/cullenmcdermott/external-dns-nextdns-webhook/pkg/webhook"
)

const (
	banner = `
external-dns-nextdns-webhook
Version: %s
`
)

var (
	// Version is set during build
	Version = "dev"
)

func main() {
	fmt.Printf(banner, Version)

	config, err := nextdns.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set log level
	var level slog.Level
	if config.LogLevel != "" {
		if err := level.UnmarshalText([]byte(config.LogLevel)); err != nil {
			slog.Warn("Invalid log level, using 'info'", "level", config.LogLevel)
			level = slog.LevelInfo
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	slog.Info("Starting NextDNS webhook provider")
	slog.Info("Configuration", "api_port", config.ServerPort, "health_port", config.HealthPort, "dry_run", config.DryRun)

	// Create NextDNS provider
	provider, err := nextdns.NewProvider(config)
	if err != nil {
		slog.Error("Failed to create NextDNS provider", "error", err)
		os.Exit(1)
	}

	// Create and start the webhook server
	srv, err := webhook.NewServer(config, provider)
	if err != nil {
		slog.Error("Failed to create webhook server", "error", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Start the server
	if err := srv.Start(ctx); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
