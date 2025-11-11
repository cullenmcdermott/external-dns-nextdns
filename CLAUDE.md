# CLAUDE.md - Instructions for AI Assistants

**CRITICAL**: Read this file BEFORE making any changes to this repository!

---

## 🎯 Project Purpose

This is a NextDNS webhook provider for external-dns that enables Kubernetes-native DNS management using NextDNS's DNS Rewrites API.

**Status**: Under active development - currently in scaffolding phase

---

## 📋 MANDATORY: Always Update the Plan

### **RULE #1: KEEP IMPLEMENTATION_PLAN.md IN SYNC**

This is **THE MOST IMPORTANT RULE** for this project.

**BEFORE starting any work**:
1. ✅ Read `IMPLEMENTATION_PLAN.md` to understand current status
2. ✅ Identify which phase/task you're working on
3. ✅ Check for any blockers or dependencies

**DURING work**:
1. ✅ Mark tasks as in-progress as you start them
2. ✅ Update with any discoveries or issues
3. ✅ Document technical decisions made

**AFTER completing work**:
1. ✅ Mark completed tasks with [x]
2. ✅ Update progress percentages
3. ✅ Add lessons learned to the plan
4. ✅ Update "Current Sprint Focus"
5. ✅ Document any deviations from original plan
6. ✅ Update "Known Issues & TODOs" section

**EXAMPLE WORKFLOW**:
```
User: "Let's implement the Records() method"

You should:
1. Read IMPLEMENTATION_PLAN.md Phase 2.3
2. Understand the requirements
3. Start implementation
4. Update IMPLEMENTATION_PLAN.md to mark Phase 2.3 tasks as [x]
5. Add any issues discovered
6. Update progress percentage
```

### Why This Matters

This project will be developed across **multiple chat sessions**. Without keeping the plan updated:
- Future sessions won't know what's done
- Duplicate work will happen
- Context will be lost
- Bugs won't be tracked

---

## 🏗️ Project Architecture

### Structure

```
.
├── cmd/webhook/           # Entry point (main.go)
├── internal/nextdns/      # Core provider logic
│   ├── config.go          # Configuration management
│   ├── provider.go        # Provider interface implementation
│   └── client.go          # NextDNS API client (TO BE CREATED)
├── pkg/webhook/           # HTTP server
│   └── server.go          # Webhook HTTP endpoints
├── deploy/                # Kubernetes manifests (TO BE CREATED)
├── test/                  # Tests (TO BE CREATED)
├── IMPLEMENTATION_PLAN.md # ⭐ THE SOURCE OF TRUTH
├── CLAUDE.md              # This file
└── README.md              # User-facing documentation
```

### Key Files

| File | Purpose | Modify Frequency |
|------|---------|------------------|
| `IMPLEMENTATION_PLAN.md` | Source of truth for project status | **EVERY SESSION** |
| `internal/nextdns/provider.go` | Core DNS logic | High |
| `internal/nextdns/config.go` | Configuration | Low |
| `cmd/webhook/main.go` | Entry point | Very Low |
| `README.md` | User documentation | Medium |
| `CLAUDE.md` | AI instructions (this file) | Low |

---

## 🎓 Key Technical Concepts

### 1. External-DNS Webhook Architecture

External-DNS calls our webhook via HTTP:
- `GET /` - Returns domain filter configuration
- `GET /records` - Fetches current DNS records
- `POST /records` - Applies DNS changes
- `POST /adjustendpoints` - Adjusts endpoints before processing

**Our provider must implement**: `provider.Provider` interface

### 2. NextDNS Rewrites API

NextDNS manages DNS via "Rewrites":
- Endpoint: `https://api.nextdns.io/profiles/{profileId}/rewrites`
- Supported: A, AAAA, CNAME records
- Auth: API key in header
- SDK: `github.com/amalucelli/nextdns-go`

### 3. Overwrite Protection

**Critical Requirement**: Don't overwrite existing records by default!

