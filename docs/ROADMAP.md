# Project Tracking & Roadmap

Project status and progress tracking for Claim Machinery API based on [SPEC.md](./SPEC.md).

---

## 🚀 Roadmap

### Phase 1: MVP (Sprint 1-2)
**Goal:** Basic REST API with template discovery and KCL rendering
**Status:** 60% Complete - KCL rendering done, API endpoints next

- [ ] **Project Setup** ✅ DONE
  - [x] Go project initialization
  - [x] Dependency management (go.mod)
  - [x] GitHub repository configuration
  - [ ] HTTP server boilerplate (gorilla/mux) - NEXT
  - [ ] Middleware setup (logging, CORS, error handling) - NEXT
  - Issue: Completed

- [ ] **Template Handling (Struct, Read, Validate)**
  - [x] ClaimTemplate struct definition (Spec 4)
  - [x] Parameter struct definition
  - [x] OrderRequest/OrderResponse structs
  - [x] Template loading from filesystem
  - [x] YAML/JSON parsing
  - [ ] GET /api/v1/claim-templates (list) (Spec 3.3.1)
  - [ ] GET /api/v1/claim-templates/{name} (detail) (Spec 3.3.2)
  - [ ] Basic parameter validation (required, type, enum)
  - Issues:
    - #1 - Implement template discovery and serving
    - #2 - Implement parameter validation

- [x] **KCL Integration (Spec 5)** ✅ COMPLETED
  - [x] KCL CLI execution wrapper (RenderKCLFromOCI)
  - [x] KCL SDK execution wrapper (RenderKCL)
  - [x] Parameter injection (-D flags)
  - [x] Output parsing & quote normalization
  - [x] File output support
  - [x] Comprehensive testing (16 tests passing)
  - Issue: Completed - All rendering functions working

---

### Phase 2: Enhancement (Sprint 3-4)
**Goal:** Claim rendering, validation, Backstage compatibility

- [ ] **Parameter Validation (Spec 6)**
  - [ ] Required field validation
  - [ ] Type validation (string, boolean, array, number)
  - [ ] Pattern matching (regex)
  - [ ] Enum validation
  - [ ] Length constraints
  - Issues: TBD

- [ ] **Claim Rendering (Spec 3.3.3)**
  - [ ] POST /api/v1/claim-templates/{name}/order
  - [ ] Parameter validation before rendering
  - [ ] KCL execution with timeout
  - [ ] Output formatting
  - [ ] Error handling
  - Issues: TBD

- [ ] **Dry-Run Mode**
  - [ ] Validation without execution
  - [ ] Parameter preview
  - [ ] Error simulation
  - Issues: TBD

- [ ] **Error Handling (Spec 7)**
  - [ ] Custom error types
  - [ ] Standardized error responses
  - [ ] Detailed error messages
  - [ ] HTTP status codes
  - Issues: TBD

---

### Phase 3: Production Ready (Sprint 5-6)
**Goal:** Testing, monitoring, deployment, optimization

- [ ] **Testing (Spec 10)**
  - [ ] Unit tests (handlers, validators)
  - [ ] Integration tests (API endpoints)
  - [ ] KCL execution tests
  - [ ] 80%+ code coverage
  - Issues: TBD

- [ ] **Monitoring & Logging (Spec 11.3)**
  - [ ] Structured JSON logging
  - [ ] Prometheus metrics
  - [ ] Health check endpoint
  - [ ] Request/response logging
  - Issues: TBD

- [ ] **Configuration (Spec 8)**
  - [ ] Environment variable support
  - [ ] YAML config file support
  - [ ] Config validation
  - Issues: TBD

- [ ] **Deployment (Spec 11)**
  - [ ] Dockerfile creation
  - [ ] Kubernetes manifests
  - [ ] CI/CD pipeline (GitHub Actions)
  - [ ] Image registry setup
  - Issues: TBD

- [ ] **Performance & Caching (Spec 9)**
  - [ ] Template caching (in-memory)
  - [ ] TTL-based cache invalidation
  - [ ] OCI module caching
  - [ ] Load testing
  - Issues: TBD

