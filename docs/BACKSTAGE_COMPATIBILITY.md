# Backstage Compatibility

## Overview

The Claim Machinery API is **fully compatible** with Backstage Custom Field Extensions and can be integrated into Backstage Scaffolder templates as a backend action.

## Compatibility Status

### ✅ Compatible Features

| Feature | Implementation | Use Case |
|---------|-----------------|----------|
| **JSON Schema Support** | `type`, `enum`, `pattern`, `required` | Form field validation |
| **Parameter Metadata** | `title`, `description` | UI rendering and help text |
| **Validation Rules** | `pattern`, `minLength`, `maxLength`, `default` | Client-side and server-side validation |
| **REST API** | Standard HTTP/JSON endpoints | Backend action integration |
| **Templating** | KCL-based rendering with parameter injection | Dynamic resource generation |
| **Response Format** | Structured JSON with metadata | Scaffolder step compatibility |

## API Endpoints for Backstage

### List Available Templates
```http
GET /api/v1/claim-templates
```

Returns all available claim templates with their parameters and metadata.

**Response:**
```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "ClaimTemplateList",
  "items": [
    {
      "apiVersion": "sthings.io/v1alpha1",
      "kind": "ClaimTemplate",
      "metadata": {
        "name": "volumeclaim",
        "title": "Crossplane Volume Claim",
        "description": "Creates a persistent volume claim via Crossplane",
        "tags": ["storage", "crossplane"]
      },
      "spec": {
        "type": "volumeclaim",
        "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
        "tag": "0.1.1",
        "parameters": [...]
      }
    }
  ]
}
```

### Get Template Details
```http
GET /api/v1/claim-templates/{name}
```

Returns a specific template with full parameter definitions.

**Response:**
```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "ClaimTemplate",
  "metadata": {
    "name": "volumeclaim",
    "title": "Crossplane Volume Claim",
    "description": "Creates a persistent volume claim via Crossplane",
    "tags": ["storage", "crossplane"]
  },
  "spec": {
    "type": "volumeclaim",
    "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
    "tag": "0.1.1",
    "parameters": [
      {
        "name": "namespace",
        "title": "Namespace",
        "description": "Kubernetes namespace",
        "type": "string",
        "required": true,
        "default": "default",
        "pattern": "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
      },
      {
        "name": "storage",
        "title": "Storage Size",
        "type": "string",
        "required": true,
        "default": "20Gi",
        "pattern": "^[0-9]+(Gi|Mi|Ti)$"
      },
      {
        "name": "storageClassName",
        "title": "Storage Class",
        "type": "string",
        "default": "standard",
        "enum": ["standard", "fast", "ssd"]
      }
    ]
  }
}
```

### Render/Order Claim
```http
POST /api/v1/claim-templates/{name}/order
Content-Type: application/json

{
  "parameters": {
    "namespace": "production",
    "storage": "100Gi",
    "storageClassName": "ssd"
  }
}
```

Returns the rendered resource definition.

**Response:**
```json
{
  "apiVersion": "v1",
  "kind": "ClaimTemplate",
  "metadata": {
    "name": "volumeclaim-prod",
    "namespace": "production"
  },
  "rendered": "---\napiVersion: v1\nkind: PersistentVolumeClaim\n..."
}
```

## Backstage Integration Examples

### Example 1: Scaffolder Template with Custom Field Extension

```yaml
apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  name: create-storage-claim
  title: Create Persistent Volume Claim
spec:
  owner: platform-team
  type: service

  parameters:
    - title: Select Storage Type
      required:
        - storageType
      properties:
        storageType:
          title: Storage Type
          type: string
          enum:
            - volumeclaim
            - postgresql
          default: volumeclaim

  steps:
    - id: fetch-template
      name: Fetch Claim Template Specification
      action: http:backstage:request
      input:
        method: GET
        path: /api/v1/claim-templates/${{ parameters.storageType }}
        baseUrl: http://claim-machinery-api:8080

    - id: render-claim
      name: Render Claim with Parameters
      action: http:backstage:request
      input:
        method: POST
        path: /api/v1/claim-templates/${{ parameters.storageType }}/order
        baseUrl: http://claim-machinery-api:8080
        body:
          parameters:
            namespace: ${{ parameters.namespace }}
            storage: ${{ parameters.storageSize }}

    - id: publish
      name: Publish Resource
      action: publish:github
      input:
        repoUrl: github.com?owner=org&repo=claims
        targetPath: claims/${{ parameters.storageType }}
        values:
          rendered: ${{ steps['render-claim'].body.rendered }}
```

