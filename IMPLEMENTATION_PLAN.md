# Implementation Plan - NextDNS External-DNS Webhook Provider

**Status**: Phase 1 Complete (Scaffolding) ✅
**Last Updated**: 2025-11-11
**Version**: 1.0

---

## 🎯 Project Overview

This document outlines the complete implementation plan for the NextDNS webhook provider for external-dns. The provider enables Kubernetes-native DNS management using NextDNS's DNS Rewrites API.

### Key Requirements

- ✅ Use webhook architecture (no in-tree provider)
- ✅ Support A, AAAA, and CNAME records
- ⚠️ Emit warnings before overwriting existing records
- ⚠️ Only overwrite when explicitly allowed
- Follow 2025 external-dns standards

---

## 📋 Implementation Phases

### Phase 1: Scaffolding ✅ COMPLETE

**Goal**: Set up project structure and basic framework

- [x] Research existing implementations
- [x] Initialize Go module
- [x] Create directory structure
- [x] Create main.go entry point
- [x] Create configuration management (config.go)
- [x] Create provider skeleton (provider.go)
- [x] Create webhook server skeleton (server.go)
- [x] Create Dockerfile
- [x] Create README.md
- [x] Create .gitignore
- [x] Create IMPLEMENTATION_PLAN.md (this file)
- [x] Create CLAUDE.md

**Deliverables**: ✅ All files created and committed

---

### Phase 2: NextDNS API Integration ⏳ IN PROGRESS

**Goal**: Implement actual NextDNS API calls using the Go SDK

#### 2.1 Dependency Management ✅ COMPLETE

- [x] Add `github.com/amalucelli/nextdns-go` to go.mod
- [x] Add `sigs.k8s.io/external-dns` dependencies
- [x] Add `github.com/sirupsen/logrus` for logging
- [x] Run `go mod tidy`
- [x] Test that dependencies resolve

**Notes**:
- Fixed external-dns webhook API compatibility issues (v0.19.0 uses `StartHTTPApi`)
- Fixed `GetDomainFilter` to return `DomainFilterInterface` instead of `DomainFilter`
- All dependencies properly resolved

#### 2.2 NextDNS Client Setup ✅ COMPLETE

**File**: `internal/nextdns/client.go` ✅ CREATED

- [x] Create NextDNS client wrapper
- [x] Initialize client with API key and profile ID
- [x] Add connection testing method
- [x] Add error handling for API failures
- [x] Update provider.go to use client

**Implementation Details**:
- Client uses `github.com/amalucelli/nextdns-go/nextdns` SDK
- Supports dry-run mode for testing
- Implements CRUD operations: GetRewrites, CreateRewrite, UpdateRewrite, DeleteRewrite
- FindRewriteByName helper method for lookups
- Proper error handling and structured logging

#### 2.3 Records Fetching (GET /records) ✅ COMPLETE

**File**: `internal/nextdns/provider.go`

- [x] Implement `Records()` method
- [x] Fetch all DNS rewrites from NextDNS API
- [x] Convert NextDNS rewrites to external-dns Endpoints
- [x] Handle pagination if needed (not required - API returns all)
- [x] Add error handling and logging

**NextDNS API**: `GET /profiles/{profileId}/rewrites`

**Implementation Details**:
- Converts NextDNS `Rewrites` to external-dns `Endpoint` objects
- Implements `determineRecordType()` to detect A/AAAA/CNAME from content
- Stores NextDNS rewrite ID in `ProviderSpecific` metadata for updates/deletes
- Filters unsupported record types
- Proper error handling and logging

#### 2.4 Record Creation

**File**: `internal/nextdns/provider.go`

- [ ] Implement `createRecord()` method fully
- [ ] Check if record already exists
- [ ] If exists and `ALLOW_OVERWRITE=false`, emit warning and skip
- [ ] If exists and `ALLOW_OVERWRITE=true`, proceed to update
- [ ] If doesn't exist, create new rewrite
- [ ] Log all operations with appropriate levels

