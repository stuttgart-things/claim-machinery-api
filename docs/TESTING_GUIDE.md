# API Testing Guide

## Quick Start

### 1. Start the Server
```bash
go run main.go
```

Expected output:
```
🚀 Claim Machinery API starting
✓ API server listening on http://localhost:8080

📋 Available endpoints:
  GET  /health                                    - Health check
  GET  /api/v1/claim-templates                    - List templates
  GET  /api/v1/claim-templates/{name}             - Get template details
  POST /api/v1/claim-templates/{name}/order       - Render template
```

---

## Testing All Endpoints

### 1️⃣ Health Check
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{"status":"healthy","timestamp":"2026-01-10T14:05:26Z"}
```

---

### 2️⃣ List All Templates
```bash
curl http://localhost:8080/api/v1/claim-templates
```

**Response (formatted):**
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
        "description": "..."
      },
      "spec": {
        "type": "template",
        "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
        "tag": "0.1.1",
        "parameters": [...]
      }
    },
    {
      "metadata": {
        "name": "postgresql",
        "title": "PostgreSQL Database Claim"
      },
      ...
    }
  ]
}
```

**Pretty print with Python:**
```bash
curl http://localhost:8080/api/v1/claim-templates | python3 -m json.tool
```

**Count templates:**
```bash
curl -s http://localhost:8080/api/v1/claim-templates | grep -o '"name"' | wc -l
```

---

### 3️⃣ Get Single Template Details

#### Get volumeclaim template:
```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim
```

**Response:**
```json
{
  "apiVersion": "templates.claim-machinery.io/v1alpha1",
  "kind": "ClaimTemplate",
  "metadata": {
    "name": "volumeclaim",
    "title": "Crossplane Volume Claim"
  },
  "spec": {
    "type": "template",
    "source": "oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim",
    "tag": "0.1.1",
    "parameters": [
      {
        "name": "templateName",
        "title": "Template Name",
        "type": "string",
        "required": false,
        "default": "demo"
      },
      {
        "name": "namespace",
        "title": "Kubernetes Namespace",
        "type": "string",
        "required": true,
        "default": "default"
      },
      {
        "name": "storage",
        "title": "Storage Size",
        "type": "string",
        "required": true,
        "default": "20Gi"
      },
      {
        "name": "storageClassName",
        "type": "string",
        "required": false,
        "default": "standard"
      }
    ]
  }
}
```

#### Get postgresql template:
```bash
curl http://localhost:8080/api/v1/claim-templates/postgresql
```

#### 404 Error - Template not found:
```bash
curl http://localhost:8080/api/v1/claim-templates/nonexistent
```

**Response (404):**
```json
{"error":"template not found"}
```

---

### 4️⃣ Render Template (POST /order)

#### Basic rendering with default parameters:
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### Rendering with custom parameters:
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "100Gi", "storageClassName": "fast"}}'
```

**Response:**
```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "OrderResponse",
  "metadata": {
    "name": "volumeclaim-order-20260110140526",
    "timestamp": "2026-01-10T14:05:26Z"
  },
  "rendered": "apiVersion: resources.stuttgart-things.com/v1alpha1\nkind: VolumeClaim\n..."
}
```

#### Render postgresql template:
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/postgresql/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "databases", "databaseName": "mydb"}}'
```

#### 404 Error - Template not found:
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/nonexistent/order \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### 400 Error - Invalid JSON:
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

**Response (400):**
```json
{"error":"invalid request body"}
```

---

## Advanced Testing

### Using a Script

**test-api.sh:**
```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

echo "🧪 Testing Claim Machinery API"
echo ""

echo "1️⃣  Health Check"
curl -s "$BASE_URL/health" | python3 -m json.tool
echo ""

echo "2️⃣  List Templates"
curl -s "$BASE_URL/api/v1/claim-templates" | python3 -m json.tool | head -40
echo ""

echo "3️⃣  Get volumeclaim Template"
curl -s "$BASE_URL/api/v1/claim-templates/volumeclaim" | python3 -m json.tool | head -30
echo ""

echo "4️⃣  Render volumeclaim (default params)"
curl -s -X POST "$BASE_URL/api/v1/claim-templates/volumeclaim/order" \
  -H "Content-Type: application/json" \
  -d '{}' | python3 -m json.tool | head -30
echo ""

echo "5️⃣  Render volumeclaim (custom params)"
curl -s -X POST "$BASE_URL/api/v1/claim-templates/volumeclaim/order" \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "prod", "storage": "50Gi"}}' | python3 -m json.tool | head -30
echo ""

echo "6️⃣  Test 404 Error"
curl -s "$BASE_URL/api/v1/claim-templates/nonexistent" | python3 -m json.tool
echo ""

echo "✓ API Testing Complete"
```

**Run it:**
```bash
chmod +x test-api.sh
./test-api.sh
```

---

### Using Postman

**Import collection:**
1. Create new collection "Claim Machinery API"
2. Add requests:

**Request 1: Health**
- Method: GET
- URL: `{{base_url}}/health`

**Request 2: List Templates**
- Method: GET
- URL: `{{base_url}}/api/v1/claim-templates`

**Request 3: Get Template**
- Method: GET
- URL: `{{base_url}}/api/v1/claim-templates/{{template_name}}`
- Params: `template_name = volumeclaim`

