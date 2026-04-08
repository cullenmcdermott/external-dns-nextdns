package nextdns

import (
	"context"
	"fmt"
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

// mockRewritesService implements nextdns.RewritesService for testing.
type mockRewritesService struct {
	rewrites []*nextdns.Rewrites
	listErr  error
}

func (m *mockRewritesService) List(_ context.Context, _ *nextdns.ListRewritesRequest) ([]*nextdns.Rewrites, error) {
	return m.rewrites, m.listErr
}

func (m *mockRewritesService) Create(_ context.Context, _ *nextdns.CreateRewritesRequest) (string, error) {
	return "", nil
}

func (m *mockRewritesService) Delete(_ context.Context, _ *nextdns.DeleteRewritesRequest) error {
	return nil
}

// newTestClient creates a Client with a mock RewritesService for unit testing.
func newTestClient(mock *mockRewritesService) *Client {
	api, _ := nextdns.New(nextdns.WithAPIKey("test-key"))
	api.Rewrites = mock
	return &Client{api: api, profileID: "test-profile"}
}

func TestFindRewriteByName(t *testing.T) {
	tests := []struct {
		name        string
		searchName  string
		searchType  string
		rewrites    []*nextdns.Rewrites
		listErr     error
		wantFound   bool
		wantContent string
		wantErr     bool
	}{
		{
			name:       "exact match",
			searchName: "test.example.com",
			searchType: "A",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
				{ID: "2", Name: "other.example.com", Type: "A", Content: "192.168.1.2"},
			},
			wantFound:   true,
			wantContent: "192.168.1.1",
		},
		{
			name:       "wrong name",
			searchName: "missing.example.com",
			searchType: "A",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
			},
			wantFound: false,
		},
		{
			name:       "wrong type",
			searchName: "test.example.com",
			searchType: "AAAA",
			rewrites: []*nextdns.Rewrites{
				{ID: "1", Name: "test.example.com", Type: "A", Content: "192.168.1.1"},
			},
			wantFound: false,
		},
		{
			name:       "empty list",
			searchName: "test.example.com",
			searchType: "A",
			rewrites:   []*nextdns.Rewrites{},
			wantFound:  false,
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
			wantContent: "192.168.1.1",
		},
		{
			name:       "list error propagates",
			searchName: "test.example.com",
			searchType: "A",
			listErr:    fmt.Errorf("API unavailable"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(&mockRewritesService{
				rewrites: tt.rewrites,
				listErr:  tt.listErr,
			})

			rewrite, found, err := client.FindRewriteByName(context.Background(), tt.searchName, tt.searchType)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FindRewriteByName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if found != tt.wantFound {
				t.Errorf("FindRewriteByName() found = %v, want %v", found, tt.wantFound)
			}
			if tt.wantFound && rewrite.Content != tt.wantContent {
				t.Errorf("FindRewriteByName() content = %v, want %v", rewrite.Content, tt.wantContent)
			}
			if !tt.wantFound && rewrite != nil {
				t.Errorf("FindRewriteByName() rewrite = %v, want nil", rewrite)
			}
		})
	}
}
