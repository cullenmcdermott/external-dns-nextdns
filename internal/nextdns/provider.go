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
	client, err := NewClient(config.APIKey, config.ProfileID, config.DryRun)
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

	return p, nil
}

// Records returns the list of DNS records from NextDNS
func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	log.Debug("Fetching records from NextDNS")

	// Fetch rewrites from NextDNS
	rewrites, err := p.client.GetRewrites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rewrites from NextDNS: %w", err)
	}

	// Convert NextDNS rewrites to external-dns endpoints
	endpoints := make([]*endpoint.Endpoint, 0, len(rewrites))
	for _, rewrite := range rewrites {
		// Skip if the rewrite doesn't have required fields
		if rewrite.Name == "" || rewrite.Content == "" {
			log.Warnf("Skipping rewrite with missing fields: ID=%s", rewrite.ID)
			continue
		}

		// Determine the record type
		// NextDNS Type field should be empty for A/AAAA (auto-detected) or "CNAME" for CNAME
		recordType := p.determineRecordType(rewrite.Type, rewrite.Content)

		// Skip unsupported record types
		if !p.isSupportedRecordType(recordType) {
			log.Debugf("Skipping unsupported record type %s for %s", recordType, rewrite.Name)
			continue
		}

		// Create endpoint
		ep := &endpoint.Endpoint{
			DNSName:    rewrite.Name,
			RecordType: recordType,
			Targets:    endpoint.Targets{rewrite.Content},
			RecordTTL:  endpoint.TTL(p.config.DefaultTTL),
		}

		// Store the NextDNS rewrite ID in provider specific metadata
		// This will help with updates and deletes
		if ep.ProviderSpecific == nil {
			ep.ProviderSpecific = endpoint.ProviderSpecific{}
		}
		ep.ProviderSpecific = append(ep.ProviderSpecific, endpoint.ProviderSpecificProperty{
			Name:  "nextdns-id",
			Value: rewrite.ID,
		})

		endpoints = append(endpoints, ep)
	}

	log.WithField("count", len(endpoints)).Info("Records fetched from NextDNS")
	return endpoints, nil
}

// determineRecordType determines the DNS record type from NextDNS rewrite
func (p *Provider) determineRecordType(rewriteType, content string) string {
	// If type is explicitly set to CNAME, use it
	if strings.EqualFold(rewriteType, "CNAME") {
		return "CNAME"
	}

	// Otherwise, determine by content format
	// If content contains a dot and doesn't look like an IP, it might be a domain (CNAME)
	// If it's an IPv4 address, it's an A record
	// If it's an IPv6 address, it's an AAAA record

	// Check for IPv6 (contains colons)
	if strings.Contains(content, ":") {
		return "AAAA"
	}

	// Check for IPv4 (simple check: four numbers separated by dots)
	parts := strings.Split(content, ".")
	if len(parts) == 4 {
		// Assume it's an A record if it looks like an IP
		return "A"
	}

	// Default to CNAME for domain names
	return "CNAME"
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
func (p *Provider) GetDomainFilter() endpoint.DomainFilterInterface {
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
