# Claim Machinery API

```
███╗   ███╗ █████╗  ██████╗██╗  ██╗██╗███╗   ██╗███████╗██████╗ ██╗   ██╗
████╗ ████║██╔══██╗██╔════╝██║  ██║██║████╗  ██║██╔════╝██╔══██╗╚██╗ ██╔╝
██╔████╔██║███████║██║     ███████║██║██╔██╗ ██║█████╗  ██████╔╝ ╚████╔╝
██║╚██╔╝██║██╔══██║██║     ██╔══██║██║██║╚██╗██║██╔══╝  ██╔══██╗  ╚██╔╝
██║ ╚═╝ ██║██║  ██║╚██████╗██║  ██║██║██║ ╚████║███████╗██║  ██║   ██║
╚═╝     ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝   ╚═╝
```

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

<details>

<summary><strong>Template Profile</strong></summary>

Add additional templates via profile file (merged with directory):

templates:
```bash
cat <<EOF > profile.yaml
---
templates:
  - https://raw.githubusercontent.com/stuttgart-things/kcl/refs/heads/main/crossplane/claim-xplane-volumeclaim/templates/volumeclaim-simple.yaml
  - /tmp/template123.yaml
EOF
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

## Usage

### API Server (Default)

Start the HTTP API server:

```bash
# Run directly (default mode)
claim-machinery-api

# Or explicitly
claim-machinery-api server
```

The server will:
- Load templates from `internal/claimtemplate/testdata` (default)
- Additionally load from `tests/profile.yaml` (default profile)
- Start HTTP server on port 8080

### Interactive CLI

Render claims interactively with a terminal UI:

```bash
claim-machinery-api render
```

**Features:**
- Interactive template selection
- Dynamic forms based on template parameters
- Enum fields as dropdowns
- Random value selection for enums
- Default values pre-filled
- Parameter validation (type, pattern, required)
- Live YAML preview with syntax highlighting
- Optional file save with suggested path

**Workflow:**
1. Select a template from the list
2. Fill in parameters (or use defaults)
3. Confirm rendering
4. View rendered YAML
5. Optionally save to file

### Configuration

Both modes support the same configuration options:

```bash
# Custom templates directory
claim-machinery-api --templates-dir /path/to/templates

# Custom profile (or disable with "")
claim-machinery-api --template-profile-path /path/to/profile.yaml

# Environment variables
export TEMPLATES_DIR=/path/to/templates
export TEMPLATE_PROFILE_PATH=/path/to/profile.yaml
claim-machinery-api render
```

**Priority:** Flag > Environment Variable > Default

## Development

<details>
<summary><strong>Getting Started</strong></summary>

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download

# Run API server
go run main.go

# Or interactive CLI
go run main.go render
```

</details>

<details>
<summary><strong>Debug Mode</strong></summary>

Enable debug logging to see parameter processing:

```bash
DEBUG=1 go run main.go
```

</details>

### Testing CLIs

<details>
<summary><strong>Legacy Test CLIs (tests/cli & tests/cli-api)</strong></summary>

> **Note:** These are legacy test tools. The new integrated CLI via `claim-machinery-api render` is recommended.

Two interactive CLI tools are available in `/tests` for testing and development.

**Local KCL CLI (tests/cli):**

Renders templates directly using KCL (requires `kcl` CLI installed locally).

```bash
go build -o tests/cli/claim-cli ./tests/cli/
./tests/cli/claim-cli
```

**API-Connected CLI (tests/cli-api):**

Connects to the running API server - no local KCL required.

```bash
# Start API first
go run main.go

# Then run CLI
go build -o tests/cli-api/claim-cli-api ./tests/cli-api/
./tests/cli-api/claim-cli-api
```

</details>

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