When `ALLOW_OVERWRITE=false`:
```go
// Pseudo-code
if recordExists && !config.AllowOverwrite {
    log.Warnf("⚠️  WARNING: Record exists...")
    return nil // Skip
}
```

### 4. Dry Run Mode

When `DRY_RUN=true`:
- Log all operations
- Don't make actual API calls
- Return success

---

## 🔨 Development Guidelines

### Adding New Features

1. **Check IMPLEMENTATION_PLAN.md** for planned approach
2. If not planned, discuss with user first
3. Implement following existing patterns
4. Add tests
5. Update IMPLEMENTATION_PLAN.md
6. Update README.md if user-facing

### Code Style

- Use `logrus` for logging (already imported)
- Follow Go conventions (gofmt, golint)
- Add godoc comments for exported functions
- Use structured logging with fields

**Example**:
```go
log.WithFields(log.Fields{
    "dns_name": ep.DNSName,
    "record_type": ep.RecordType,
}).Info("Creating DNS record")
```

### Error Handling

- Always wrap errors with context
- Log errors at appropriate level
- Don't panic (except in init)

**Example**:
```go
if err != nil {
    return fmt.Errorf("failed to create record %s: %w", ep.DNSName, err)
}
```

### Testing Philosophy

- Unit tests for business logic
- Integration tests for API calls (use test profile)
- E2E tests for Kubernetes integration
- Mock external dependencies

---

## 🚫 Common Pitfalls to Avoid

### ❌ DON'T

1. **DON'T** modify core interfaces without understanding impact
2. **DON'T** add dependencies without updating go.mod properly
3. **DON'T** implement features not in IMPLEMENTATION_PLAN.md without discussing
4. **DON'T** forget to update IMPLEMENTATION_PLAN.md after changes
5. **DON'T** assume A records only (NextDNS supports A, AAAA, CNAME)
6. **DON'T** enable overwrite by default (safety first!)
7. **DON'T** skip dry-run testing before implementing real API calls

### ✅ DO

1. **DO** read IMPLEMENTATION_PLAN.md before every session
2. **DO** update IMPLEMENTATION_PLAN.md after every session
3. **DO** add comprehensive logging
4. **DO** handle errors gracefully
5. **DO** test with dry-run first
6. **DO** document decisions in IMPLEMENTATION_PLAN.md
7. **DO** ask user if uncertain about requirements

---

## 📖 Context for Future Sessions

### Current Status (Updated: 2025-11-11)

**Phase**: Scaffolding ✅ COMPLETE
**Next Phase**: API Integration (Phase 2)

**What's Done**:
- ✅ Project structure created
- ✅ Go module initialized
- ✅ Configuration management implemented
- ✅ Provider skeleton created
- ✅ HTTP server skeleton created
- ✅ Dockerfile created
- ✅ Documentation created

**What's Next** (Recommended for next session):
- ⏳ Add NextDNS Go SDK dependency
- ⏳ Create NextDNS client wrapper
- ⏳ Implement Records() method (GET /records)

**Blockers**: None

**Open Questions**:
- Do we need to handle NextDNS API rate limits? (investigate in Phase 2)
- Does NextDNS support custom TTL? (check during implementation)

---

## 🔍 How to Start a New Session

### Step-by-Step

1. **Read this file** (CLAUDE.md)
2. **Read IMPLEMENTATION_PLAN.md** to understand current state
3. **Ask the user** what they want to work on
4. **Check IMPLEMENTATION_PLAN.md** for that feature
5. **Implement** following the plan
6. **Test** (at least dry-run)
7. **Update IMPLEMENTATION_PLAN.md** with progress
8. **Commit** changes with clear message

### Session Checklist

- [ ] Read CLAUDE.md
- [ ] Read IMPLEMENTATION_PLAN.md
- [ ] Understand current phase
- [ ] Identify task to work on
- [ ] Implement changes
- [ ] Update IMPLEMENTATION_PLAN.md
- [ ] Update README.md (if user-facing)
- [ ] Test changes
- [ ] Commit with descriptive message

---

