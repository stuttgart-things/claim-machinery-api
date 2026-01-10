# [FEATURE] Implement REST API Endpoints for Claim Templates

## Affected Spec Section
- **Section 3.3**: API Endpoints (List, Get, Order)
- **Section 11**: Deployment & Configuration
- **Section 7**: Error Handling

---

## Requirements

### API Endpoints to Implement:

#### 1. List Templates - GET /api/v1/claim-templates
- [x] Route handler for GET requests
- [x] Load all templates from internal/claimtemplate/testdata
- [x] Return ClaimTemplateList JSON response
- [x] Include proper HTTP status codes (200 OK)
- [x] Response format per SPEC.md section 3.3.1

#### 2. Get Template Detail - GET /api/v1/claim-templates/{name}
- [x] Route handler with path parameter {name}
- [x] Load specific template by name
- [x] Return full ClaimTemplate with all metadata and parameters
- [x] Return 404 if template not found
- [x] Include parameter definitions with validation rules
- [x] Response format per SPEC.md section 3.3.2

#### 3. Render Template - POST /api/v1/claim-templates/{name}/order
- [x] Route handler for POST requests with JSON body
- [x] Accept OrderRequest with parameters
- [x] Validate parameters before rendering
- [x] Execute KCL rendering via existing RenderKCLFromOCI function
- [x] Return OrderResponse with rendered YAML
- [x] Return 400 for validation errors
- [x] Return 500 for execution errors
- [x] Response format per SPEC.md section 3.3.3

### Infrastructure Components:

#### HTTP Server Setup
- [x] HTTP router using gorilla/mux
- [x] Server initialization on configurable port (default :8080)
- [x] Graceful shutdown handling
- [x] Request timeout configuration

#### Middleware
- [x] JSON content-type validation
- [x] CORS headers for browser access
- [x] Request/response logging (with context ID)
- [x] Error handling middleware
- [x] Health check endpoint (/health)

#### Error Handling
- [x] Standardized error response format (per SPEC.md section 7)
- [x] Proper HTTP status codes (200, 400, 404, 500, etc.)
- [x] Detailed error messages in logs
- [x] Request IDs for tracing

---

## Acceptance Criteria

- [x] All three API endpoints implemented and tested
- [x] HTTP router configured with proper routes
- [x] Middleware components in place (CORS, logging, error handling)
- [x] Integration tests for all endpoints (happy path + error cases)
- [x] Unit tests for handlers (80%+ coverage)
- [x] Error handling with proper status codes
- [x] Health check endpoint working
- [x] Request/response logging configured
- [x] Code review passed
- [x] Documentation updated (README.md, api-examples.md)
- [x] All existing tests still passing
- [x] Manual testing completed (curl/Postman)

---

## Technical Implementation Details

### Files to Create/Modify:

#### New Files:
- `internal/api/server.go` - HTTP server setup and initialization
- `internal/api/handlers.go` - Route handlers for all three endpoints
- `internal/api/middleware.go` - CORS, logging, error handling middleware
- `internal/api/types.go` - OrderRequest, OrderResponse, error types (if not in claimtemplate.go)
- `internal/api/handlers_test.go` - Comprehensive handler tests

#### Existing Files to Modify:
- `main.go` - Add HTTP server startup
- `go.mod` - Add gorilla/mux dependency
- `SPEC.md` - Cross-reference to actual implementation
- `README.md` - Add API documentation and example usage

### Dependencies Required:
```go
github.com/gorilla/mux  // HTTP router
```

### Architecture:

```
main()
  ├── app.LoadAllTemplates() [ALREADY DONE]
  └── api.NewServer()
      ├── RegisterRoutes()
      │   ├── GET /health → healthCheck
      │   ├── GET /api/v1/claim-templates → listTemplates
      │   ├── GET /api/v1/claim-templates/{name} → getTemplate
      │   └── POST /api/v1/claim-templates/{name}/order → orderClaim
      ├── ApplyMiddleware()
      │   ├── CORS middleware
      │   ├── Logging middleware
      │   └── Error handling middleware
      └── Listen and Serve
```

---

## Implementation Plan

### Step 1: Server Setup
1. Create `internal/api/server.go` with `Server` struct
2. Implement `NewServer()` to initialize router with gorilla/mux
3. Add `Start()` and `Stop()` methods for lifecycle management
4. Update `main.go` to start server instead of console printing

### Step 2: Route Handlers
1. Implement `listTemplates()` handler
2. Implement `getTemplate()` handler with path parameter extraction
3. Implement `orderClaim()` handler with request body parsing
4. Add `healthCheck()` handler

### Step 3: Middleware
1. Create CORS middleware to allow browser requests
2. Create request logging middleware with context IDs
3. Create error handling middleware with standardized responses

