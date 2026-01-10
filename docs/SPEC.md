# Claim Machinery API - Technical Specification

**Version:** 1.0.0
**Status:** Draft
**Date:** 2026-01-09

---

## 1. Overview

Claim Machinery API is a Go-based microservice that enables discovering, managing, and rendering KCL-based Crossplane claim templates. It provides a Backstage-compatible REST API for template discovery and claim rendering.

### 1.1 Goals

- [ ] Provide REST API for claim template discovery
- [ ] Enable schema-based parameter validation
- [ ] Support KCL rendering with OCI module loading
- [ ] Backstage Software Catalog integration
- [ ] Dry-run capability for claim validation

### 1.2 Out of Scope

- Claim storage/persistence (stateless API)
- Authentication/Authorization (handled by API Gateway)
- Crossplane resource management (templates only)

---

## 2. Architecture

### 2.1 High-Level Design

```
┌─────────────────────┐
│   Client            │
│  (Backstage/Web)    │
└──────────┬──────────┘
           │
    ┌──────▼──────┐
    │   HTTP API  │
    │  Server     │
    └──────┬──────┘
           │
    ┌──────▼──────────────────┐
    │  Handler Layer          │
    │  - Validation           │
    │  - Parameter processing │
    └──────┬──────────────────┘
           │
    ┌──────▼──────────────────┐
    │  KCL Service            │
    │  - Template loading     │
    │  - KCL execution        │
    │  - Output processing    │
    └──────┬──────────────────┘
           │
    ┌──────▼──────────────────┐
    │  External Services      │
    │  - OCI Registry (pull)  │
    │  - KCL CLI              │
    └─────────────────────────┘
```

### 2.2 Components

| Component | Responsibility | Key Tasks |
|-----------|----------------|-----------|
| **HTTP Server** | Request handling | Routing, CORS, health checks |
| **Handler Layer** | API logic | Parameter validation, response formatting |
| **Template Service** | Template management | Loading, caching, schema parsing |
| **KCL Service** | KCL execution | Running KCL CLI, parameter injection |
| **Validator** | Input validation | Type checking, enum validation, pattern matching |

---

## 3. API Specification

### 3.1 Base URL

```
http://localhost:8080/api/v1
```

### 3.2 Content-Type

All requests and responses use `application/json`.

---

## 3.3 Endpoints

### 3.3.1 GET /claim-templates

List all available claim templates.

**Query Parameters:**
- `tag` (optional) - Filter by template tag
- `search` (optional) - Search by name or title

**Response (200 OK):**
```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "ClaimTemplateList",
  "items": [
    {
      "apiVersion": "sthings.io/v1alpha1",
      "kind": "ClaimTemplate",
      "metadata": {
        "name": "volumeclaim",
        "title": "Crossplane Volume Claim",
        "description": "Creates a persistent volume claim",
        "tags": ["storage", "crossplane", "kubernetes"],
        "labels": {
          "category": "storage"
        }
      },
      "spec": {
        "type": "volumeclaim",
        "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
        "tag": "0.1.1",
        "parameters": [
          {
            "name": "namespace",
            "title": "Namespace",
            "type": "string",
            "default": "default",
            "required": true
          }
        ]
      }
    }
  ]
}
```

**Error Responses:**
- `500 Internal Server Error` - Failed to load templates

---

### 3.3.2 GET /claim-templates/{name}

Get template details including schema and parameters.

**Path Parameters:**
- `name` (required) - Template name

**Response (200 OK):**
```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "ClaimTemplate",
  "metadata": {
    "name": "volumeclaim",
    "title": "Crossplane Volume Claim",
    "description": "Creates a persistent volume claim using Crossplane",
    "tags": ["storage", "crossplane"],
    "labels": {
      "category": "storage",
      "managed-by": "claim-machinery"
    }
  },
  "spec": {
    "type": "volumeclaim",
    "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
    "tag": "0.1.1",
    "parameters": [
      {
        "name": "namespace",
        "title": "Kubernetes Namespace",
        "description": "Target namespace for the volume claim",
        "type": "string",
        "default": "default",
        "required": true,
        "pattern": "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
        "minLength": 1,
        "maxLength": 63,
        "ui:options": {
          "widget": "text",
          "placeholder": "production"
        }
      },
      {
        "name": "storage",
        "title": "Storage Size",
        "description": "Storage capacity (e.g., 10Gi, 100Mi)",
        "type": "string",
        "default": "20Gi",
        "required": true,
        "pattern": "^[0-9]+(Gi|Mi|Ti|G|M|T)$",
        "ui:options": {
          "widget": "text",
          "placeholder": "20Gi"
        }
      },
      {
        "name": "storageClassName",
        "title": "Storage Class",
        "description": "Storage class for provisioning",
        "type": "string",
        "default": "standard",
        "required": true,
        "enum": ["standard", "fast-ssd", "nvme"],
        "ui:options": {
          "widget": "select"
        }
      }
    ]
  }
}
```

