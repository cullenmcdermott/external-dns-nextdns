package nextdns

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/amalucelli/nextdns-go/nextdns"
	log "github.com/sirupsen/logrus"
)

// Retry configuration constants
const (
	maxRetryAttempts = 3
)

// retryDelays defines the exponential backoff delays for retry attempts
var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// Client wraps the NextDNS API client and provides DNS record management
type Client struct {
	api       *nextdns.Client
	profileID string
}

// NewClient creates a new NextDNS client wrapper
func NewClient(apiKey, profileID, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if profileID == "" {
		return nil, fmt.Errorf("profile ID cannot be empty")
	}

	// Build client options
	opts := []nextdns.ClientOption{
		nextdns.WithAPIKey(apiKey),
	}

	// Add custom base URL if provided
	if baseURL != "" && baseURL != "https://api.nextdns.io" {
		opts = append(opts, nextdns.WithBaseURL(baseURL))
	}

	// Create NextDNS client
	api, err := nextdns.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create NextDNS client: %w", err)
	}

	client := &Client{
		api:       api,
		profileID: profileID,
	}

	log.WithFields(log.Fields{
		"profile_id": profileID,
		"base_url":   baseURL,
	}).Debug("NextDNS client created successfully")

	return client, nil
}

// isRetryableError determines if an error is retryable based on HTTP status codes
// and error types. Retryable errors include:
// - Network timeouts
// - 5xx server errors (500, 502, 503, 504)
// - Rate limit errors (429)
// Non-retryable errors include:
// - 4xx client errors (400, 401, 403, 404)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for network timeout errors
	var netErr net.Error
	if ok := isNetError(err, &netErr); ok && netErr.Timeout() {
		return true
	}

	// Check for HTTP status codes in error message
	// NextDNS SDK returns errors with status codes in the message
	retryableStatusCodes := []int{429, 500, 502, 503, 504}
	for _, code := range retryableStatusCodes {
		if strings.Contains(errStr, fmt.Sprintf("%d", code)) ||
			strings.Contains(errStr, http.StatusText(code)) {
			return true
		}
	}

	// Check for common network error patterns
	networkErrorPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"temporary failure",
		"timeout",
		"EOF",
		"broken pipe",
	}
	for _, pattern := range networkErrorPatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	// Non-retryable 4xx errors
	nonRetryableStatusCodes := []int{400, 401, 403, 404}
	for _, code := range nonRetryableStatusCodes {
		if strings.Contains(errStr, fmt.Sprintf("%d", code)) ||
			strings.Contains(errStr, http.StatusText(code)) {
			return false
		}
	}

	return false
}

// isNetError attempts to extract a net.Error from the error chain
func isNetError(err error, target *net.Error) bool {
	if netErr, ok := err.(net.Error); ok {
		*target = netErr
		return true
	}
	return false
}

// retryWithBackoff executes an operation with exponential backoff retry logic.
// It retries the operation up to maxRetryAttempts times with delays of 1s, 2s, 4s.
// The function respects context cancellation during retry delays.
//
// Parameters:
// - ctx: Context for cancellation support
// - operation: The function to execute (returns error)
// - operationName: Name of the operation for logging purposes
//
// Returns:
// - nil if the operation succeeds
// - The last error if all retries are exhausted
// - The error immediately if it is non-retryable (4xx errors)
func retryWithBackoff(ctx context.Context, operation func() error, operationName string) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetryAttempts; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			// Success
			if attempt > 0 {
				log.WithFields(log.Fields{
					"operation": operationName,
					"attempt":   attempt + 1,
				}).Debug("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			log.WithFields(log.Fields{
				"operation": operationName,
				"error":     err.Error(),
			}).Debug("Non-retryable error encountered, failing immediately")
			return err
		}

		// Check if we have retries remaining
		if attempt >= maxRetryAttempts {
			break
		}

		// Get delay for this attempt
		delay := retryDelays[attempt]

		log.WithFields(log.Fields{
			"operation": operationName,
			"attempt":   attempt + 1,
			"error":     err.Error(),
			"delay":     delay.String(),
		}).Debug("Retryable error encountered, will retry after delay")

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	log.WithFields(log.Fields{
		"operation":    operationName,
		"max_attempts": maxRetryAttempts + 1,
		"error":        lastErr.Error(),
	}).Warn("All retry attempts exhausted")

	return lastErr
}