**NextDNS API**: `POST /profiles/{profileId}/rewrites`

**Warning Format**:
```
⚠️  WARNING: Record already exists and will NOT be overwritten
    DNS Name: app.example.com
    Current:  A -> 192.168.1.100
    Planned:  A -> 192.168.1.200
    Set ALLOW_OVERWRITE=true to enable automatic overwrites
```

#### 2.5 Record Updates

**File**: `internal/nextdns/provider.go`

- [ ] Implement `updateRecord()` method fully
- [ ] Delete old record
- [ ] Create new record
- [ ] Handle atomic operations
- [ ] Add rollback logic if needed

**NextDNS API**:
- `DELETE /profiles/{profileId}/rewrites/{id}`
- `POST /profiles/{profileId}/rewrites`

#### 2.6 Record Deletion

**File**: `internal/nextdns/provider.go`

- [ ] Implement `deleteRecord()` method fully
- [ ] Find record ID by DNS name and target
- [ ] Delete record via API
- [ ] Handle "not found" gracefully

**NextDNS API**: `DELETE /profiles/{profileId}/rewrites/{id}`

#### 2.7 Record Type Validation

**File**: `internal/nextdns/provider.go`

- [ ] Verify A record support
- [ ] Verify AAAA record support
- [ ] Verify CNAME record support
- [ ] Add validation for record types
- [ ] Update config with actual supported types if different

**Testing Checklist**:
- [ ] Test with A records
- [ ] Test with AAAA records
- [ ] Test with CNAME records
- [ ] Test with unsupported types (should reject)

---

### Phase 3: Testing & Validation ⏳ TODO

**Goal**: Ensure reliability and correctness

#### 3.1 Unit Tests

- [ ] Test configuration loading
- [ ] Test domain filtering
- [ ] Test record type validation
- [ ] Test endpoint adjustment
- [ ] Test change logging (dry-run)
- [ ] Mock NextDNS API responses
- [ ] Achieve >80% code coverage

**Files to Create**:
- `internal/nextdns/config_test.go`
- `internal/nextdns/provider_test.go`
- `pkg/webhook/server_test.go`

#### 3.2 Integration Tests

- [ ] Test against actual NextDNS API (dev profile)
- [ ] Test record creation
- [ ] Test record updates
- [ ] Test record deletion
- [ ] Test overwrite protection
- [ ] Test dry-run mode
- [ ] Test domain filtering

**Files to Create**:
- `test/integration/nextdns_test.go`
- `test/integration/e2e_test.go`

#### 3.3 Manual Testing

- [ ] Deploy to test Kubernetes cluster
- [ ] Create test Ingress resources
- [ ] Verify DNS records created in NextDNS
- [ ] Test record updates
- [ ] Test record deletion on Ingress removal
- [ ] Test overwrite scenarios

---

### Phase 4: Kubernetes Integration ⏳ TODO

**Goal**: Make it production-ready for Kubernetes

#### 4.1 Kubernetes Manifests

**Directory**: `deploy/kubernetes/` (NEW)

- [ ] Create ServiceAccount
- [ ] Create RBAC (if needed)
- [ ] Create ConfigMap for configuration
- [ ] Create Secret for API key
- [ ] Create Deployment with sidecar pattern
- [ ] Create Service for health checks
- [ ] Create example Ingress for testing

**Files to Create**:
- `deploy/kubernetes/serviceaccount.yaml`
- `deploy/kubernetes/configmap.yaml`
- `deploy/kubernetes/secret.yaml.example`
- `deploy/kubernetes/deployment.yaml`
- `deploy/kubernetes/service.yaml`
- `deploy/kubernetes/example-ingress.yaml`

#### 4.2 Helm Chart (Optional but Recommended)

**Directory**: `charts/external-dns-nextdns-webhook/` (NEW)