**Error Responses:**
- `404 Not Found` - Template not found

---

### 3.3.3 POST /claim-templates/{name}/order

Render a claim based on template and parameters.

**Path Parameters:**
- `name` (required) - Template name

**Request Body:**
```json
{
  "parameters": {
    "namespace": "production",
    "storage": "10Gi",
    "storageClassName": "fast-ssd"
  },
  "dryRun": false
}
```

**Response (200 OK):**
```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "OrderResponse",
  "metadata": {
    "orderId": "550e8400-e29b-41d4-a716-446655440000",
    "template": "volumeclaim",
    "createdAt": "2026-01-09T10:00:00Z"
  },
  "status": "success",
  "parameters": {
    "namespace": "production",
    "storage": "10Gi",
    "storageClassName": "fast-ssd"
  },
  "output": "apiVersion: storage.k8s.io/v1\nkind: PersistentVolumeClaim\n..."
}
```

**Error Responses:**
- `400 Bad Request` - Invalid parameters
- `404 Not Found` - Template not found
- `422 Unprocessable Entity` - Parameter validation failed
- `500 Internal Server Error` - KCL execution error

---

### 3.4 Error Response Format

```json
{
  "status": "error",
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Parameter validation failed",
    "details": {
      "parameters": [
        {
          "name": "namespace",
          "reason": "must match pattern: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"
        }
      ]
    }
  }
}
```

---

## 4. Data Models

### 4.1 ClaimTemplate

```go
type ClaimTemplate struct {
    APIVersion string                `json:"apiVersion"`
    Kind       string                `json:"kind"`
    Metadata   ClaimTemplateMetadata `json:"metadata"`
    Spec       ClaimTemplateSpec     `json:"spec"`
}

type ClaimTemplateMetadata struct {
    Name        string            `json:"name"`
    Title       string            `json:"title,omitempty"`
    Description string            `json:"description,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
}

type ClaimTemplateSpec struct {
    Type       string      `json:"type"`
    Source     string      `json:"source"` // OCI URL
    Tag        string      `json:"tag,omitempty"`
    Parameters []Parameter `json:"parameters"`
}

type Parameter struct {
    Name        string      `json:"name"`
    Title       string      `json:"title"`
    Description string      `json:"description,omitempty"`
    Type        string      `json:"type"` // string, boolean, array, number
    Default     interface{} `json:"default,omitempty"`
    Required    bool        `json:"required,omitempty"`
    Enum        []string    `json:"enum,omitempty"`
    Pattern     string      `json:"pattern,omitempty"`
    MinLength   *int        `json:"minLength,omitempty"`
    MaxLength   *int        `json:"maxLength,omitempty"`
    UIOptions   *UIOptions  `json:"ui:options,omitempty"`
}

type UIOptions struct {
    Widget      string `json:"widget,omitempty"`
    Placeholder string `json:"placeholder,omitempty"`
    Rows        int    `json:"rows,omitempty"`
}
```

### 4.2 OrderRequest

```go
type OrderRequest struct {
    Parameters map[string]interface{} `json:"parameters"`
    DryRun     bool                   `json:"dryRun,omitempty"`
}
```

### 4.3 OrderResponse

```go
type OrderResponse struct {
    APIVersion string                 `json:"apiVersion"`
    Kind       string                 `json:"kind"`
    Metadata   OrderMetadata          `json:"metadata"`
    Status     string                 `json:"status"` // success, error
    Parameters map[string]interface{} `json:"parameters,omitempty"`
    Output     string                 `json:"output,omitempty"` // Rendered YAML
    Error      string                 `json:"error,omitempty"`
}

type OrderMetadata struct {
    OrderID   string    `json:"orderId"`
    Template  string    `json:"template"`
    CreatedAt time.Time `json:"createdAt"`
}
```

---

## 5. KCL Integration

### 5.1 KCL Execution

```go
// Execute KCL with parameters
cmd := exec.Command("kcl", "run", "oci://module", "-D", "param=value")
output, err := cmd.Output()
```

### 5.2 Parameter Injection

Parameters are passed via `-D` flags:

```bash
kcl run oci://ghcr.io/module \
  -D namespace=production \
  -D storage=10Gi \
  -D storageClassName=fast-ssd