// TestConnection verifies that the client can communicate with the NextDNS API
func (c *Client) TestConnection(ctx context.Context) error {
	log.Debug("Testing connection to NextDNS API")

	// Try to list rewrites as a connection test
	_, err := c.ListRewrites(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	log.Info("Successfully connected to NextDNS API")
	return nil
}

// ListRewrites fetches all DNS rewrites for the configured profile
// This method includes automatic retry with exponential backoff for transient errors
func (c *Client) ListRewrites(ctx context.Context) ([]*nextdns.Rewrites, error) {
	log.WithField("profile_id", c.profileID).Debug("Listing DNS rewrites")

	var rewrites []*nextdns.Rewrites

	err := retryWithBackoff(ctx, func() error {
		request := &nextdns.ListRewritesRequest{
			ProfileID: c.profileID,
		}

		var listErr error
		rewrites, listErr = c.api.Rewrites.List(ctx, request)
		return listErr
	}, "ListRewrites")

	if err != nil {
		return nil, fmt.Errorf("failed to list rewrites: %w", err)
	}

	log.WithFields(log.Fields{
		"profile_id": c.profileID,
		"count":      len(rewrites),
	}).Debug("Successfully listed DNS rewrites")

	return rewrites, nil
}

// CreateRewrite creates a new DNS rewrite record
// This method includes automatic retry with exponential backoff for transient errors
func (c *Client) CreateRewrite(ctx context.Context, name, recordType, content string) (string, error) {
	log.WithFields(log.Fields{
		"name":    name,
		"type":    recordType,
		"content": content,
	}).Debug("Creating DNS rewrite")

	var id string

	err := retryWithBackoff(ctx, func() error {
		// Note: NextDNS API does not accept the Type field on creation
		// It automatically determines the type based on the content
		request := &nextdns.CreateRewritesRequest{
			ProfileID: c.profileID,
			Rewrites: &nextdns.Rewrites{
				Name:    name,
				Content: content,
			},
		}

		var createErr error
		id, createErr = c.api.Rewrites.Create(ctx, request)
		return createErr
	}, "CreateRewrite")

	if err != nil {
		return "", fmt.Errorf("failed to create rewrite: %w", err)
	}

	log.WithFields(log.Fields{
		"id":      id,
		"name":    name,
		"type":    recordType,
		"content": content,
	}).Info("Successfully created DNS rewrite")

	return id, nil
}

// DeleteRewrite deletes a DNS rewrite record by ID
// This method includes automatic retry with exponential backoff for transient errors
func (c *Client) DeleteRewrite(ctx context.Context, id string) error {
	log.WithField("id", id).Debug("Deleting DNS rewrite")

	err := retryWithBackoff(ctx, func() error {
		request := &nextdns.DeleteRewritesRequest{
			ProfileID: c.profileID,
			ID:        id,
		}

		return c.api.Rewrites.Delete(ctx, request)
	}, "DeleteRewrite")

	if err != nil {
		return fmt.Errorf("failed to delete rewrite: %w", err)
	}

	log.WithField("id", id).Info("Successfully deleted DNS rewrite")
	return nil
}

// FindRewriteByName finds a DNS rewrite by its name and type
// Returns the rewrite and true if found, nil and false if not found
func (c *Client) FindRewriteByName(ctx context.Context, name, recordType string) (*nextdns.Rewrites, bool, error) {
	log.WithFields(log.Fields{
		"name": name,
		"type": recordType,
	}).Debug("Finding DNS rewrite by name")

	rewrites, err := c.ListRewrites(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, rewrite := range rewrites {
		if rewrite.Name == name && rewrite.Type == recordType {
			log.WithFields(log.Fields{
				"id":      rewrite.ID,
				"name":    rewrite.Name,
				"type":    rewrite.Type,
				"content": rewrite.Content,
			}).Debug("Found matching DNS rewrite")
			return rewrite, true, nil
		}
	}

	log.WithFields(log.Fields{
		"name": name,
		"type": recordType,
	}).Debug("No matching DNS rewrite found")

	return nil, false, nil
}

// UpdateRewrite updates a DNS rewrite by deleting the old one and creating a new one
// NextDNS API does not have a native update endpoint, so we use delete + create
// Note: Both DeleteRewrite and CreateRewrite have their own retry logic
func (c *Client) UpdateRewrite(ctx context.Context, id, name, recordType, content string) (string, error) {
	log.WithFields(log.Fields{
		"id":          id,
		"name":        name,
		"type":        recordType,
		"new_content": content,
	}).Debug("Updating DNS rewrite")

	// Delete the old rewrite
	if err := c.DeleteRewrite(ctx, id); err != nil {
		return "", fmt.Errorf("failed to delete old rewrite during update: %w", err)
	}

	// Create the new rewrite
	newID, err := c.CreateRewrite(ctx, name, recordType, content)
	if err != nil {
		return "", fmt.Errorf("failed to create new rewrite during update: %w", err)
	}

	log.WithFields(log.Fields{
		"old_id": id,
		"new_id": newID,
		"name":   name,
	}).Info("Successfully updated DNS rewrite")

	return newID, nil
}
