# Project Tracking & Roadmap

Project status and progress tracking for Claim Machinery API based on [SPEC.md](./SPEC.md).

---

## Roadmap

### Phase 1: MVP
**Goal:** Basic REST API with template discovery and KCL rendering
**Status:** COMPLETE

- [x] Go project setup with dependency management
- [x] HTTP server (gorilla/mux) with middleware (logging, CORS, error handling)
- [x] ClaimTemplate / Parameter / OrderRequest / OrderResponse structs
- [x] Template loading from filesystem and YAML/JSON parsing
- [x] GET /api/v1/claim-templates (list)
- [x] GET /api/v1/claim-templates/{name} (detail)
- [x] POST /api/v1/claim-templates/{name}/order (render)
- [x] Basic parameter validation (required, type, enum)
- [x] KCL CLI wrapper (RenderKCLFromOCI) and SDK wrapper (RenderKCL)
- [x] Parameter injection (-D flags), output parsing, file output
- [x] GET /health endpoint
- [x] Multi-stage Dockerfile (Alpine-based)
- [x] Documentation (API summary, testing guide, API examples, Backstage compatibility)

---

### Phase 2: Enhancement
**Goal:** Advanced features, observability, Backstage integration, interactive CLI
**Status:** IN PROGRESS

#### Completed

- [x] **Backstage Integration**
  - [x] API structure compatible with Custom Field Extensions
  - [x] Parameter metadata for UI rendering (types, enums, patterns, defaults)
  - [x] Validation rules support (pattern, enum, required)
  - [x] OpenAPI/Swagger endpoints (`/openapi.yaml`, `/docs` via Redoc)
  - [x] catalog-info.yaml for Backstage catalog

- [x] **Observability**
  - [x] Request correlation IDs (X-Request-ID generation and propagation)
  - [x] Structured JSON logging (`LOG_FORMAT=json`)
  - [x] Panic recovery middleware with JSON error body and requestId
  - [x] Request/response timing in logs (duration, status, bytes)

- [x] **Interactive CLI** (`claim-machinery-api render`)
  - [x] Template selection with interactive forms (charmbracelet/huh)
  - [x] Dynamic parameter forms (enums as dropdowns, multiselect, boolean, integer)
  - [x] Random value selection for enum parameters
  - [x] Default values pre-filled, hidden parameter support
  - [x] Live YAML preview and optional file save

- [x] **Template Profile System**
  - [x] Load templates from URLs and local paths via profile YAML
  - [x] Merge and deduplicate templates (profile overrides directory on conflict)
  - [x] Unreachable entries trigger warning and are skipped
  - [x] Flag, env var, and default path resolution

- [x] **Configuration**
  - [x] Environment variable support (PORT, LOG_FORMAT, TEMPLATES_DIR, TEMPLATE_PROFILE_PATH, ENABLE_TEMPLATES_DIR)
  - [x] CLI flags (--templates-dir, --template-profile-path, --enable-templates-dir)
  - [x] Flag > env var > default priority chain

- [x] **CI/CD**
  - [x] Dagger pipeline (build, lint, test, image build/scan)
  - [x] GitHub Actions workflows (build-test, build-scan-image, lint-repo)
  - [x] Taskfile.yaml with build, test, lint, PR tasks
  - [x] Image registry automation (ghcr.io)

- [x] **Version endpoint** (GET /version)

#### Remaining

- [ ] **Advanced Parameter Validation**
  - [ ] JSON Schema validation
  - [ ] Cross-field validation
  - [ ] Length constraints (minLength, maxLength)

- [ ] **Backstage Integration**
  - [ ] Backstage scaffolder template example
  - [ ] Backstage action for template discovery

- [ ] **Observability**
  - [ ] Prometheus /metrics endpoint
  - [ ] Error rate tracking

- [ ] **Performance & Caching**
  - [ ] TTL-based cache invalidation
  - [ ] OCI module caching
  - [ ] Load testing & benchmarks

- [ ] **Dry-Run Mode**
  - [ ] Validation without execution
  - [ ] Parameter preview

---

### Phase 3: Production Ready
**Goal:** Hardened testing, authentication, Helm deployment
**Status:** Planned

- [ ] **Testing**
  - [ ] Full integration test suite (end-to-end API flows)
  - [ ] Load testing (k6 or similar)
  - [ ] Security testing
  - [ ] 80%+ code coverage

- [ ] **Authentication**
  - [ ] Backstage token validation
  - [ ] OIDC integration
  - [ ] API key authentication

- [ ] **Deployment**
  - [ ] Helm chart
  - [ ] Kubernetes manifests (Deployment, Service, ConfigMap)

---

## Current Status

**Version:** 0.4.1

**Phase 2 is the active phase.** Most observability, Backstage, CLI, and CI/CD items are done. Remaining work focuses on advanced validation, Prometheus metrics, caching, and Backstage scaffolder integration.

---

## Release Timeline

| Phase | Status | Version |
|-------|--------|---------|
| MVP | COMPLETE | 0.1.0 |
| Enhancement | IN PROGRESS | 0.2.0 - 0.4.1 |
| Production | Planned | 1.0.0 |

---

## Related Documentation

- [SPEC.md](./SPEC.md) - Technical specification
- [API_IMPLEMENTATION_SUMMARY.md](./API_IMPLEMENTATION_SUMMARY.md) - API reference
- [TESTING_GUIDE.md](./TESTING_GUIDE.md) - Testing guide
- [BACKSTAGE_COMPATIBILITY.md](./BACKSTAGE_COMPATIBILITY.md) - Backstage integration
- [KCL_INTEGRATION_SUMMARY.md](./KCL_INTEGRATION_SUMMARY.md) - KCL rendering docs
- [claim-template-schema.md](./claim-template-schema.md) - Template specification
- [openapi.yaml](./openapi.yaml) - OpenAPI definition