```

---

## 6. Validation Rules

### 6.1 Parameter Validation

- **Required**: Check if required parameters are provided
- **Type**: Validate parameter type (string, boolean, array, number)
- **Pattern**: Regex validation
- **Length**: Min/max string length
- **Enum**: Check if value is in allowed list
- **Minimum/Maximum**: Numeric bounds

### 6.2 Validation Error Response

```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_FAILED",
    "details": {
      "namespace": "must match pattern: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$",
      "storage": "invalid format: expected format like 10Gi"
    }
  }
}
```

---

## 7. Security

### 7.1 Input Validation

- All parameters validated before KCL execution
- Pattern matching for string parameters
- Enum validation for restricted values
- Type checking

### 7.2 KCL Execution Safety

- Timeout for KCL execution (30 seconds)
- Resource limits enforced
- Dry-run mode for testing

### 7.3 CORS Configuration

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

---

## 8. Configuration

### 8.1 Environment Variables

```env
# Server
SERVER_PORT=8080
SERVER_ADDR=0.0.0.0
ENV=production

# KCL
KCL_TIMEOUT=30s
KCL_PATH=/usr/local/bin/kcl

# Template Loading
TEMPLATE_PATH=./templates
TEMPLATE_CACHE_TTL=5m

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### 8.2 Configuration File (config.yaml)

```yaml
server:
  port: 8080
  addr: 0.0.0.0
  timeout: 30s

kcl:
  timeout: 30s
  path: /usr/local/bin/kcl

templates:
  path: ./templates
  cacheTTL: 5m
  loadFromOCI: true

logging:
  level: info
  format: json
```

---

## 9. Performance & Scalability

### 9.1 Performance Targets

- **Response Time**: < 500ms (p99) for template list
- **Claim Rendering**: < 5s (p99) including KCL execution
- **Throughput**: 100+ requests/second

### 9.2 Caching

- **Template Cache**: 5-minute TTL for loaded templates
- **Schema Cache**: Persistent in-memory cache
- **OCI Module Cache**: Local cache for downloaded modules

### 9.3 Horizontal Scaling

- Stateless API design
- Load balancing ready
- Shared template storage (filesystem or OCI)

---

## 10. Testing Strategy

### 10.1 Unit Tests

- Parameter validation tests
- KCL execution mocking
- Error handling tests
- **Coverage Goal**: 80%+

### 10.2 Integration Tests

- API endpoint tests
- KCL integration tests
- Template loading tests

### 10.3 Load Testing

- k6 or Apache JMeter
- Target: 100+ concurrent users

---

## 11. Deployment

### 11.1 Docker Image

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api main.go

FROM alpine:latest
RUN apk add --no-cache curl ca-certificates
RUN curl -fsSL https://kcl-lang.io/script/install-cli.sh | sh
COPY --from=builder /app/api /usr/local/bin/
EXPOSE 8080
CMD ["api"]
```

### 11.2 Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: claim-machinery-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: claim-machinery-api
  template:
    metadata:
      labels:
        app: claim-machinery-api
    spec:
      containers:
      - name: api
        image: claim-machinery-api:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: SERVER_PORT
          value: "8080"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
```

### 11.3 Monitoring

- **Prometheus**: Metrics export
- **Grafana**: Dashboards
- **Structured Logging**: JSON format for aggregation

---

## 12. Roadmap

### Phase 1: MVP (Sprint 1-2)
- [x] REST API structure
- [x] Template discovery endpoints
- [x] KCL integration
- [ ] Basic parameter validation

### Phase 2: Enhancement (Sprint 3-4)
- [ ] Advanced validation (patterns, custom rules)
- [ ] Template caching
- [ ] Error handling refinement
- [ ] Backstage integration

### Phase 3: Production (Sprint 5-6)
- [ ] Comprehensive testing
- [ ] Monitoring & logging
- [ ] Performance optimization
- [ ] Security audit

---

## 13. Change History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-09 | Initial specification |

---

## Appendix: References

- [Backstage Software Catalog](https://backstage.io/docs/features/software-catalog)
- [KCL Language Documentation](https://kcl-lang.io/docs)
- [Crossplane Documentation](https://docs.crossplane.io)
- [Go Best Practices](https://golang.org/doc/effective_go)