**Request 4: Order Claim**
- Method: POST
- URL: `{{base_url}}/api/v1/claim-templates/{{template_name}}/order`
- Headers: `Content-Type: application/json`
- Body (JSON):
```json
{
  "parameters": {
    "namespace": "production",
    "storage": "100Gi"
  }
}
```

---

## Using httpie (Better than curl)

If you have `httpie` installed:

```bash
# Health check
http GET localhost:8080/health

# List templates
http GET localhost:8080/api/v1/claim-templates

# Get single template
http GET localhost:8080/api/v1/claim-templates/volumeclaim

# Render template
http POST localhost:8080/api/v1/claim-templates/volumeclaim/order \
  parameters:='{"namespace":"prod","storage":"50Gi"}'
```

---

## Monitoring & Debugging

### Check Server Logs
The server outputs:
- 📨 Request log: `📨 GET /api/v1/claim-templates`
- ✓ Completion log: `✓ GET /api/v1/claim-templates completed in 15ms`

### Use verbose curl:
```bash
curl -v http://localhost:8080/health
```

Shows:
- Request headers
- Response headers (HTTP status, Content-Type, etc.)
- Response body

### Check server port is listening:
```bash
# On Linux/Mac
lsof -i :8080
# or
netstat -an | grep 8080

# On any OS
curl -i http://localhost:8080/health
```

### Check logs in another terminal:
```bash
# Watch server output
tail -f <server_output.log>
```

---

## Testing with Different Parameters

### volumeclaim Template Tests

**Test 1: Default parameters only**
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Test 2: Override namespace**
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "custom-ns"}}'
```

**Test 3: Override all parameters**
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "templateName": "my-pvc",
      "namespace": "production",
      "storage": "200Gi",
      "storageClassName": "premium"
    }
  }'
```

### postgresql Template Tests

**Test 1: Minimal required parameters**
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/postgresql/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "namespace": "databases",
      "username": "admin"
    }
  }'
```

**Test 2: All parameters**
```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/postgresql/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "instanceClass": "db.t3.large",
      "namespace": "production",
      "databaseName": "myapp_db",
      "username": "dbuser",
      "storageSize": "100",
      "backupRetention": "30",
      "enableEncryption": true,
      "tags": ["prod", "critical"]
    }
  }'
```

---

## Response Validation Checklist

For each request, verify:

- ✅ **Status Code**: 200, 404, 400, 500 as expected
- ✅ **Content-Type**: `application/json`
- ✅ **JSON Valid**: Can parse with `python3 -m json.tool`
- ✅ **Required Fields**: `apiVersion`, `kind`, `metadata`
- ✅ **Data Completeness**: All expected fields present
- ✅ **CORS Headers**: `Access-Control-Allow-Origin: *`

**Example validation:**
```bash
curl -i http://localhost:8080/api/v1/claim-templates/volumeclaim | head -20
```

Output shows:
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Content-Type: application/json
Date: Fri, 10 Jan 2026 14:05:26 GMT
Content-Length: 2345
```

---

## Expected Parameter Values

### volumeclaim Parameters
| Parameter | Type | Required | Default | Example |
|-----------|------|----------|---------|---------|
| templateName | string | no | demo | "my-volume" |
| namespace | string | **yes** | default | "production" |
| storage | string | **yes** | 20Gi | "100Gi" |
| storageClassName | string | no | standard | "fast" |

### postgresql Parameters
| Parameter | Type | Required | Default | Example |
|-----------|------|----------|---------|---------|
| instanceClass | string | **yes** | db.t3.micro | "db.t3.large" |
| namespace | string | **yes** | databases | "production" |
| databaseName | string | **yes** | mydb | "myapp_db" |
| username | string | **yes** | - | "dbuser" |
| storageSize | string | **yes** | 20 | "100" |
| backupRetention | string | no | 7 | "30" |
| enableEncryption | boolean | no | true | true |
| tags | array | no | - | ["prod"] |

---

## Troubleshooting

### Server Won't Start
```bash
# Check if port is in use
lsof -i :8080

# Kill existing process
kill -9 $(lsof -t -i:8080)

# Start server
go run main.go
```

### Connection Refused
```bash
# Make sure server is running
curl http://localhost:8080/health

# If fails, check logs:
# - Do you see "API server listening"?
# - Are there any error messages?
```

### Timeout on Render
- Rendering can take a few seconds (OCI pull on first run)
- Subsequent requests are faster
- This is expected

### Invalid JSON Response
- Verify `Content-Type: application/json` header
- Use `python3 -m json.tool` to validate
- Check server logs for errors

---

## Quick Reference

| Endpoint | Method | Purpose | Example |
|----------|--------|---------|---------|
| `/health` | GET | Server health | `curl localhost:8080/health` |
| `/api/v1/claim-templates` | GET | List all | `curl localhost:8080/api/v1/claim-templates` |
| `/api/v1/claim-templates/{name}` | GET | Get one | `curl localhost:8080/api/v1/claim-templates/volumeclaim` |
| `/api/v1/claim-templates/{name}/order` | POST | Render | `curl -X POST ... -d '{"parameters":{...}}'` |

---

**Happy Testing! 🚀**