- [ ] Create Chart.yaml
- [ ] Create values.yaml with all configuration options
- [ ] Create templates for all resources
- [ ] Add NOTES.txt with usage instructions
- [ ] Test Helm installation

#### 4.3 Documentation

- [ ] Add Kubernetes installation guide to README
- [ ] Add Helm installation guide
- [ ] Add troubleshooting section
- [ ] Add examples for different use cases

---

### Phase 5: Advanced Features ⏳ TODO

**Goal**: Add production-grade features

#### 5.1 Metrics & Monitoring

- [ ] Add Prometheus metrics endpoint
- [ ] Track record operations (create/update/delete)
- [ ] Track API call durations
- [ ] Track error rates
- [ ] Add Grafana dashboard example

**File**: `pkg/webhook/metrics.go` (NEW)

#### 5.2 Enhanced Logging

- [ ] Add structured logging
- [ ] Add request IDs for tracing
- [ ] Add log sampling for high-volume
- [ ] Add sensitive data redaction

#### 5.3 Rate Limiting

- [ ] Implement rate limiting for NextDNS API
- [ ] Add backoff/retry logic
- [ ] Handle API quota errors gracefully

**File**: `internal/nextdns/ratelimit.go` (NEW)

#### 5.4 Caching

- [ ] Cache DNS records to reduce API calls
- [ ] Implement cache invalidation
- [ ] Add cache metrics

**File**: `internal/nextdns/cache.go` (NEW)

---

### Phase 6: Documentation & Release ⏳ TODO

**Goal**: Prepare for public release

#### 6.1 Documentation

- [ ] Complete API documentation (GoDoc)
- [ ] Add architecture diagrams
- [ ] Create CONTRIBUTING.md
- [ ] Create SECURITY.md
- [ ] Add LICENSE file (choose license)
- [ ] Create CHANGELOG.md
- [ ] Add usage examples
- [ ] Add FAQ section

#### 6.2 CI/CD

- [ ] Set up GitHub Actions
- [ ] Add linting (golangci-lint)
- [ ] Add automated tests
- [ ] Add Docker image building
- [ ] Add security scanning
- [ ] Add release automation

**File**: `.github/workflows/ci.yml` (NEW)

#### 6.3 Release Preparation

- [ ] Create GitHub release
- [ ] Publish Docker image to registry
- [ ] Announce on external-dns issue #3709
- [ ] Submit to external-dns documentation
- [ ] Create blog post (optional)

---

## 🔧 Technical Decisions

### Record Type Support

**Decision**: Support A, AAAA, and CNAME records
**Rationale**: These are the record types supported by NextDNS Rewrites API
**Date**: 2025-11-11

### Overwrite Behavior

**Decision**: Default to NOT overwriting existing records
**Rationale**: Safety first - prevent accidental overwrites in production
**Configuration**: `ALLOW_OVERWRITE` environment variable
**Date**: 2025-11-11

### API Library Choice

**Decision**: Use `github.com/amalucelli/nextdns-go`
**Rationale**: Mature, well-maintained Go SDK for NextDNS API
**Alternative Considered**: Implement direct HTTP client (rejected - reinventing wheel)
**Date**: 2025-11-11

### Webhook Architecture

**Decision**: Use external-dns webhook provider interface
**Rationale**: Required by external-dns as of 2025 - no in-tree providers
**Date**: 2025-11-11

---

## 📊 Progress Tracking

### Overall Progress

- [x] Phase 1: Scaffolding (100%)
- [~] Phase 2: API Integration (43%) - 3 of 7 sub-phases complete
- [ ] Phase 3: Testing (0%)
- [ ] Phase 4: Kubernetes Integration (0%)
- [ ] Phase 5: Advanced Features (0%)
- [ ] Phase 6: Documentation & Release (0%)

**Overall**: 23.9% Complete (Phase 1: 16.7% + Phase 2: 7.2%)

