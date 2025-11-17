package nextdns

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			envVars: map[string]string{
				"NEXTDNS_API_KEY":    "test-api-key",
				"NEXTDNS_PROFILE_ID": "test-profile",
				"NEXTDNS_BASE_URL":   "https://test.nextdns.io",
				"SERVER_PORT":        "9999",
				"HEALTH_PORT":        "9998",
				"DRY_RUN":            "true",
				"ALLOW_OVERWRITE":    "true",
				"LOG_LEVEL":          "debug",
				"SUPPORTED_RECORDS":  "A,AAAA,CNAME,TXT",
				"DEFAULT_TTL":        "600",
				"DOMAIN_FILTER":      "example.com,test.com",
			},
			want: &Config{
				APIKey:           "test-api-key",
				ProfileID:        "test-profile",
				BaseURL:          "https://test.nextdns.io",
				ServerPort:       9999,
				HealthPort:       9998,
				DryRun:           true,
				AllowOverwrite:   true,
				LogLevel:         "debug",
				SupportedRecords: []string{"A", "AAAA", "CNAME", "TXT"},
				DefaultTTL:       600,
				DomainFilter:     []string{"example.com", "test.com"},
			},
			wantErr: false,
		},
		{
			name: "valid config with defaults",
			envVars: map[string]string{
				"NEXTDNS_API_KEY":    "test-api-key",
				"NEXTDNS_PROFILE_ID": "test-profile",
			},
			want: &Config{
				APIKey:           "test-api-key",
				ProfileID:        "test-profile",
				BaseURL:          "https://api.nextdns.io",
				ServerPort:       8888,
				HealthPort:       8080,
				DryRun:           false,
				AllowOverwrite:   false,
				LogLevel:         "info",
				SupportedRecords: []string{"A", "AAAA", "CNAME"},
				DefaultTTL:       300,
				DomainFilter:     nil,
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			envVars: map[string]string{
				"NEXTDNS_PROFILE_ID": "test-profile",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing profile ID",
			envVars: map[string]string{
				"NEXTDNS_API_KEY": "test-api-key",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "domain filter with spaces",
			envVars: map[string]string{
				"NEXTDNS_API_KEY":    "test-api-key",
				"NEXTDNS_PROFILE_ID": "test-profile",
				"DOMAIN_FILTER":      "  example.com  ,  test.com  ",
			},
			want: &Config{
				APIKey:           "test-api-key",
				ProfileID:        "test-profile",
				BaseURL:          "https://api.nextdns.io",
				ServerPort:       8888,
				HealthPort:       8080,
				DryRun:           false,
				AllowOverwrite:   false,
				LogLevel:         "info",
				SupportedRecords: []string{"A", "AAAA", "CNAME"},
				DefaultTTL:       300,
				DomainFilter:     []string{"example.com", "test.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			got, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid int",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "200",
			want:         200,
		},
		{
			name:         "invalid int",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "not-a-number",
			want:         100,
		},
		{
			name:         "empty env var",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "",
			want:         100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "true",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "false",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "1 (true)",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			want:         true,
		},
		{
			name:         "0 (false)",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			want:         false,
		},
		{
			name:         "invalid bool",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "not-a-bool",
			want:         true,
		},
		{
			name:         "empty env var",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvList(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue []string
		envValue     string
		want         []string
	}{
		{
			name:         "simple list",
			key:          "TEST_LIST",
			defaultValue: []string{"default1", "default2"},
			envValue:     "a,b,c",
			want:         []string{"a", "b", "c"},
		},
		{
			name:         "list with spaces",
			key:          "TEST_LIST",
			defaultValue: []string{"default1", "default2"},
			envValue:     "  a  ,  b  ,  c  ",
			want:         []string{"a", "b", "c"},
		},
		{
			name:         "empty env var",
			key:          "TEST_LIST",
			defaultValue: []string{"default1", "default2"},
			envValue:     "",
			want:         []string{"default1", "default2"},
		},
		{
			name:         "single item",
			key:          "TEST_LIST",
			defaultValue: []string{"default1", "default2"},
			envValue:     "single",
			want:         []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvList(tt.key, tt.defaultValue)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getEnvList() = %v, want %v", got, tt.want)
			}
		})
	}
}