### Step 4: Testing
1. Write handler tests (mocking templates, testing all routes)
2. Write integration tests (full request/response cycle)
3. Test error cases (404, 400, 500)
4. Verify 80%+ code coverage

### Step 5: Documentation
1. Update README.md with API server information
2. Add curl examples to api-examples.md
3. Document configuration options
4. Add deployment instructions

---

## Testing Strategy

### Unit Tests (handlers_test.go):
```
✓ TestListTemplates_Success
✓ TestListTemplates_Empty
✓ TestGetTemplate_Found
✓ TestGetTemplate_NotFound
✓ TestOrderClaim_Success
✓ TestOrderClaim_InvalidParameters
✓ TestOrderClaim_RenderError
✓ TestHealthCheck
```

### Integration Tests:
```
✓ Full request cycle: GET /claim-templates
✓ Full request cycle: GET /claim-templates/{name}
✓ Full request cycle: POST /claim-templates/{name}/order with valid data
✓ Full request cycle: POST with invalid parameter types
✓ Error propagation and response format
```

### Manual Testing:
```bash
# List templates
curl http://localhost:8080/api/v1/claim-templates

# Get template detail
curl http://localhost:8080/api/v1/claim-templates/volumeclaim

# Order claim (render)
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "50Gi"}}'

# Health check
curl http://localhost:8080/health
```

---

## Expected Output Examples

### GET /api/v1/claim-templates (200 OK)
```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "ClaimTemplateList",
  "items": [
    {
      "apiVersion": "templates.claim-machinery.io/v1alpha1",
      "kind": "ClaimTemplate",
      "metadata": {
        "name": "volumeclaim",
        "title": "Crossplane Volume Claim",
        "description": "Create a volume claim for storage provisioning"
      },
      "spec": {
        "type": "template",
        "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
        "tag": "0.1.1",
        "parameters": [...]
      }
    },
    {...}
  ]
}
```

### GET /api/v1/claim-templates/{name} (200 OK)
```json
{
  "apiVersion": "templates.claim-machinery.io/v1alpha1",
  "kind": "ClaimTemplate",
  "metadata": {...},
  "spec": {
    "type": "template",
    "source": "oci://ghcr.io/...",
    "parameters": [
      {
        "name": "namespace",
        "title": "Kubernetes Namespace",
        "type": "string",
        "required": true,
        "default": "default"
      }
    ]
  }
}
```

### POST /api/v1/claim-templates/{name}/order (200 OK)
```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "OrderResponse",
  "metadata": {
    "name": "volumeclaim-order-123",
    "timestamp": "2026-01-10T12:34:56Z"
  },
  "rendered": {
    "apiVersion": "resources.stuttgart-things.com/v1alpha1",
    "kind": "VolumeClaim",
    "metadata": {
      "name": "demo-pvc",
      "namespace": "production"
    },
    "spec": {
      "storage": "50Gi",
      "storageClassName": "standard"
    }
  }
}
```

### Error Response (400 Bad Request)
```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "Error",
  "metadata": {
    "name": "validation-error",
    "timestamp": "2026-01-10T12:34:56Z"
  },
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid parameter: namespace is required",
    "details": [
      {
        "field": "namespace",
        "issue": "required field missing"
      }
    ]
  }
}
```

---

## Related Issues/PRs

- Depends on: #KCL-RENDER (KCL rendering implementation) ✅ DONE
- Related: SPEC.md sections 3.3, 7, 11

---

## Complexity Assessment

**Estimated Complexity:** Medium (2-5 days)

- Server setup: 1 day
- Route handlers: 1 day
- Middleware: 1 day
- Testing & Documentation: 1-2 days

---

## Additional Notes

### Configuration Options (for future):
- Port configuration (env var: `API_PORT`, default: 8080)
- Template directory (env var: `TEMPLATES_DIR`)
- CORS origin whitelist
- Request timeout settings
- Logging level

### Performance Considerations:
- Cache template list in memory (loaded once)
- Implement TTL-based cache invalidation
- Consider rate limiting per template

### Future Enhancements:
- WebSocket support for streaming renders
- Async rendering for long-running KCL operations
- OpenAPI/Swagger documentation
- GraphQL endpoint variant

---

## Success Criteria

✅ API responds to all three endpoints with correct HTTP status codes
✅ Request/response format matches SPEC.md
✅ Parameter validation prevents invalid requests
✅ Error handling returns proper error responses
✅ Middleware logs all requests with context IDs
✅ All tests passing with 80%+ coverage
✅ Manual curl testing succeeds
✅ Code review approved

---

**Priority:** HIGH (blocks Phase 1 completion)
**Effort:** 2-5 days
**Owner:** @user
**Target Date:** 2026-01-23 (End of Sprint 1)