## 🤔 Decision Making

### When to Make Decisions

**You CAN decide**:
- Code organization within existing structure
- Specific error messages
- Log levels for messages
- Variable names
- Internal implementation details

**You MUST ASK USER**:
- Adding new dependencies
- Changing API behavior
- Modifying configuration options
- Adding new features not in plan
- Changing project scope
- Security-related decisions

### Recording Decisions

**All significant decisions** must be recorded in IMPLEMENTATION_PLAN.md under "Technical Decisions" section.

**Template**:
```markdown
### [Decision Name]

**Decision**: [What was decided]
**Rationale**: [Why this approach]
**Alternatives Considered**: [What else was considered]
**Date**: [Date]
```

---

## 🧪 Testing Strategy

### Before Implementing API Calls

1. Implement with dry-run mode only
2. Test all code paths with dry-run
3. Add unit tests with mocks
4. Only then implement real API calls

### Integration Testing

- Use a dedicated test NextDNS profile
- Don't use production profiles
- Clean up test records after each test
- Document test profile ID in .env.example

---

## 📚 Key Resources

### Must-Read Documentation

1. [External-DNS Webhook Provider Tutorial](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/)
2. [NextDNS API Documentation](https://nextdns.github.io/api/)
3. [NextDNS Go SDK](https://github.com/amalucelli/nextdns-go)

### Reference Implementations

Good examples to learn from:
- [STACKIT Provider](https://github.com/stackitcloud/external-dns-stackit-webhook)
- [PiHole Provider](https://github.com/tarantini-io/external-dns-pihole-webhook)

---

## 🎯 Project Goals Reminder

### Primary Goals

1. ✅ Enable Kubernetes-native DNS management with NextDNS
2. ✅ Follow 2025 external-dns standards (webhook architecture)
3. ⚠️ Safety first: Don't overwrite existing records by default
4. ⏳ Support A, AAAA, and CNAME records
5. ⏳ Production-ready code with proper error handling
6. ⏳ Comprehensive testing

### Non-Goals

- ❌ Not supporting other NextDNS features (only DNS rewrites)
- ❌ Not a general-purpose NextDNS client
- ❌ Not supporting record types beyond A/AAAA/CNAME

---

## 🚀 Quick Reference Commands

### Development

```bash
# Build
go build -o webhook ./cmd/webhook

# Run
export NEXTDNS_API_KEY="test"
export NEXTDNS_PROFILE_ID="test"
./webhook

# Test
go test ./...

# Lint (when setup)
golangci-lint run
```

### Docker

```bash
# Build
docker build -t external-dns-nextdns-webhook:dev .

# Run
docker run -e NEXTDNS_API_KEY=xxx -e NEXTDNS_PROFILE_ID=yyy external-dns-nextdns-webhook:dev
```

---

## 🔄 Version History of This File

- **v1.0** (2025-11-11): Initial creation after scaffolding phase

---

## 💬 Communication Style

### With the User

- Be clear and concise
- Explain what you're doing and why
- Ask before making significant decisions
- Update them on progress
- Show them what changed

### In Code Comments

- Explain WHY, not WHAT
- Add TODOs for future work
- Reference IMPLEMENTATION_PLAN.md for complex features
- Add context for non-obvious decisions

### In Commit Messages

Use conventional commits:
```
feat: implement Records() method for fetching DNS records
fix: handle nil pointer in endpoint adjustment
docs: update IMPLEMENTATION_PLAN.md with Phase 2 progress
test: add unit tests for configuration loading
```

---

## ⚠️ Final Reminders

1. **ALWAYS READ IMPLEMENTATION_PLAN.md FIRST**
2. **ALWAYS UPDATE IMPLEMENTATION_PLAN.MD AFTER**
3. Safety first: default to NOT overwriting records
4. Test with dry-run before real API calls
5. Log everything at appropriate levels
6. Handle errors gracefully
7. Ask when uncertain

---

**This file is your guide. Follow it, and the project will stay on track across multiple sessions.**

Good luck! 🚀
