package nextdns

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the configuration for the NextDNS provider
type Config struct {
	// NextDNS API configuration
	APIKey    string
	ProfileID string
	BaseURL   string

	// Server configuration
	ServerPort int
	HealthPort int

	// Domain filtering
	DomainFilter []string

	// Behavior configuration
	DryRun           bool
	LogLevel         string
	SupportedRecords []string
	DefaultTTL       int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		APIKey:           getEnv("NEXTDNS_API_KEY", ""),
		ProfileID:        getEnv("NEXTDNS_PROFILE_ID", ""),
		BaseURL:          getEnv("NEXTDNS_BASE_URL", "https://api.nextdns.io"),
		ServerPort:       getEnvInt("SERVER_PORT", 8888),
		HealthPort:       getEnvInt("HEALTH_PORT", 8080),
		DryRun:           getEnvBool("DRY_RUN", false),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		SupportedRecords: getEnvList("SUPPORTED_RECORDS", []string{"A", "AAAA", "CNAME"}),
		DefaultTTL:       getEnvInt("DEFAULT_TTL", 300),
	}

	// Domain filter
	domainFilterStr := getEnv("DOMAIN_FILTER", "")
	if domainFilterStr != "" {
		config.DomainFilter = strings.Split(domainFilterStr, ",")
		for i := range config.DomainFilter {
			config.DomainFilter[i] = strings.TrimSpace(config.DomainFilter[i])
		}
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, fmt.Errorf("NEXTDNS_API_KEY environment variable is required")
	}

	if config.ProfileID == "" {
		return nil, fmt.Errorf("NEXTDNS_PROFILE_ID environment variable is required")
	}

	return config, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvList gets a comma-separated list from environment variable
func getEnvList(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	values := strings.Split(valueStr, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}