### Current Sprint Focus

**Sprint 1**: ✅ Complete
- Scaffolding and project setup

**Sprint 2**: ✅ Complete (2025-11-12)
- Phase 2.1: Dependency Management ✅
- Phase 2.2: NextDNS Client Setup ✅
- Phase 2.3: Records Fetching ✅

**Sprint 3**: ⏳ Next (Recommended for next session)
- Phase 2.4: Record Creation (with overwrite protection)
- Phase 2.5: Record Updates
- Phase 2.6: Record Deletion
- Phase 2.7: Record Type Validation

---

## 🚦 Testing Strategy

### Unit Tests
- Mock NextDNS API client
- Test all provider methods in isolation
- Test configuration parsing
- Test error handling

### Integration Tests
- Use test NextDNS profile
- Test actual API calls
- Test end-to-end flows
- Clean up test records after

### E2E Tests
- Deploy to test cluster
- Test with real Kubernetes resources
- Verify DNS records in NextDNS
- Test update and delete flows

---

## 🐛 Known Issues & TODOs

### Current Issues
- [ ] None yet (scaffolding phase)

### Future Considerations
- [ ] NextDNS API rate limits (investigate)
- [ ] Support for DNS record TTL (check if NextDNS supports)
- [ ] Support for TXT records (check if NextDNS supports)
- [ ] Bulk operations for performance
- [ ] Multi-profile support (if needed)

---

## 📚 Resources

### External-DNS
- [Webhook Provider Documentation](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/)
- [Provider Interface](https://pkg.go.dev/sigs.k8s.io/external-dns/provider)
- [Webhook API](https://pkg.go.dev/sigs.k8s.io/external-dns/provider/webhook/api)

### NextDNS
- [API Documentation](https://nextdns.github.io/api/)
- [Go SDK](https://github.com/amalucelli/nextdns-go)
- [Help Center](https://help.nextdns.io/)

### Reference Implementations
- [STACKIT Webhook Provider](https://github.com/stackitcloud/external-dns-stackit-webhook)
- [PiHole Webhook Provider](https://github.com/tarantini-io/external-dns-pihole-webhook)

---

## 🔄 Maintenance Notes

### Keeping This Plan Updated

**CRITICAL**: As implementation progresses, this plan MUST be kept up to date.

**After each work session**:
1. Mark completed tasks as [x]
2. Update progress percentages
3. Add any new issues discovered
4. Update technical decisions if changed
5. Add lessons learned
6. Update sprint focus

**Before starting work**:
1. Review this plan
2. Identify current phase
3. Check dependencies
4. Review technical decisions

---

## 💡 Lessons Learned

*This section will be updated as implementation progresses*

### Session 1 (2025-11-11): Scaffolding
- NextDNS supports A, AAAA, and CNAME (not just A as initially thought)
- Webhook architecture is mandatory for new providers
- amalucelli/nextdns-go SDK exists and is mature
- Good reference implementations exist for guidance

### Session 2 (2025-11-12): API Integration (Phase 2.1-2.3)
- External-dns v0.19.0 uses `StartHTTPApi` instead of handler-based approach
- The `GetDomainFilter()` must return `DomainFilterInterface` (pointer type)
- NextDNS SDK uses `Rewrites` (plural) for the type and service names
- NextDNS API structure uses request objects (e.g., `CreateRewritesRequest`)
- NextDNS rewrite fields: `ID`, `Name`, `Type`, `Content` (not `Answer`)
- Record type detection needed: Type field may be empty for A/AAAA records
- Storing NextDNS ID in `ProviderSpecific` metadata enables efficient updates
- NextDNS API doesn't have separate Update endpoint (delete + create pattern)

---

## 🤝 Contributors

- Initial scaffolding: Claude (AI Assistant)
- [Add human contributors as they join]

---

**Remember**: This is a living document. Keep it updated with every significant change!
