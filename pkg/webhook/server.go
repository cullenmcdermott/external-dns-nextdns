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
	config       *nextdns.Config
	provider     provider.Provider
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

	// Start health server in goroutine
	healthErrChan := make(chan error, 1)
	go func() {
		log.Infof("Starting health server on %s", s.healthServer.Addr)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			healthErrChan <- err
		}
	}()

	// Start API server using external-dns webhook API
	// This will automatically set up the required endpoints:
	// GET / - Domain filter
	// GET /records - Get records
	// POST /records - Apply changes
	// POST /adjustendpoints - Adjust endpoints
	startedChan := make(chan struct{})
	go func() {
		log.Infof("Starting webhook API server on port %d", s.config.ServerPort)
		api.StartHTTPApi(
			s.provider,
			startedChan,
			defaultTimeout,
			defaultTimeout,
			fmt.Sprintf("%d", s.config.ServerPort),
		)
	}()

	// Wait for API server to start
	<-startedChan
	log.Info("Webhook API server started successfully")

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Info("Shutting down servers...")
		return s.shutdown()
	case err := <-healthErrChan:
		return fmt.Errorf("health server error: %w", err)
	}
}

// shutdown gracefully shuts down the servers
func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.healthServer != nil {
		if err := s.healthServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("health server shutdown error: %w", err)
		}
	}

	// Note: StartHTTPApi doesn't provide a shutdown mechanism
	// The server will exit when the process terminates
	log.Info("Servers shut down successfully")
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
