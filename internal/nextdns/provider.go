package nextdns

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

// Provider implements the external-dns provider interface for NextDNS
type Provider struct {
	provider.BaseProvider
	config *Config
	client *Client
}

// NewProvider creates a new NextDNS provider
func NewProvider(config *Config) (*Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create NextDNS API client
	client, err := NewClient(config.APIKey, config.ProfileID, config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NextDNS client: %w", err)
	}

	p := &Provider{
		config: config,
		client: client,
	}

	log.WithFields(log.Fields{
		"profile_id": config.ProfileID,
		"base_url":   config.BaseURL,
		"dry_run":    config.DryRun,
	}).Info("NextDNS provider initialized")

	// Test connection if not in dry-run mode
	if !config.DryRun {
		ctx := context.Background()
		if err := client.TestConnection(ctx); err != nil {
			log.WithError(err).Warn("Failed to connect to NextDNS API - provider will continue but may fail on actual operations")
			// Don't return error here - allow provider to start even if connection test fails
			// This is useful for scenarios where API might be temporarily unavailable
		}
	}

	return p, nil
}

// Records returns the list of DNS records from NextDNS
func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	log.Debug("Fetching records from NextDNS")

	// If in dry-run mode, return empty list
	if p.config.DryRun {
		log.Debug("Dry run mode enabled, skipping record fetch")
		return []*endpoint.Endpoint{}, nil
	}

	// Fetch all DNS rewrites from NextDNS
	rewrites, err := p.client.ListRewrites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records from NextDNS: %w", err)
	}

	log.WithField("count", len(rewrites)).Debug("Retrieved rewrites from NextDNS")

	// Convert NextDNS rewrites to external-dns endpoints
	endpoints := make([]*endpoint.Endpoint, 0, len(rewrites))
	for _, rewrite := range rewrites {
		// Skip records that don't match our domain filter
		if len(p.config.DomainFilter) > 0 && !p.matchesDomainFilter(rewrite.Name) {
			log.WithFields(log.Fields{
				"name": rewrite.Name,
				"type": rewrite.Type,
			}).Debug("Skipping record that doesn't match domain filter")
			continue
		}

		// Skip unsupported record types
		if !p.isSupportedRecordType(rewrite.Type) {
			log.WithFields(log.Fields{
				"name": rewrite.Name,
				"type": rewrite.Type,
			}).Debug("Skipping unsupported record type")
			continue
		}

		// Create endpoint from rewrite
		ep := endpoint.NewEndpoint(
			rewrite.Name,
			rewrite.Type,
			endpoint.TTL(0), // NextDNS doesn't support custom TTL
			rewrite.Content,
		)

		// Store the NextDNS rewrite ID in the endpoint's provider-specific data
		// This will be useful for updates and deletes
		if ep.ProviderSpecific == nil {
			ep.ProviderSpecific = make(endpoint.ProviderSpecific, 0)
		}
		ep.ProviderSpecific = append(ep.ProviderSpecific, endpoint.ProviderSpecificProperty{
			Name:  "nextdns-id",
			Value: rewrite.ID,
		})

		endpoints = append(endpoints, ep)

		log.WithFields(log.Fields{
			"name":    rewrite.Name,
			"type":    rewrite.Type,
			"content": rewrite.Content,
			"id":      rewrite.ID,
		}).Debug("Converted rewrite to endpoint")
	}

	log.WithField("count", len(endpoints)).Info("Successfully fetched records from NextDNS")
	return endpoints, nil
}

// ApplyChanges applies the given changes to NextDNS
func (p *Provider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	log.WithFields(log.Fields{
		"create": len(changes.Create),
		"update": len(changes.UpdateOld),
		"delete": len(changes.Delete),
	}).Info("Applying changes to NextDNS")

	if p.config.DryRun {
		log.Info("Dry run mode enabled, changes will not be applied")
		p.logChanges(changes)
		return nil
	}

	// Process creates
	for _, ep := range changes.Create {
		if err := p.createRecord(ctx, ep); err != nil {
			return fmt.Errorf("failed to create record %s: %w", ep.DNSName, err)
		}
	}

	// Process updates
	for i := range changes.UpdateOld {
		oldEp := changes.UpdateOld[i]
		newEp := changes.UpdateNew[i]
		if err := p.updateRecord(ctx, oldEp, newEp); err != nil {
			return fmt.Errorf("failed to update record %s: %w", oldEp.DNSName, err)
		}
	}

	// Process deletes
	for _, ep := range changes.Delete {
		if err := p.deleteRecord(ctx, ep); err != nil {
			return fmt.Errorf("failed to delete record %s: %w", ep.DNSName, err)
		}
	}

	log.Info("Successfully applied changes to NextDNS")
	return nil
}

