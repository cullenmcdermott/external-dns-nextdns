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

// overwriteAnnotationKey is the Kubernetes annotation key used to control
// per-record overwrite behavior. When set to "true" (case-insensitive),
// it allows the provider to overwrite existing DNS records.
const overwriteAnnotationKey = "external-dns.alpha.kubernetes.io/nextdns-allow-overwrite"

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
		} else {
			log.WithFields(log.Fields{
				"profile_id": config.ProfileID,
			}).Info("NextDNS connection verified - successfully authenticated with NextDNS API")
		}
	} else {
		log.WithFields(log.Fields{
			"profile_id": config.ProfileID,
		}).Info("Dry-run mode enabled - skipping NextDNS API connection test")
	}

	return p, nil
}

// Records returns the list of DNS records from NextDNS
func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	log.Debug("Fetching records from NextDNS")

	// Handle nil client (e.g., in tests)
	if p.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// Fetch all rewrites from NextDNS API
	rewrites, err := p.client.ListRewrites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list rewrites: %w", err)
	}

	// Convert NextDNS rewrites to external-dns endpoints
	endpoints := make([]*endpoint.Endpoint, 0, len(rewrites))
	for _, rewrite := range rewrites {
		ep := &endpoint.Endpoint{
			DNSName:    rewrite.Name,
			Targets:    []string{rewrite.Content},
			RecordType: rewrite.Type,
		}
		endpoints = append(endpoints, ep)
	}

	log.WithField("count", len(endpoints)).Info("Records fetched from NextDNS")
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
		p.logChanges(ctx, changes)
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

// parseOverwriteAnnotation checks the endpoint's ProviderSpecific annotations
// for the overwrite permission annotation. Returns true if the annotation
// is present and set to "true" (case-insensitive), false otherwise.
// Default behavior when annotation is absent: block overwrite (return false).
func parseOverwriteAnnotation(ep *endpoint.Endpoint) bool {
	if ep == nil {
		return false
	}

	for _, prop := range ep.ProviderSpecific {
		if prop.Name == overwriteAnnotationKey {
			allowed := strings.EqualFold(prop.Value, "true")
			log.WithFields(log.Fields{
				"dns_name":          ep.DNSName,
				"annotation_key":    overwriteAnnotationKey,
				"annotation_value":  prop.Value,
				"overwrite_allowed": allowed,
			}).Debug("Parsed overwrite annotation")
			return allowed
		}
	}

	log.WithFields(log.Fields{
		"dns_name":          ep.DNSName,
		"annotation_key":    overwriteAnnotationKey,
		"overwrite_allowed": false,
	}).Debug("Overwrite annotation not present, defaulting to block overwrite")
	return false
}

// createRecord creates a new DNS record in NextDNS
func (p *Provider) createRecord(ctx context.Context, ep *endpoint.Endpoint) error {
	// Skip unsupported record types (e.g., TXT records used by external-dns registry)
	if !p.isSupportedRecordType(ep.RecordType) {
		log.WithFields(log.Fields{
			"name": ep.DNSName,
			"type": ep.RecordType,
		}).Debug("Skipping unsupported record type")
		return nil
	}

	log.WithFields(log.Fields{
		"name":   ep.DNSName,
		"type":   ep.RecordType,
		"target": ep.Targets,
	}).Info("Creating record")

	// Handle multiple targets (create one rewrite per target)
	for _, target := range ep.Targets {
		// Check if record already exists
		existing, found, err := p.client.FindRewriteByName(ctx, ep.DNSName, ep.RecordType)
		if err != nil {
			return fmt.Errorf("failed to check for existing record: %w", err)
		}

		if found {
			// Record exists - check overwrite policy via annotation
			if !parseOverwriteAnnotation(ep) {
				// Emit warning and skip
				log.WithFields(log.Fields{
					"dns_name":      ep.DNSName,
					"record_type":   ep.RecordType,
					"current_value": existing.Content,
					"planned_value": target,
				}).Warn("Record already exists and will NOT be overwritten. " +
					"To allow overwrite, add annotation: " +
					overwriteAnnotationKey + ": \"true\"")
				continue
			}

			// Overwrite is allowed via annotation - update the record
			log.WithFields(log.Fields{
				"dns_name":    ep.DNSName,
				"record_type": ep.RecordType,
				"old_value":   existing.Content,
				"new_value":   target,
			}).Info("Overwriting existing record (annotation allows overwrite)")

			_, err = p.client.UpdateRewrite(ctx, existing.ID, ep.DNSName, ep.RecordType, target)
			if err != nil {
				return fmt.Errorf("failed to update existing record: %w", err)
			}
		} else {
			// Record doesn't exist - create it
			_, err = p.client.CreateRewrite(ctx, ep.DNSName, ep.RecordType, target)
			if err != nil {
				return fmt.Errorf("failed to create record: %w", err)
			}
		}
	}

	return nil
}

