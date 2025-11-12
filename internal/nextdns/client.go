package nextdns

import (
	"context"
	"fmt"

	nextdnsapi "github.com/amalucelli/nextdns-go/nextdns"
	log "github.com/sirupsen/logrus"
)

// Client wraps the NextDNS API client
type Client struct {
	client    *nextdnsapi.Client
	profileID string
	dryRun    bool
}

// NewClient creates a new NextDNS API client
func NewClient(apiKey, profileID string, dryRun bool) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if profileID == "" {
		return nil, fmt.Errorf("profile ID cannot be empty")
	}

	// Create the NextDNS SDK client with API key
	client, err := nextdnsapi.New(nextdnsapi.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create NextDNS client: %w", err)
	}

	c := &Client{
		client:    client,
		profileID: profileID,
		dryRun:    dryRun,
	}

	log.WithFields(log.Fields{
		"profile_id": profileID,
		"dry_run":    dryRun,
	}).Info("NextDNS client initialized")

	return c, nil
}

// TestConnection tests the connection to the NextDNS API
func (c *Client) TestConnection(ctx context.Context) error {
	if c.dryRun {
		log.Info("Dry run mode: skipping connection test")
		return nil
	}

	log.Debug("Testing connection to NextDNS API")

	// Try to list rewrites as a connection test
	_, err := c.client.Rewrites.List(ctx, &nextdnsapi.ListRewritesRequest{
		ProfileID: c.profileID,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to NextDNS API: %w", err)
	}

	log.Info("Successfully connected to NextDNS API")
	return nil
}

// GetRewrites fetches all DNS rewrites from NextDNS
func (c *Client) GetRewrites(ctx context.Context) ([]*nextdnsapi.Rewrites, error) {
	if c.dryRun {
		log.Info("Dry run mode: returning empty rewrite list")
		return []*nextdnsapi.Rewrites{}, nil
	}

	log.WithField("profile_id", c.profileID).Debug("Fetching DNS rewrites from NextDNS")

	rewrites, err := c.client.Rewrites.List(ctx, &nextdnsapi.ListRewritesRequest{
		ProfileID: c.profileID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rewrites: %w", err)
	}

	log.WithField("count", len(rewrites)).Info("Successfully fetched DNS rewrites")
	return rewrites, nil
}

// CreateRewrite creates a new DNS rewrite in NextDNS
func (c *Client) CreateRewrite(ctx context.Context, rewrite *nextdnsapi.Rewrites) (string, error) {
	if c.dryRun {
		log.WithFields(log.Fields{
			"name":    rewrite.Name,
			"content": rewrite.Content,
			"type":    rewrite.Type,
		}).Info("Dry run mode: would create rewrite")
		return "dry-run-id", nil
	}

	log.WithFields(log.Fields{
		"name":    rewrite.Name,
		"content": rewrite.Content,
		"type":    rewrite.Type,
	}).Debug("Creating DNS rewrite in NextDNS")

	id, err := c.client.Rewrites.Create(ctx, &nextdnsapi.CreateRewritesRequest{
		ProfileID: c.profileID,
		Rewrites:  rewrite,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create rewrite: %w", err)
	}

	log.WithField("id", id).Info("Successfully created DNS rewrite")
	return id, nil
}

// UpdateRewrite updates an existing DNS rewrite in NextDNS
// Note: NextDNS API doesn't have an update endpoint, so we delete and recreate
func (c *Client) UpdateRewrite(ctx context.Context, rewriteID string, rewrite *nextdnsapi.Rewrites) (string, error) {
	if c.dryRun {
		log.WithFields(log.Fields{
			"id":      rewriteID,
			"name":    rewrite.Name,
			"content": rewrite.Content,
			"type":    rewrite.Type,
		}).Info("Dry run mode: would update rewrite (delete + create)")
		return rewriteID, nil
	}

	log.WithFields(log.Fields{
		"id":   rewriteID,
		"name": rewrite.Name,
	}).Debug("Updating DNS rewrite in NextDNS (delete + create)")

	// Delete old rewrite
	if err := c.DeleteRewrite(ctx, rewriteID); err != nil {
		return "", fmt.Errorf("failed to delete old rewrite: %w", err)
	}

	// Create new rewrite
	newID, err := c.CreateRewrite(ctx, rewrite)
	if err != nil {
		return "", fmt.Errorf("failed to create new rewrite: %w", err)
	}

	log.WithField("new_id", newID).Info("Successfully updated DNS rewrite")
	return newID, nil
}

// DeleteRewrite deletes a DNS rewrite from NextDNS
func (c *Client) DeleteRewrite(ctx context.Context, rewriteID string) error {
	if c.dryRun {
		log.WithField("id", rewriteID).Info("Dry run mode: would delete rewrite")
		return nil
	}

	log.WithField("id", rewriteID).Debug("Deleting DNS rewrite from NextDNS")

	err := c.client.Rewrites.Delete(ctx, &nextdnsapi.DeleteRewritesRequest{
		ProfileID: c.profileID,
		ID:        rewriteID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete rewrite: %w", err)
	}

	log.WithField("id", rewriteID).Info("Successfully deleted DNS rewrite")
	return nil
}

// FindRewriteByName finds a rewrite by DNS name
// Returns the rewrite and true if found, nil and false if not found
func (c *Client) FindRewriteByName(ctx context.Context, dnsName string) (*nextdnsapi.Rewrites, bool, error) {
	rewrites, err := c.GetRewrites(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, rewrite := range rewrites {
		if rewrite.Name == dnsName {
			return rewrite, true, nil
		}
	}

	return nil, false, nil
}
