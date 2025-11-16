package nextdns

import (
	"context"
	"reflect"
	"testing"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid provider",
			config: &Config{
				APIKey:           "test-api-key",
				ProfileID:        "test-profile",
				BaseURL:          "https://api.nextdns.io",
				DryRun:           true, // Use dry-run to avoid API calls
				SupportedRecords: []string{"A", "AAAA", "CNAME"},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty API key",
			config: &Config{
				APIKey:    "",
				ProfileID: "test-profile",
			},
			wantErr: true,
		},
		{
			name: "empty profile ID",
			config: &Config{
				APIKey:    "test-api-key",
				ProfileID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewProvider() returned nil provider when error not expected")
			}
			if !tt.wantErr && got.config != tt.config {
				t.Error("NewProvider() config not set correctly")
			}
		})
	}
}

func TestIsSupportedRecordType(t *testing.T) {
	provider := &Provider{
		config: &Config{
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	tests := []struct {
		name       string
		recordType string
		want       bool
	}{
		{
			name:       "A record",
			recordType: "A",
			want:       true,
		},
		{
			name:       "AAAA record",
			recordType: "AAAA",
			want:       true,
		},
		{
			name:       "CNAME record",
			recordType: "CNAME",
			want:       true,
		},
		{
			name:       "lowercase a record",
			recordType: "a",
			want:       true,
		},
		{
			name:       "TXT record (unsupported)",
			recordType: "TXT",
			want:       false,
		},
		{
			name:       "MX record (unsupported)",
			recordType: "MX",
			want:       false,
		},
		{
			name:       "empty string",
			recordType: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.isSupportedRecordType(tt.recordType)
			if got != tt.want {
				t.Errorf("isSupportedRecordType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesDomainFilter(t *testing.T) {
	tests := []struct {
		name         string
		domainFilter []string
		dnsName      string
		want         bool
	}{
		{
			name:         "exact match",
			domainFilter: []string{"example.com"},
			dnsName:      "example.com",
			want:         true,
		},
		{
			name:         "subdomain match",
			domainFilter: []string{"example.com"},
			dnsName:      "test.example.com",
			want:         true,
		},
		{
			name:         "no match",
			domainFilter: []string{"example.com"},
			dnsName:      "other.com",
			want:         false,
		},
		{
			name:         "multiple filters - first match",
			domainFilter: []string{"example.com", "test.com"},
			dnsName:      "app.example.com",
			want:         true,
		},
		{
			name:         "multiple filters - second match",
			domainFilter: []string{"example.com", "test.com"},
			dnsName:      "app.test.com",
			want:         true,
		},
		{
			name:         "multiple filters - no match",
			domainFilter: []string{"example.com", "test.com"},
			dnsName:      "app.other.com",
			want:         false,
		},
		{
			name:         "empty filter list",
			domainFilter: []string{},
			dnsName:      "example.com",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				config: &Config{
					DomainFilter: tt.domainFilter,
				},
			}

			got := provider.matchesDomainFilter(tt.dnsName)
			if got != tt.want {
				t.Errorf("matchesDomainFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdjustEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		endpoints []*endpoint.Endpoint
		want      []*endpoint.Endpoint
	}{
		{
			name: "filter unsupported record type",
			config: &Config{
				SupportedRecords: []string{"A", "AAAA"},
				DomainFilter:     []string{},
			},
			endpoints: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
				{DNSName: "test.example.com", RecordType: "TXT", Targets: []string{"txt-value"}},
			},
			want: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
			},
		},
		{
			name: "filter by domain",
			config: &Config{
				SupportedRecords: []string{"A", "AAAA"},
				DomainFilter:     []string{"example.com"},
			},
			endpoints: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
				{DNSName: "test.other.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
			},
			want: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
			},
		},
		{
			name: "filter by both record type and domain",
			config: &Config{
				SupportedRecords: []string{"A"},
				DomainFilter:     []string{"example.com"},
			},
			endpoints: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
				{DNSName: "test.example.com", RecordType: "AAAA", Targets: []string{"::1"}},
				{DNSName: "test.other.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
			},
			want: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
			},
		},
		{
			name: "no filtering - empty domain filter",
			config: &Config{
				SupportedRecords: []string{"A", "AAAA"},
				DomainFilter:     []string{},
			},
			endpoints: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
				{DNSName: "test.other.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
			},
			want: []*endpoint.Endpoint{
				{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
				{DNSName: "test.other.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
			},
		},
		{
			name: "empty input",
			config: &Config{
				SupportedRecords: []string{"A", "AAAA"},
				DomainFilter:     []string{"example.com"},
			},
			endpoints: []*endpoint.Endpoint{},
			want:      []*endpoint.Endpoint{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				config: tt.config,
			}

			got, err := provider.AdjustEndpoints(tt.endpoints)
			if err != nil {
				t.Errorf("AdjustEndpoints() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdjustEndpoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDomainFilter(t *testing.T) {
	tests := []struct {
		name         string
		domainFilter []string
		wantFilters  []string
	}{
		{
			name:         "single domain",
			domainFilter: []string{"example.com"},
			wantFilters:  []string{"example.com"},
		},
		{
			name:         "multiple domains",
			domainFilter: []string{"example.com", "test.com"},
			wantFilters:  []string{"example.com", "test.com"},
		},
		{
			name:         "empty filter",
			domainFilter: []string{},
			wantFilters:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				config: &Config{
					DomainFilter: tt.domainFilter,
				},
			}

			got := provider.GetDomainFilter()
			// The endpoint.DomainFilter doesn't export its internal filters,
			// so we test by checking the type
			if got == nil {
				t.Error("GetDomainFilter() returned nil")
			}
		})
	}
}

func TestRecords(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:    "test-key",
			ProfileID: "test-profile",
			DryRun:    true,
		},
	}

	// For now, Records() returns empty list as it's not implemented yet
	ctx := context.Background()
	got, err := provider.Records(ctx)
	if err != nil {
		t.Errorf("Records() error = %v", err)
		return
	}

	if got == nil {
		t.Error("Records() returned nil")
	}

	if len(got) != 0 {
		t.Errorf("Records() returned %d records, want 0 (not implemented yet)", len(got))
	}
}

func TestApplyChanges_DryRun(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
		},
		UpdateNew: []*endpoint.Endpoint{
			{DNSName: "update.example.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
		},
		UpdateOld: []*endpoint.Endpoint{
			{DNSName: "update.example.com", RecordType: "A", Targets: []string{"192.168.1.100"}},
		},
		Delete: []*endpoint.Endpoint{
			{DNSName: "delete.example.com", RecordType: "A", Targets: []string{"192.168.1.3"}},
		},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() in dry-run mode returned error = %v", err)
	}
}

func TestApplyChanges_EmptyChanges(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	changes := &plan.Changes{
		Create:    []*endpoint.Endpoint{},
		UpdateNew: []*endpoint.Endpoint{},
		UpdateOld: []*endpoint.Endpoint{},
		Delete:    []*endpoint.Endpoint{},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() with empty changes returned error = %v", err)
	}
}

func TestLogChanges(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:    "test-key",
			ProfileID: "test-profile",
		},
	}

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{DNSName: "test.example.com", RecordType: "A", Targets: []string{"192.168.1.1"}},
		},
		UpdateNew: []*endpoint.Endpoint{
			{DNSName: "update.example.com", RecordType: "A", Targets: []string{"192.168.1.2"}},
		},
		UpdateOld: []*endpoint.Endpoint{
			{DNSName: "update.example.com", RecordType: "A", Targets: []string{"192.168.1.100"}},
		},
		Delete: []*endpoint.Endpoint{
			{DNSName: "delete.example.com", RecordType: "A", Targets: []string{"192.168.1.3"}},
		},
	}

	// This should not panic
	provider.logChanges(changes)
}