// AdjustEndpoints modifies endpoints before they are processed
func (p *Provider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	log.Debugf("Adjusting %d endpoints", len(endpoints))

	adjusted := make([]*endpoint.Endpoint, 0, len(endpoints))

	for _, ep := range endpoints {
		// Filter by supported record types
		if !p.isSupportedRecordType(ep.RecordType) {
			log.Warnf("Skipping unsupported record type %s for %s", ep.RecordType, ep.DNSName)
			continue
		}

		// Apply domain filtering if configured
		if len(p.config.DomainFilter) > 0 && !p.matchesDomainFilter(ep.DNSName) {
			log.Debugf("Skipping %s as it doesn't match domain filter", ep.DNSName)
			continue
		}

		adjusted = append(adjusted, ep)
	}

	log.Debugf("Adjusted to %d endpoints", len(adjusted))
	return adjusted, nil
}

// GetDomainFilter returns the domain filter for this provider
func (p *Provider) GetDomainFilter() endpoint.DomainFilter {
	if len(p.config.DomainFilter) == 0 {
		return endpoint.NewDomainFilter([]string{})
	}
	return endpoint.NewDomainFilter(p.config.DomainFilter)
}

// isSupportedRecordType checks if the record type is supported
func (p *Provider) isSupportedRecordType(recordType string) bool {
	for _, supported := range p.config.SupportedRecords {
		if strings.EqualFold(recordType, supported) {
			return true
		}
	}
	return false
}

// matchesDomainFilter checks if a DNS name matches the domain filter
func (p *Provider) matchesDomainFilter(dnsName string) bool {
	for _, domain := range p.config.DomainFilter {
		if strings.HasSuffix(dnsName, domain) || dnsName == strings.TrimPrefix(domain, ".") {
			return true
		}
	}
	return false
}

// createRecord creates a new DNS record in NextDNS
func (p *Provider) createRecord(ctx context.Context, ep *endpoint.Endpoint) error {
	log.WithFields(log.Fields{
		"name":   ep.DNSName,
		"type":   ep.RecordType,
		"target": ep.Targets,
	}).Info("Creating record")

	// TODO: Implement actual NextDNS API call
	// This is where we'll:
	// 1. Check if record already exists
	// 2. If exists and !AllowOverwrite, emit warning
	// 3. If exists and AllowOverwrite, update it
	// 4. If doesn't exist, create it

	return nil
}

// updateRecord updates an existing DNS record in NextDNS
func (p *Provider) updateRecord(ctx context.Context, oldEp, newEp *endpoint.Endpoint) error {
	log.WithFields(log.Fields{
		"name":       oldEp.DNSName,
		"old_target": oldEp.Targets,
		"new_target": newEp.Targets,
	}).Info("Updating record")

	// TODO: Implement actual NextDNS API call
	// For now, we'll delete and recreate
	if err := p.deleteRecord(ctx, oldEp); err != nil {
		return err
	}
	return p.createRecord(ctx, newEp)
}

// deleteRecord deletes a DNS record from NextDNS
func (p *Provider) deleteRecord(ctx context.Context, ep *endpoint.Endpoint) error {
	log.WithFields(log.Fields{
		"name":   ep.DNSName,
		"type":   ep.RecordType,
		"target": ep.Targets,
	}).Info("Deleting record")

	// TODO: Implement actual NextDNS API call

	return nil
}

// logChanges logs the changes that would be applied (for dry-run mode)
func (p *Provider) logChanges(changes *plan.Changes) {
	for _, ep := range changes.Create {
		log.WithFields(log.Fields{
			"action": "CREATE",
			"name":   ep.DNSName,
			"type":   ep.RecordType,
			"target": ep.Targets,
		}).Info("Dry run: would create record")
	}

	for i := range changes.UpdateOld {
		oldEp := changes.UpdateOld[i]
		newEp := changes.UpdateNew[i]
		log.WithFields(log.Fields{
			"action":     "UPDATE",
			"name":       oldEp.DNSName,
			"old_target": oldEp.Targets,
			"new_target": newEp.Targets,
		}).Info("Dry run: would update record")
	}

	for _, ep := range changes.Delete {
		log.WithFields(log.Fields{
			"action": "DELETE",
			"name":   ep.DNSName,
			"type":   ep.RecordType,
			"target": ep.Targets,
		}).Info("Dry run: would delete record")
	}
}
