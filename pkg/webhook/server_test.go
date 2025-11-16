package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cullenmcdermott/external-dns-nextdns-webhook/internal/nextdns"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

// mockProvider implements the provider.Provider interface for testing
type mockProvider struct{}

func (m *mockProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	return []*endpoint.Endpoint{}, nil
}

func (m *mockProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	return nil
}

func (m *mockProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	return endpoints, nil
}

func (m *mockProvider) GetDomainFilter() endpoint.DomainFilter {
	return endpoint.DomainFilter{}
}

func TestNewServer(t *testing.T) {
	validConfig := &nextdns.Config{
		APIKey:     "test-key",
		ProfileID:  "test-profile",
		ServerPort: 8888,
		HealthPort: 8080,
	}

	validProvider := &mockProvider{}

	tests := []struct {
		name     string
		config   *nextdns.Config
		provider *mockProvider
		wantErr  bool
	}{
		{
			name:     "valid server",
			config:   validConfig,
			provider: validProvider,
			wantErr:  false,
		},
		{
			name:     "nil config",
			config:   nil,
			provider: validProvider,
			wantErr:  true,
		},
		{
			name:     "nil provider",
			config:   validConfig,
			provider: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewServer(tt.config, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewServer() returned nil server when error not expected")
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	config := &nextdns.Config{
		APIKey:     "test-key",
		ProfileID:  "test-profile",
		ServerPort: 8888,
		HealthPort: 8080,
	}

	provider := &mockProvider{}

	server, err := NewServer(config, provider)
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "OK" {
		t.Errorf("handleHealth() body = %v, want OK", string(body))
	}
}

func TestReadyEndpoint(t *testing.T) {
	config := &nextdns.Config{
		APIKey:     "test-key",
		ProfileID:  "test-profile",
		ServerPort: 8888,
		HealthPort: 8080,
	}

	provider := &mockProvider{}

	server, err := NewServer(config, provider)
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleReady() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "Ready" {
		t.Errorf("handleReady() body = %v, want Ready", string(body))
	}
}

func TestServerShutdown(t *testing.T) {
	config := &nextdns.Config{
		APIKey:     "test-key",
		ProfileID:  "test-profile",
		ServerPort: 18888, // Use different port to avoid conflicts
		HealthPort: 18080,
	}

	provider := &mockProvider{}

	server, err := NewServer(config, provider)
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)

	go func() {
		errChan <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint while server is running
	resp, err := http.Get("http://127.0.0.1:18080/healthz")
	if err != nil {
		t.Logf("Health check failed (server might not be started yet): %v", err)
	} else {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Health check status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	}

	// Cancel context to trigger shutdown
	cancel()

	// Wait for server to shutdown with timeout
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("Server shutdown returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server shutdown timed out")
	}
}

func TestServerConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		serverPort int
		healthPort int
	}{
		{
			name:       "default ports",
			serverPort: 8888,
			healthPort: 8080,
		},
		{
			name:       "custom ports",
			serverPort: 9999,
			healthPort: 9998,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &nextdns.Config{
				APIKey:     "test-key",
				ProfileID:  "test-profile",
				ServerPort: tt.serverPort,
				HealthPort: tt.healthPort,
			}

			provider := &mockProvider{}

			server, err := NewServer(config, provider)
			if err != nil {
				t.Fatalf("NewServer() failed: %v", err)
			}

			if server.config.ServerPort != tt.serverPort {
				t.Errorf("ServerPort = %v, want %v", server.config.ServerPort, tt.serverPort)
			}

			if server.config.HealthPort != tt.healthPort {
				t.Errorf("HealthPort = %v, want %v", server.config.HealthPort, tt.healthPort)
			}
		})
	}
}

func TestServerFields(t *testing.T) {
	config := &nextdns.Config{
		APIKey:     "test-key",
		ProfileID:  "test-profile",
		ServerPort: 8888,
		HealthPort: 8080,
	}

	provider := &mockProvider{}

	server, err := NewServer(config, provider)
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	if server.config == nil {
		t.Error("Server.config should not be nil")
	}

	if server.provider == nil {
		t.Error("Server.provider should not be nil")
	}

	// Initially servers should be nil (created in Start())
	if server.apiServer != nil {
		t.Error("Server.apiServer should be nil before Start()")
	}

	if server.healthServer != nil {
		t.Error("Server.healthServer should be nil before Start()")
	}
}
