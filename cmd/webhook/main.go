package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cullenmcdermott/external-dns-nextdns-webhook/internal/nextdns"
	"github.com/cullenmcdermott/external-dns-nextdns-webhook/pkg/webhook"
	log "github.com/sirupsen/logrus"
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
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set log level
	if config.LogLevel != "" {
		level, err := log.ParseLevel(config.LogLevel)
		if err != nil {
			log.Warnf("Invalid log level '%s', using 'info'", config.LogLevel)
			level = log.InfoLevel
		}
		log.SetLevel(level)
	}

	log.Info("Starting NextDNS webhook provider")
	log.Infof("API port: %d", config.ServerPort)
	log.Infof("Health port: %d", config.HealthPort)
	log.Infof("Dry run: %v", config.DryRun)

	// Create NextDNS provider
	provider, err := nextdns.NewProvider(config)
	if err != nil {
		log.Fatalf("Failed to create NextDNS provider: %v", err)
	}

	// Create and start the webhook server
	srv, err := webhook.NewServer(config, provider)
	if err != nil {
		log.Fatalf("Failed to create webhook server: %v", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Start the server
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	log.Info("Server stopped")
}
