package nextdns

import (
	"context"
	"fmt"

	"github.com/amalucelli/nextdns-go/nextdns"
	log "github.com/sirupsen/logrus"
)

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
func (c *Client) ListRewrites(ctx context.Context) ([]*nextdns.Rewrites, error) {
	log.WithField("profile_id", c.profileID).Debug("Listing DNS rewrites")

	request := &nextdns.ListRewritesRequest{
		ProfileID: c.profileID,
	}

	rewrites, err := c.api.Rewrites.List(ctx, request)
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
func (c *Client) CreateRewrite(ctx context.Context, name, recordType, content string) (string, error) {
	log.WithFields(log.Fields{
		"name":    name,
		"type":    recordType,
		"content": content,
	}).Debug("Creating DNS rewrite")

	request := &nextdns.CreateRewritesRequest{
		ProfileID: c.profileID,
		Rewrites: &nextdns.Rewrites{
			Name:    name,
			Type:    recordType,
			Content: content,
		},
	}

	id, err := c.api.Rewrites.Create(ctx, request)
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
func (c *Client) DeleteRewrite(ctx context.Context, id string) error {
	log.WithField("id", id).Debug("Deleting DNS rewrite")

	request := &nextdns.DeleteRewritesRequest{
		ProfileID: c.profileID,
		ID:        id,
	}

	err := c.api.Rewrites.Delete(ctx, request)
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
// NextDNS API doesn't have a native update endpoint, so we use delete + create
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
