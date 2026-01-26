# Claim Machinery API

A Backstage-compatible API for discovering, managing, and rendering KCL-based Crossplane claim templates.

## Features

<details>
<summary><strong>Feature Overview</strong></summary>

| Feature | Description |
|---------|-------------|
| Template Discovery | Browse and search KCL-based Crossplane claim templates |
| Template Details | Get schema information including parameters, validation rules, and UI hints |
| Claim Rendering | Render claims with custom parameters using KCL |
| Backstage Integration | Native support for Backstage Software Catalog |
| OCI Support | Load templates from OCI registries |
| Parameter Validation | Built-in parameter validation with custom rules |

</details>

## API

<details>
<summary><strong>API Endpoints Overview</strong></summary>

```bash
# List all available claim templates
GET /api/v1/claim-templates

# Get template details with schema
GET /api/v1/claim-templates/{name}

# Render a claim with parameters
POST /api/v1/claim-templates/{name}/order
```

</details>

<details>
<summary><strong>Version Endpoint</strong></summary>

```bash
curl http://localhost:8080/version
# {"version":"dev","commit":"none","buildDate":"unknown"}
```

</details>

<details>
<summary><strong>OpenAPI Specification and Documentation</strong></summary>

```bash
# OpenAPI spec (served from docs/openapi.yaml if present)
curl http://localhost:8080/openapi.yaml

# Redoc UI
open http://localhost:8080/docs
```

</details>

<details>
<summary><strong>Health Check</strong></summary>

```bash
curl http://localhost:8080/health
```

</details>

<details>
<summary><strong>List All Templates</strong></summary>

```bash
curl http://localhost:8080/api/v1/claim-templates
```

</details>

<details>
<summary><strong>Get Single Template Details</strong></summary>

```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim
```

```bash
curl http://localhost:8080/api/v1/claim-templates/harborproject
```

</details>

<details>
<summary><strong>Render Template - VolumeClaim Example</strong></summary>

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{}'
```

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "100Gi"}}'
```

**Extract YAML from response:**

```bash
curl -s -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "100Gi"}}' | jq -r '.rendered'
```

</details>

<details>
<summary><strong>Render Template - HarborProject Example</strong></summary>

**With default parameters:**

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/harborproject/order \
  -H "Content-Type: application/json" \
  -d '{}'
```

**With custom parameters:**

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/harborproject/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "projectName": "my-app-project",
      "harborURL": "https://harbor.idp.kubermatic.sva.dev",
      "storageQuota": 10737418240,
      "harborInsecure": false,
      "providerConfigRef": "default"
    }
  }'
```

**Extract YAML from response:**

```bash
curl -s -X POST http://localhost:8080/api/v1/claim-templates/harborproject/order \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "projectName": "my-app-project",
      "harborURL": "https://harbor.idp.kubermatic.sva.dev",
      "storageQuota": 10737418240
    }
  }' | jq -r '.rendered'
```

</details>

## Development

<details>
<summary><strong>Getting Started</strong></summary>

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download
go run main.go
```

</details>

<details>
<summary><strong>Debug Mode</strong></summary>

Enable debug logging to see parameter processing:

```bash
DEBUG=1 go run main.go
```

</details>

### CLI Tools (MVP)

Two interactive CLI tools are available in `/tests` for testing and development.

<details>
<summary><strong>Local KCL CLI (tests/cli)</strong></summary>

Renders templates directly using KCL (requires `kcl` CLI installed locally).

**Build:**

```bash
go build -o tests/cli/claim-cli ./tests/cli/
```

**Usage:**

```bash
# Default profile (tests/profile.yaml)
./tests/cli/claim-cli

# Custom profile
TEMPLATE_PROFILE_PATH=/path/to/profile.yaml ./tests/cli/claim-cli
```

**Features:**
- Interactive template selection
- Dynamic form based on template parameters
- Enum fields as dropdowns
- Default values pre-filled
- Saves rendered YAML to `/tmp/{template}-{name}.yaml`

</details>

<details>
<summary><strong>API-Connected CLI (tests/cli-api)</strong></summary>

Connects to the running API server - no local KCL required.

**Build:**

```bash
go build -o tests/cli-api/claim-cli-api ./tests/cli-api/
```

**Usage:**

```bash
# Start the API first
go run main.go

