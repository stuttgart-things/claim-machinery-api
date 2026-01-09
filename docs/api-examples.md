# API Examples

Complete examples for using the Claim Machinery API.

---

## 1. List All Claim Templates

Retrieve a list of all available claim templates.

### Request

```bash
curl -X GET http://localhost:8080/api/v1/claim-templates
```

### Response (200 OK)

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
        "tags": ["storage", "crossplane"],
        "labels": {
          "category": "storage"
        }
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

---

## 2. Get Template Details

Retrieve detailed information about a specific template including parameter schema.

### Request

```bash
curl -X GET http://localhost:8080/api/v1/claim-templates/volumeclaim
```

### Response (200 OK)

```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "ClaimTemplate",
  "metadata": {
    "name": "volumeclaim",
    "title": "Crossplane Volume Claim",
    "description": "Creates a persistent volume claim using Crossplane",
    "tags": ["storage", "crossplane", "kubernetes"],
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

---

## 3. Render a Claim (Order)

Render/execute a claim with custom parameters.

### Request

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "namespace": "production",
      "storage": "10Gi",
      "storageClassName": "fast-ssd"
    },
    "dryRun": false
  }'
```

### Response (200 OK)

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
  "output": "apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: pvc-production\n  namespace: production\nspec:\n  accessModes:\n    - ReadWriteOnce\n  storageClassName: fast-ssd\n  resources:\n    requests:\n      storage: 10Gi\n"
}
```

---

## 4. Dry-Run: Validate Without Execution

Test parameters without actually rendering the claim.

### Request

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "namespace": "prod",
      "storage": "10Gi",
      "storageClassName": "fast-ssd"
    },
    "dryRun": true
  }'
```

### Response (200 OK)

```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "OrderResponse",
  "metadata": {
    "orderId": "550e8400-e29b-41d4-a716-446655440001",
    "template": "volumeclaim",
    "createdAt": "2026-01-09T10:05:00Z"
  },
  "status": "success",
  "parameters": {
    "namespace": "prod",
    "storage": "10Gi",
    "storageClassName": "fast-ssd"
  },
  "output": "✓ All parameters validated successfully\n✓ Template requirements met\n✓ Ready to render"
}
```

---

## 5. Error: Invalid Parameters

Example of parameter validation error.

### Request

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "namespace": "invalid_NAMESPACE",
      "storage": "invalid",
      "storageClassName": "unknown-class"
    }
  }'
```

### Response (422 Unprocessable Entity)

```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Parameter validation failed",
    "details": {
      "parameters": [
        {
          "name": "namespace",
          "reason": "must match pattern: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
        },
        {
          "name": "storage",
          "reason": "invalid format: expected format like 10Gi"
        },
        {
          "name": "storageClassName",
          "reason": "must be one of: standard, fast-ssd, nvme"
        }
      ]
    }
  }
}
```

---

## 6. Error: Missing Required Parameter

### Request

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "storage": "10Gi"
    }
  }'
```

### Response (400 Bad Request)

```json
{
  "status": "error",
  "error": {
    "code": "MISSING_REQUIRED_PARAMETER",
    "message": "Required parameter is missing",
    "details": {
      "parameter": "namespace",
      "reason": "required parameter 'namespace' is missing"
    }
  }
}
```

---

## 7. Error: Template Not Found

### Request

```bash
curl -X GET http://localhost:8080/api/v1/claim-templates/nonexistent
```

### Response (404 Not Found)

```json
{
  "status": "error",
  "error": {
    "code": "TEMPLATE_NOT_FOUND",
    "message": "Requested template not found",
    "details": {
      "template": "nonexistent"
    }
  }
}
```

---

## 8. Search Templates

Filter templates by tag or search term.

### Request

```bash
curl -X GET "http://localhost:8080/api/v1/claim-templates?tag=storage&search=volume"
```

### Response (200 OK)

```json
{
  "apiVersion": "sthings.io/v1alpha1",
  "kind": "ClaimTemplateList",
  "items": [
    {
      "metadata": {
        "name": "volumeclaim",
        "title": "Crossplane Volume Claim",
        "tags": ["storage", "crossplane", "volume"],
        ...
      },
      ...
    }
  ]
}
```