- [ ] **Backstage Integration**
  - [ ] CORS configuration
  - [ ] Catalog compatibility
  - [ ] UI schema generation
  - Issues: TBD

---

## 🎯 Current Focus

**✅ COMPLETED:**
- Template discovery from filesystem
- ClaimTemplate struct and types
- KCL rendering (local files + OCI sources)
- Parameter extraction and injection
- Output parsing and normalization
- File output support
- Comprehensive testing (16 tests)
- Clean architecture (internal/app, internal/render, internal/claimtemplate)

**🚀 NEXT PRIORITY: REST API Implementation**
1. HTTP server setup (gorilla/mux)
2. GET /api/v1/claim-templates (list templates)
3. GET /api/v1/claim-templates/{name} (get template details)
4. POST /api/v1/claim-templates/{name}/order (render template)
5. Middleware (logging, CORS, error handling)
6. Request/response validation

---

### Sprint 1 (Week 1-2)
**Focus:** Foundation & Basic API

| Task | Owner | Status | Due | PR |
|------|-------|--------|-----|-----|
| Project Setup | @user | TODO | 2026-01-16 | |
| Template Handling (Struct, Read, Validate) | @user | TODO | 2026-01-23 | |
| KCL Integration | @user | TODO | 2026-01-23 | |

**Issues:** #1, #2

---

### Sprint 2 (Week 3-4)
**Focus:** KCL Integration & Rendering

| Task | Owner | Status | Due | PR |
|------|-------|--------|-----|-----|
| KCL Service | @user | TODO | 2026-02-06 | |
| Parameter Validation | @user | TODO | 2026-02-06 | |
| Order Endpoint | @user | TODO | 2026-02-06 | |
| Error Handling | @user | TODO | 2026-02-06 | |

---

### Sprint 3 (Week 5-6)
**Focus:** Testing & Production Readiness

| Task | Owner | Status | Due | PR |
|------|-------|--------|-----|-----|
| Unit Tests | @user | TODO | 2026-02-20 | |
| Integration Tests | @user | TODO | 2026-02-20 | |
| Logging & Monitoring | @user | TODO | 2026-02-20 | |
| Dockerization | @user | TODO | 2026-02-20 | |

---

## 🐛 Known Issues

| ID | Description | Severity | Status |
|----|-------------|----------|--------|
| - | - | - | - |

---

## 📝 Spec Changes

| Date | Section | Change | Status |
|-------|--------|---------|--------|
| 2026-01-09 | All | Initial Draft in English | ✅ Draft |
| 2026-01-09 | 1.0 | Changed from Claims processing to KCL template rendering | ✅ Draft |

---

## 📈 Metrics

### Code Quality
- **Test Coverage:** 0% → Goal: 80%+
- **Code Review:** All PRs reviewed before merge
- **Documentation:** 0% → Goal: 100%
- **Linting:** golangci-lint clean

### Performance (Target)
- **Template List Response:** < 100ms (p99)
- **Template Detail Response:** < 100ms (p99)
- **Claim Rendering:** < 5s (p99, including KCL execution)
- **Throughput:** 100+ req/s

### Release
- **Current Version:** 0.0.1 (alpha)
- **Target Release:** v1.0.0 (2026-03-01)
- **Features Implemented:** 0 / 10
- **Tests Passing:** 0 / N

---

## 🔗 Related Links

- [SPEC.md](./SPEC.md) - Technical specification
- [GitHub Project Board](https://github.com/your-org/claim-machinery-api/projects)
- [Issue Tracker](https://github.com/your-org/claim-machinery-api/issues)
- [KCL Documentation](https://kcl-lang.io/docs)
- [Backstage Documentation](https://backstage.io/docs)

---

## Contact & Support

**Tech Lead:** @user
**Questions?** Open an issue or contact the team on Slack.

---

## Deployment Timeline

| Phase | Start | End | Status |
|-------|-------|-----|--------|
| MVP | 2026-01-16 | 2026-02-06 | Planned |
| Enhancement | 2026-02-09 | 2026-02-27 | Planned |
| Production | 2026-03-02 | 2026-03-16 | Planned |
| **Release v1.0.0** | - | **2026-03-16** | **Target** |