// updateRecord updates an existing DNS record in NextDNS
func (p *Provider) updateRecord(ctx context.Context, oldEp, newEp *endpoint.Endpoint) error {
	// Skip unsupported record types
	if !p.isSupportedRecordType(oldEp.RecordType) {
		log.WithFields(log.Fields{
			"name": oldEp.DNSName,
			"type": oldEp.RecordType,
		}).Debug("Skipping update for unsupported record type")
		return nil
	}

	log.WithFields(log.Fields{
		"operation":  "update",
		"dns_name":   oldEp.DNSName,
		"old_target": oldEp.Targets,
		"new_target": newEp.Targets,
	}).Info("Updating record")

	// NextDNS doesn't have a native update API - we use delete + create pattern
	// First, delete the old record
	if err := p.deleteRecord(ctx, oldEp); err != nil {
		log.WithFields(log.Fields{
			"operation":   "update",
			"phase":       "delete",
			"dns_name":    oldEp.DNSName,
			"record_type": oldEp.RecordType,
			"old_target":  oldEp.Targets,
			"error":       err.Error(),
		}).Error("Failed to delete old record during update")
		return fmt.Errorf("failed to delete old record during update: %w", err)
	}

	// Then create the new record
	if err := p.createRecord(ctx, newEp); err != nil {
		log.WithFields(log.Fields{
			"operation":   "update",
			"phase":       "create",
			"dns_name":    newEp.DNSName,
			"record_type": newEp.RecordType,
			"old_target":  oldEp.Targets,
			"new_target":  newEp.Targets,
			"error":       err.Error(),
		}).Error("Failed to create new record during update")
		log.WithFields(log.Fields{
			"dns_name":   newEp.DNSName,
			"old_target": oldEp.Targets,
			"new_target": newEp.Targets,
		}).Warn("DNS record is in inconsistent state - old record deleted but new record not created")
		return fmt.Errorf("failed to create new record during update: %w", err)
	}

	log.WithFields(log.Fields{
		"operation":   "update",
		"dns_name":    newEp.DNSName,
		"record_type": newEp.RecordType,
		"old_target":  oldEp.Targets,
		"new_target":  newEp.Targets,
	}).Info("Successfully updated record")

	return nil
}

// deleteRecord deletes a DNS record from NextDNS
func (p *Provider) deleteRecord(ctx context.Context, ep *endpoint.Endpoint) error {
	// Skip unsupported record types
	if !p.isSupportedRecordType(ep.RecordType) {
		log.WithFields(log.Fields{
			"name": ep.DNSName,
			"type": ep.RecordType,
		}).Debug("Skipping delete for unsupported record type")
		return nil
	}

	log.WithFields(log.Fields{
		"name":   ep.DNSName,
		"type":   ep.RecordType,
		"target": ep.Targets,
	}).Info("Deleting record")

	// Handle nil client (e.g., in tests)
	if p.client == nil {
		return fmt.Errorf("client not initialized")
	}

	// Handle multiple targets (delete each matching rewrite)
	for _, target := range ep.Targets {
		// Find the record by name and type
		existing, found, err := p.client.FindRewriteByName(ctx, ep.DNSName, ep.RecordType)
		if err != nil {
			return fmt.Errorf("failed to find record for deletion: %w", err)
		}

		if !found {
			// Record doesn't exist - log warning but don't fail (idempotency)
			log.WithFields(log.Fields{
				"dns_name":    ep.DNSName,
				"record_type": ep.RecordType,
				"target":      target,
			}).Warn("Record not found for deletion, may have already been deleted")
			continue
		}

		// Delete the record
		err = p.client.DeleteRewrite(ctx, existing.ID)
		if err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}

		log.WithFields(log.Fields{
			"id":          existing.ID,
			"dns_name":    ep.DNSName,
			"record_type": ep.RecordType,
		}).Info("Successfully deleted record")
	}

	return nil
}

// logChanges logs the changes that would be applied (for dry-run mode)
func (p *Provider) logChanges(ctx context.Context, changes *plan.Changes) {
	// Fetch current records for comparison
	currentRecords, err := p.Records(ctx)
	if err != nil {
		log.WithError(err).Warn("Failed to fetch current records for dry-run comparison")
		currentRecords = []*endpoint.Endpoint{}
	}

	// Build lookup map for current records
	currentByName := make(map[string]*endpoint.Endpoint)
	for _, ep := range currentRecords {
		key := fmt.Sprintf("%s/%s", ep.DNSName, ep.RecordType)
		currentByName[key] = ep
	}

	log.Info("=== DRY RUN PREVIEW ===")

	for _, ep := range changes.Create {
		key := fmt.Sprintf("%s/%s", ep.DNSName, ep.RecordType)
		current, exists := currentByName[key]

		fields := log.Fields{
			"action":      "CREATE",
			"dns_name":    ep.DNSName,
			"record_type": ep.RecordType,
			"target":      ep.Targets,
		}

		if exists {
			fields["current_value"] = current.Targets
			fields["conflict"] = true
			if parseOverwriteAnnotation(ep) {
				fields["overwrite"] = "allowed (annotation present)"
			} else {
				fields["overwrite"] = "blocked (annotation not present)"
			}
		}
		log.WithFields(fields).Info("Would create record")
	}

	for i := range changes.UpdateOld {
		oldEp := changes.UpdateOld[i]
		newEp := changes.UpdateNew[i]
		log.WithFields(log.Fields{
			"action":      "UPDATE",
			"dns_name":    oldEp.DNSName,
			"record_type": oldEp.RecordType,
			"current":     oldEp.Targets,
			"planned":     newEp.Targets,
		}).Info("Would update record")
	}

	for _, ep := range changes.Delete {
		log.WithFields(log.Fields{
			"action":      "DELETE",
			"dns_name":    ep.DNSName,
			"record_type": ep.RecordType,
			"target":      ep.Targets,
		}).Info("Would delete record")
	}

	log.Info("=== END DRY RUN PREVIEW ===")
}