---

## Client Library Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

const BASE_URL = 'http://localhost:8080/api/v1';

// List templates
async function listTemplates() {
  try {
    const response = await axios.get(`${BASE_URL}/claim-templates`);
    console.log(response.data);
  } catch (error) {
    console.error('Error:', error.response.data);
  }
}

// Get template details
async function getTemplate(name) {
  try {
    const response = await axios.get(`${BASE_URL}/claim-templates/${name}`);
    console.log(response.data);
  } catch (error) {
    console.error('Error:', error.response.data);
  }
}

// Order/render a claim
async function orderClaim(name, parameters, dryRun = false) {
  try {
    const response = await axios.post(
      `${BASE_URL}/claim-templates/${name}/order`,
      { parameters, dryRun }
    );
    console.log(response.data);
  } catch (error) {
    console.error('Error:', error.response.data);
  }
}

// Usage
listTemplates();
getTemplate('volumeclaim');
orderClaim('volumeclaim', {
  namespace: 'production',
  storage: '10Gi',
  storageClassName: 'fast-ssd'
});
```

### Python

```python
import requests
import json

BASE_URL = 'http://localhost:8080/api/v1'

def list_templates():
    """List all available templates"""
    response = requests.get(f'{BASE_URL}/claim-templates')
    return response.json()

def get_template(name):
    """Get template details"""
    response = requests.get(f'{BASE_URL}/claim-templates/{name}')
    return response.json()

def order_claim(name, parameters, dry_run=False):
    """Render/order a claim"""
    payload = {
        'parameters': parameters,
        'dryRun': dry_run
    }
    response = requests.post(
        f'{BASE_URL}/claim-templates/{name}/order',
        json=payload,
        headers={'Content-Type': 'application/json'}
    )
    return response.json()

# Usage
templates = list_templates()
print(json.dumps(templates, indent=2))

template = get_template('volumeclaim')
print(json.dumps(template, indent=2))

result = order_claim('volumeclaim', {
    'namespace': 'production',
    'storage': '10Gi',
    'storageClassName': 'fast-ssd'
})
print(json.dumps(result, indent=2))
```

### Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "http://localhost:8080/api/v1"

func listTemplates() {
	resp, err := http.Get(fmt.Sprintf("%s/claim-templates", baseURL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func orderClaim(name string, parameters map[string]interface{}) {
	payload := map[string]interface{}{
		"parameters": parameters,
		"dryRun":     false,
	}

	jsonBody, _ := json.Marshal(payload)
	resp, err := http.Post(
		fmt.Sprintf("%s/claim-templates/%s/order", baseURL, name),
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func main() {
	listTemplates()

	params := map[string]interface{}{
		"namespace":         "production",
		"storage":           "10Gi",
		"storageClassName":  "fast-ssd",
	}
	orderClaim("volumeclaim", params)
}
```

---

## Integration with Backstage

### Catalog Integration

Templates can be registered as Backstage components in your `catalog-info.yaml`:

```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: claim-machinery-api
  description: KCL-based Crossplane claim template service
  annotations:
    github.com/project-slug: stuttgart-things/claim-machinery-api
    backstage.io/source-location: url:https://github.com/stuttgart-things/claim-machinery-api
spec:
  type: service
  owner: platform-team
  lifecycle: production
  system: infrastructure
  endpoints:
    - name: API
      url: http://claim-machinery-api:8080/api/v1
```

### Template scaffolder integration

Use templates in Backstage Scaffolder:

```yaml
apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  name: create-volume-claim
  title: Create Volume Claim
spec:
  owner: platform-team
  type: resource
  parameters:
    - title: Volume Configuration
      properties:
        namespace:
          title: Namespace
          type: string
          default: default
        storage:
          title: Storage Size
          type: string
          default: 20Gi
  steps:
    - id: call-api
      name: Create Volume Claim
      action: fetch:curl
      input:
        url: http://claim-machinery-api:8080/api/v1/claim-templates/volumeclaim/order
        method: POST
        body: |
          {
            "parameters": {
              "namespace": "${{ parameters.namespace }}",
              "storage": "${{ parameters.storage }}",
              "storageClassName": "standard"
            }
          }
```
