package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cullenmcdermott/external-dns-nextdns-webhook/internal/nextdns"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/provider"
	"sigs.k8s.io/external-dns/provider/webhook/api"
)

const (
	mediaTypeFormat = "application/external.dns.webhook+json;version=%d"
	defaultTimeout  = 30 * time.Second
)

// Server represents the webhook HTTP server
type Server struct {
	config   *nextdns.Config
	provider provider.Provider
	apiServer *http.Server
	healthServer *http.Server
}

// NewServer creates a new webhook server
func NewServer(config *nextdns.Config, provider provider.Provider) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}

	return &Server{
		config:   config,
		provider: provider,
	}, nil
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	// Create the webhook API handler using external-dns webhook API
	webhookServer := &api.WebhookServer{
		Provider: s.provider,
	}

	mux := http.NewServeMux()

	// Setup webhook endpoints as per external-dns specification:
	// GET / - Negotiate/Domain filter
	// GET /records - Get records
	// POST /records - Apply changes
	// POST /adjustendpoints - Adjust endpoints
	mux.HandleFunc("/", webhookServer.NegotiateHandler)
	mux.HandleFunc("/records", webhookServer.RecordsHandler)
	mux.HandleFunc("/adjustendpoints", webhookServer.AdjustEndpointsHandler)

	// Setup API server (webhook endpoints)
	s.apiServer = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", s.config.ServerPort),
		Handler:      mux,
		ReadTimeout:  defaultTimeout,
		WriteTimeout: defaultTimeout,
	}

	// Setup health server
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", s.handleHealth)
	healthMux.HandleFunc("/readyz", s.handleReady)

	s.healthServer = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", s.config.HealthPort),
		Handler:      healthMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start servers in goroutines
	apiErrChan := make(chan error, 1)
	healthErrChan := make(chan error, 1)

	go func() {
		log.Infof("Starting API server on %s", s.apiServer.Addr)
		if err := s.apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			apiErrChan <- err
		}
	}()

	go func() {
		log.Infof("Starting health server on %s", s.healthServer.Addr)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			healthErrChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Info("Shutting down servers...")
		return s.shutdown()
	case err := <-apiErrChan:
		return fmt.Errorf("API server error: %w", err)
	case err := <-healthErrChan:
		return fmt.Errorf("health server error: %w", err)
	}
}

// shutdown gracefully shuts down the servers
func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var apiErr, healthErr error

	if s.apiServer != nil {
		apiErr = s.apiServer.Shutdown(ctx)
	}

	if s.healthServer != nil {
		healthErr = s.healthServer.Shutdown(ctx)
	}

	if apiErr != nil {
		return fmt.Errorf("API server shutdown error: %w", apiErr)
	}
	if healthErr != nil {
		return fmt.Errorf("health server shutdown error: %w", healthErr)
	}

	return nil
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness checks (e.g., can connect to NextDNS API)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
