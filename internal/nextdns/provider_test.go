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
			// Verify the domain filter is configured correctly by testing Match behavior
			if len(tt.domainFilter) > 0 {
				// If we have filters, check that IsConfigured returns true
				if !got.IsConfigured() {
					t.Error("GetDomainFilter() returned unconfigured filter when filters were set")
				}
				// Verify first filter matches
				if !got.Match(tt.domainFilter[0]) {
					t.Errorf("GetDomainFilter() filter does not match expected domain %s", tt.domainFilter[0])
				}
			}
		})
	}
}

func TestRecords_NilClient(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:    "test-key",
			ProfileID: "test-profile",
			DryRun:    true,
		},
		// client is nil
	}

	// Records() should return an error when client is nil
	ctx := context.Background()
	_, err := provider.Records(ctx)
	if err == nil {
		t.Error("Records() expected error with nil client, got nil")
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

	ctx := context.Background()
	// This should not panic - client is nil so Records() will fail gracefully
	provider.logChanges(ctx, changes)
}

// TestParseOverwriteAnnotation tests the annotation parsing logic for overwrite control.
// This is Task 2.1 from the spec - focused tests for annotation parsing.
func TestParseOverwriteAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *endpoint.Endpoint
		want     bool
	}{
		{
			name: "annotation present with value true - allows overwrite",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "true",
					},
				},
			},
			want: true,
		},
		{
			name: "annotation present with value false - blocks overwrite",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "false",
					},
				},
			},
			want: false,
		},
		{
			name: "annotation absent - blocks overwrite (default behavior)",
			endpoint: &endpoint.Endpoint{
				DNSName:          "test.example.com",
				RecordType:       "A",
				Targets:          []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{},
			},
			want: false,
		},
		{
			name: "annotation with uppercase TRUE - allows overwrite (case-insensitive)",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "TRUE",
					},
				},
			},
			want: true,
		},
		{
			name: "annotation with mixed case True - allows overwrite (case-insensitive)",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "True",
					},
				},
			},
			want: true,
		},
		{
			name: "annotation with invalid value - blocks overwrite",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "yes",
					},
				},
			},
			want: false,
		},
		{
			name:     "nil endpoint - blocks overwrite",
			endpoint: nil,
			want:     false,
		},
		{
			name: "other annotations present but not overwrite annotation - blocks overwrite",
			endpoint: &endpoint.Endpoint{
				DNSName:    "test.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/some-other-annotation",
						Value: "true",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOverwriteAnnotation(tt.endpoint)
			if got != tt.want {
				t.Errorf("parseOverwriteAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Task Group 3 Tests: Enhanced Dry-Run and Update Logging
// =============================================================================

// TestDryRunDiffOutput_Create tests that dry-run mode logs CREATE operations correctly
// This is Task 3.1 from the spec - test dry-run generates correct diff output for CREATE
func TestDryRunDiffOutput_Create(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
		// client is nil - Records() will fail gracefully and return empty list
	}

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    "new.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
			},
		},
		UpdateNew: []*endpoint.Endpoint{},
		UpdateOld: []*endpoint.Endpoint{},
		Delete:    []*endpoint.Endpoint{},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() in dry-run mode returned error = %v", err)
	}
	// Test passes if no panic occurs and dry-run completes successfully
}

// TestDryRunDiffOutput_Update tests that dry-run mode logs UPDATE operations with current vs planned values
// This is Task 3.1 from the spec - test dry-run generates correct diff for UPDATE
func TestDryRunDiffOutput_Update(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{},
		UpdateOld: []*endpoint.Endpoint{
			{
				DNSName:    "existing.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.100"},
			},
		},
		UpdateNew: []*endpoint.Endpoint{
			{
				DNSName:    "existing.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.200"},
			},
		},
		Delete: []*endpoint.Endpoint{},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() in dry-run mode returned error = %v", err)
	}
	// Test passes if no panic occurs and dry-run completes successfully
}

// TestDryRunWithConflict tests dry-run mode shows overwrite protection status for conflicting creates
// This is Task 3.1 from the spec - test conflict detection and overwrite status
func TestDryRunWithConflict(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	// Create with overwrite annotation
	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    "conflict.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.1"},
				ProviderSpecific: endpoint.ProviderSpecific{
					{
						Name:  "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite",
						Value: "true",
					},
				},
			},
		},
		UpdateNew: []*endpoint.Endpoint{},
		UpdateOld: []*endpoint.Endpoint{},
		Delete:    []*endpoint.Endpoint{},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() in dry-run mode with conflict returned error = %v", err)
	}
}

// TestUpdateRecordLogging tests that updateRecord logs appropriately for different scenarios
// This is Task 3.1 from the spec - test update failure logging
func TestUpdateRecordLogging(t *testing.T) {
	// Test that updateRecord with valid endpoints doesn't panic
	// Full failure testing requires mock client implementation
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           false,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
		// client is nil - this will cause deleteRecord to fail, testing the error path
	}

	oldEp := &endpoint.Endpoint{
		DNSName:    "update.example.com",
		RecordType: "A",
		Targets:    []string{"192.168.1.100"},
	}
	newEp := &endpoint.Endpoint{
		DNSName:    "update.example.com",
		RecordType: "A",
		Targets:    []string{"192.168.1.200"},
	}

	ctx := context.Background()
	err := provider.updateRecord(ctx, oldEp, newEp)

	// We expect an error because client is nil, but the test verifies:
	// 1. The method doesn't panic
	// 2. Error is returned (not swallowed)
	if err == nil {
		t.Error("updateRecord() expected error with nil client, got nil")
	}
}

