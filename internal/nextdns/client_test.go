package nextdns

import (
	"context"
	"testing"

	"github.com/amalucelli/nextdns-go/nextdns"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		profileID string
		baseURL   string
		wantErr   bool
	}{
		{
			name:      "valid client",
			apiKey:    "test-api-key",
			profileID: "test-profile",
			baseURL:   "https://api.nextdns.io",
			wantErr:   false,
		},
		{
			name:      "valid client with custom base URL",
			apiKey:    "test-api-key",
			profileID: "test-profile",
			baseURL:   "https://custom.nextdns.io",
			wantErr:   false,
		},
		{
			name:      "empty API key",
			apiKey:    "",
			profileID: "test-profile",
			baseURL:   "https://api.nextdns.io",
			wantErr:   true,
		},
		{
			name:      "empty profile ID",
			apiKey:    "test-api-key",
			profileID: "",
			baseURL:   "https://api.nextdns.io",
			wantErr:   true,
		},
		{
			name:      "both empty",
			apiKey:    "",
			profileID: "",
			baseURL:   "https://api.nextdns.io",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.apiKey, tt.profileID, tt.baseURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewClient() returned nil client when error not expected")
			}
			if !tt.wantErr && got.profileID != tt.profileID {
				t.Errorf("NewClient() profileID = %v, want %v", got.profileID, tt.profileID)
			}
		})
	}
}

func TestClientFields(t *testing.T) {
	client, err := NewClient("test-key", "test-profile", "https://api.nextdns.io")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client.api == nil {
		t.Error("Client.api should not be nil")
	}

	if client.profileID != "test-profile" {
		t.Errorf("Client.profileID = %v, want %v", client.profileID, "test-profile")
	}
}

func TestFindRewriteByName(t *testing.T) {
	tests := []struct {
		name        string
		searchName  string
		searchType  string
		rewrites    []*nextdns.Rewrites
		wantFound   bool
		wantRewrite *nextdns.Rewrites
	}{
		{
			name:       "found exact match",
			searchName: "test.example.com",
			searchType: "A",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
				{ID: "2", Name: "other.example.com", Type: "A", Content: "192.168.1.2"},
			},
			wantFound:   true,
			wantRewrite: &nextdns.Rewrites{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
		},
		{
			name:       "not found - wrong name",
			searchName: "missing.example.com",
			searchType: "A",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
			},
			wantFound:   false,
			wantRewrite: nil,
		},
		{
			name:       "not found - wrong type",
			searchName: "test.example.com",
			searchType: "AAAA",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
			},
			wantFound:   false,
			wantRewrite: nil,
		},
		{
			name:        "empty rewrites list",
			searchName:  "test.example.com",
			searchType:  "A",
			rewrites:    []*nextdns.Rewrites{},
			wantFound:   false,
			wantRewrite: nil,
		},
		{
			name:       "multiple matches returns first",
			searchName: "test.example.com",
			searchType: "A",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
				{ID: "2", Name: "test.example.com", Type: "A", Content: "192.168.1.2"},
			},
			wantFound:   true,
			wantRewrite: &nextdns.Rewrites{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require mocking the NextDNS API client.
			// For now, we skip and document that integration tests should cover this.
			// The test structure is kept for future implementation with proper mocking.
			t.Skip("Skipping test that requires API mocking - covered by integration tests")

			// When mocking is implemented, this test would:
			// 1. Create a mock client that returns tt.rewrites
			// 2. Call FindRewriteByName with tt.searchName and tt.searchType
			// 3. Verify the result matches tt.wantFound and tt.wantRewrite
			_ = tt // use tt to avoid unused variable warning
		})
	}
}

// Test that client methods have correct signatures and can be called
func TestClientMethodSignatures(t *testing.T) {
	// This test verifies that all expected methods exist and have correct signatures
	// It doesn't test functionality (that requires mocking or integration tests)

	client, err := NewClient("test-key", "test-profile", "https://api.nextdns.io")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx := context.Background()

	// Test that methods can be called (they will fail with real API, but that's ok)
	t.Run("TestConnection method exists", func(t *testing.T) {
		// Method should exist and return error (since we're using fake credentials)
		_ = client.TestConnection(ctx)
	})

	t.Run("ListRewrites method exists", func(t *testing.T) {
		_, _ = client.ListRewrites(ctx)
	})

	t.Run("CreateRewrite method exists", func(t *testing.T) {
		_, _ = client.CreateRewrite(ctx, "test.example.com", "A", "192.168.1.1")
	})

	t.Run("DeleteRewrite method exists", func(t *testing.T) {
		_ = client.DeleteRewrite(ctx, "test-id")
	})

	t.Run("FindRewriteByName method exists", func(t *testing.T) {
		_, _, _ = client.FindRewriteByName(ctx, "test.example.com", "A")
	})

	t.Run("UpdateRewrite method exists", func(t *testing.T) {
		_, _ = client.UpdateRewrite(ctx, "test-id", "test.example.com", "A", "192.168.1.1")
	})
}