# Then run CLI (default: localhost:8080)
./tests/cli-api/claim-cli-api

# Custom API URL
CLAIM_API_URL=http://api.example.com:8080 ./tests/cli-api/claim-cli-api
```

**Features:**
- Same interactive UX as local CLI
- Lightweight client (no KCL dependency)
- Works with remote API servers
- Good for testing API changes

</details>

Both CLIs support an "Enter-Enter" workflow - defaults are pre-selected so you can quickly render with minimal input.

## CI/CD

<details>
<summary><strong>Dagger Build Pipeline</strong></summary>

This project uses [Dagger](https://dagger.io) for reproducible builds, tests, and container image creation.

**Available functions:**

| Function | Description |
|----------|-------------|
| `build-and-test` | Compile binary and run integration tests |
| `build` | Build Go binary only |
| `build-image` | Build container image with ko (with optional Trivy scanning) |
| `scan-image` | Scan container images for vulnerabilities |
| `lint` | Run Go linting |
| `test` | Run Go tests |

**Quick start:**

```bash
# Run tests
dagger call -m .dagger build-and-test --src . --progress plain

# Build container image and push to ttl.sh
dagger call -m .dagger build-image \
  --src . \
  --repo ttl.sh/claim-machinery-api-test \
  --push true \
  --scan true \
  --progress plain

# Scan existing image
dagger call -m .dagger scan-image \
  --image-ref ttl.sh/my-app:latest \
  --severity "HIGH,CRITICAL" \
  export --path /tmp/scan-report.json
```

Full documentation: [.dagger/README.md](.dagger/README.md)

</details>

<details>
<summary><strong>Task Automation</strong></summary>

Common tasks are available via [Taskfile](https://taskfile.dev):

```bash
# Interactive task selector
task do

# Build and push image
task build-push-image

# Scan an image
task scan-image

# Run API locally
task run-local-go
```

See [Taskfile.yaml](Taskfile.yaml) for all available tasks.

</details>

## Configuration

<details>
<summary><strong>Templates Directory</strong></summary>

Configure the templates directory (defaults to `internal/claimtemplate/testdata`):

```bash
export TEMPLATES_DIR=/path/to/your/templates
go run main.go
```

Equivalent via CLI flag (overrides env):

```bash
go run main.go --templates-dir /path/to/your/templates
```

</details>

<details>
<summary><strong>Template Profile</strong></summary>

Add additional templates via profile file (merged with directory):

```yaml
---
templates:
  - https://raw.githubusercontent.com/stuttgart-things/kcl/refs/heads/main/crossplane/claim-xplane-volumeclaim/templates/volumeclaim-simple.yaml
  - /tmp/template123.yaml
```

```bash
export TEMPLATE_PROFILE_PATH=/absolute/path/to/profile.yaml
go run main.go
```

Or via CLI flag (overrides env):

```bash
go run main.go --template-profile-path /absolute/path/to/profile.yaml
```

**Behavior:**
- Profile entries (URLs/paths) are validated; unreachable entries trigger a warning and are skipped
- Templates from the profile and directory are merged; duplicates are deduplicated based on `metadata.name` (profile takes precedence)
- On startup, the API displays loaded sources and final template names

</details>

<details>
<summary><strong>Server Port</strong></summary>

Set a custom port with the `PORT` environment variable (default `8080`):

```bash
PORT=9090 go run main.go
```

</details>

<details>
<summary><strong>Logging</strong></summary>

- Standard: Text logs with method, path, status, duration, remote IP, and user agent
- Enable JSON logs:

```bash
LOG_FORMAT=json go run main.go
```

</details>

<details>
<summary><strong>Request ID and Correlation</strong></summary>

- Incoming `X-Request-ID` header is preserved; otherwise the server generates an ID
- Response always includes the `X-Request-ID` header (CORS: exposed)
- Logs (text/JSON) include `requestId` for correlation
- On panics, the server returns JSON with `{"error":"internal server error","requestId":"..."}` and logs structured output

</details>

## Documentation

<details>
<summary><strong>Additional Resources</strong></summary>

| Document | Description |
|----------|-------------|
| [SPEC.md](./SPEC.md) | Full technical specification |
| [ROADMAP.md](./ROADMAP.md) | Project roadmap and tracking |
| [API Examples](./docs/api-examples.md) | API usage examples |

</details>

## License

Apache 2.0