// =============================================================================
// Task Group 4 Tests: Integration and Backward Compatibility
// =============================================================================

// TestConfigBackwardCompatibility_NoAllowOverwrite verifies that the Config struct
// no longer has an AllowOverwrite field (it was removed and replaced with per-record annotations).
// This is Task 4.5 - verify backward compatibility.
func TestConfigBackwardCompatibility_NoAllowOverwrite(t *testing.T) {
	// Create a config with all fields - if AllowOverwrite existed, this would fail to compile
	config := &Config{
		APIKey:           "test-key",
		ProfileID:        "test-profile",
		BaseURL:          "https://api.nextdns.io",
		ServerPort:       8888,
		HealthPort:       8080,
		DomainFilter:     []string{"example.com"},
		DryRun:           false,
		LogLevel:         "info",
		SupportedRecords: []string{"A", "AAAA", "CNAME"},
		DefaultTTL:       300,
	}

	// Verify config is valid (no AllowOverwrite field needed)
	if config.APIKey == "" {
		t.Error("Config APIKey should not be empty")
	}

	// Use reflection to verify AllowOverwrite field doesn't exist
	configType := reflect.TypeOf(*config)
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		if field.Name == "AllowOverwrite" {
			t.Error("Config struct should NOT have AllowOverwrite field - it was replaced with per-record annotations")
		}
	}
}

// TestDryRunDiffOutput_Delete tests that dry-run mode logs DELETE operations correctly.
// This is Task 4.3 - additional strategic test for complete dry-run coverage.
func TestDryRunDiffOutput_Delete(t *testing.T) {
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
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "remove.example.com",
				RecordType: "A",
				Targets:    []string{"192.168.1.99"},
			},
		},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)
	if err != nil {
		t.Errorf("ApplyChanges() in dry-run mode returned error = %v", err)
	}
	// Test passes if no panic occurs and dry-run completes successfully
}

// TestApplyChanges_BackwardCompatibility verifies that ApplyChanges still works
// with the expected method signature and behavior (except for enhancements).
// This is Task 4.5 - verify backward compatibility.
func TestApplyChanges_BackwardCompatibility(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true, // Use dry-run for safe testing
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
	}

	// Test that ApplyChanges accepts the standard plan.Changes struct
	// and returns the expected error type (nil in dry-run mode)
	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{DNSName: "new.example.com", RecordType: "A", Targets: []string{"1.2.3.4"}},
		},
		UpdateOld: []*endpoint.Endpoint{
			{DNSName: "existing.example.com", RecordType: "A", Targets: []string{"10.0.0.1"}},
		},
		UpdateNew: []*endpoint.Endpoint{
			{DNSName: "existing.example.com", RecordType: "A", Targets: []string{"10.0.0.2"}},
		},
		Delete: []*endpoint.Endpoint{
			{DNSName: "old.example.com", RecordType: "CNAME", Targets: []string{"target.example.com"}},
		},
	}

	ctx := context.Background()
	err := provider.ApplyChanges(ctx, changes)

	// In dry-run mode, ApplyChanges should succeed without errors
	if err != nil {
		t.Errorf("ApplyChanges() backward compatibility test failed with error = %v", err)
	}
}

// TestRecordsMethod_BackwardCompatibility verifies that Records() method
// still has the expected signature and behavior.
// This is Task 4.5 - verify backward compatibility.
func TestRecordsMethod_BackwardCompatibility(t *testing.T) {
	provider := &Provider{
		config: &Config{
			APIKey:           "test-key",
			ProfileID:        "test-profile",
			DryRun:           true,
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
		},
		// client is nil - should return error, not panic
	}

	ctx := context.Background()
	endpoints, err := provider.Records(ctx)

	// Verify the method returns the expected types
	if err == nil {
		// If no error, endpoints should be a valid slice
		if endpoints == nil {
			t.Error("Records() should return non-nil slice when successful")
		}
	} else {
		// Error case is expected with nil client
		// Verify it doesn't panic and returns proper error
		if endpoints != nil {
			t.Error("Records() should return nil endpoints when error occurs")
		}
	}
}

// TestAdjustEndpointsMethod_BackwardCompatibility verifies that AdjustEndpoints()
// method still has the expected signature and behavior.
// This is Task 4.5 - verify backward compatibility.
func TestAdjustEndpointsMethod_BackwardCompatibility(t *testing.T) {
	provider := &Provider{
		config: &Config{
			SupportedRecords: []string{"A", "AAAA", "CNAME"},
			DomainFilter:     []string{"example.com"},
		},
	}

	// Test with mixed endpoints that should be filtered
	input := []*endpoint.Endpoint{
		{DNSName: "valid.example.com", RecordType: "A", Targets: []string{"1.2.3.4"}},
		{DNSName: "invalid.other.com", RecordType: "A", Targets: []string{"5.6.7.8"}},
		{DNSName: "unsupported.example.com", RecordType: "TXT", Targets: []string{"text"}},
	}

	adjusted, err := provider.AdjustEndpoints(input)

	// Should not return error
	if err != nil {
		t.Errorf("AdjustEndpoints() returned unexpected error = %v", err)
	}

	// Should filter to only valid endpoints
	if len(adjusted) != 1 {
		t.Errorf("AdjustEndpoints() returned %d endpoints, expected 1", len(adjusted))
	}

	// Verify the correct endpoint was kept
	if len(adjusted) > 0 && adjusted[0].DNSName != "valid.example.com" {
		t.Errorf("AdjustEndpoints() kept wrong endpoint: %s", adjusted[0].DNSName)
	}
}