### Example 2: Custom Field Extension

Create a custom Backstage field extension that uses the API:

```typescript
// backstage-plugin/src/fields/ClaimTemplateField.tsx
import React from 'react';
import { useApi } from '@backstage/core-plugin-api';

interface ClaimTemplateFieldProps {
  templateName: string;
  onChange: (parameters: Record<string, any>) => void;
}

export const ClaimTemplateField: React.FC<ClaimTemplateFieldProps> = ({
  templateName,
  onChange,
}) => {
  const [template, setTemplate] = React.useState<any>(null);

  React.useEffect(() => {
    // Fetch template from claim-machinery API
    fetch(`http://claim-machinery-api:8080/api/v1/claim-templates/${templateName}`)
      .then(r => r.json())
      .then(data => setTemplate(data))
      .catch(err => console.error('Failed to fetch template', err));
  }, [templateName]);

  if (!template) return <div>Loading...</div>;

  return (
    <div>
      <h3>{template.metadata.title}</h3>
      <p>{template.metadata.description}</p>

      {template.spec.parameters.map((param: any) => (
        <ParameterInput
          key={param.name}
          parameter={param}
          onChange={(value) => onChange({ [param.name]: value })}
        />
      ))}
    </div>
  );
};
```

## Parameter Type Mappings

The API supports the following parameter types that map to Backstage form fields:

| Type | Backstage Field | Validation |
|------|-----------------|-----------|
| `string` | Text Input | `pattern`, `minLength`, `maxLength`, `enum` |
| `boolean` | Checkbox | None |
| `number` | Number Input | Min/max via pattern |
| `array` | Multi-select | `enum` values |

## Validation Support

### Pattern Validation

Parameters can include regex patterns for validation:

```json
{
  "name": "namespace",
  "type": "string",
  "pattern": "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
}
```

### Enum Validation

Restrict parameters to predefined values:

```json
{
  "name": "storageClassName",
  "type": "string",
  "enum": ["standard", "fast", "ssd"]
}
```

### Required Fields

Mark parameters as required:

```json
{
  "name": "namespace",
  "type": "string",
  "required": true
}
```

### Default Values

Provide sensible defaults:

```json
{
  "name": "namespace",
  "type": "string",
  "default": "default"
}
```

## Integration Checklist

- [x] REST API with standard HTTP methods
- [x] JSON request/response format
- [x] Parameter type definitions
- [x] Validation rules (pattern, enum, required)
- [x] Metadata and help text
- [x] Default values
- [ ] OpenAPI/Swagger documentation (Phase 2)
- [ ] Prometheus metrics endpoint (Phase 2)
- [ ] Request correlation IDs (Phase 2)

## Future Enhancements

### Phase 2 Roadmap

1. **OpenAPI Specification**
   - Auto-generated `/api/openapi.json` endpoint
   - Machine-readable API documentation
   - Better IDE support and client generation

2. **Advanced Validation**
   - JSON Schema validation
   - Cross-field validation
   - Async validation hooks

3. **Observability**
   - Prometheus `/metrics` endpoint
   - Request tracing with correlation IDs
   - Structured logging for debugging

4. **Backend Authentication**
   - Backstage token validation
   - OIDC integration
   - Service account support

## Testing Backstage Integration

### Health Check
```bash
curl http://localhost:8080/health
```

### List Templates
```bash
curl http://localhost:8080/api/v1/claim-templates | jq .
```

### Get Template
```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim | jq .
```

### Render Claim
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "namespace": "backstage-test",
      "storage": "50Gi",
      "storageClassName": "ssd"
    }
  }' | jq .
```

## See Also

- [API_IMPLEMENTATION_SUMMARY.md](API_IMPLEMENTATION_SUMMARY.md) - Complete API reference
- [TESTING_GUIDE.md](TESTING_GUIDE.md) - API testing examples
- [SPEC.md](SPEC.md) - Technical specification
- [ROADMAP.md](ROADMAP.md) - Development roadmap
