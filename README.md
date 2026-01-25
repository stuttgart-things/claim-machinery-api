# Claim Machinery API

A Backstage-compatible API for discovering, managing, and rendering KCL-based Crossplane claim templates.

## Features

- 📋 **Template Discovery**: Browse and search KCL-based Crossplane claim templates
- 🎯 **Template Details**: Get schema information including parameters, validation rules, and UI hints
- 🔧 **Claim Rendering**: Render claims with custom parameters using KCL
- 🏗️ **Backstage Integration**: Native support for Backstage Software Catalog
- 🐳 **OCI Support**: Load templates from OCI registries
- ✅ **Parameter Validation**: Built-in parameter validation with custom rules

## API

<details open>
<summary>API Endpoints</summary>

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
<summary>🔢 Version</summary>

```bash
curl http://localhost:8080/version
# {"version":"dev","commit":"none","buildDate":"unknown"}
```

</details>

<details open>
<summary>📜 OpenAPI & Docs</summary>

```bash
# OpenAPI spec (served from docs/openapi.yaml if present)
curl http://localhost:8080/openapi.yaml

# Redoc UI
open http://localhost:8080/docs
```

</details>

<details open>
<summary>1️⃣ Health Check</summary>

```bash
curl http://localhost:8080/health
```

</details>

<details open>
<summary>2️⃣ List All Templates</summary>

```bash
curl http://localhost:8080/api/v1/claim-templates
```

</details>

<details open>
<parameter name="summary">3️⃣ Get Single Template Details</summary>

```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim
```

```bash
curl http://localhost:8080/api/v1/claim-templates/harborproject
```

</details>

<details open>
<parameter name="summary">4️⃣ Render Template - VolumeClass Example</summary>

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

<details open>
<parameter name="summary">5️⃣ Render Template - HarborProject Example</summary>

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

## DEV

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download
go run main.go
```

### Debug Mode

Enable debug logging to see parameter processing:

```bash
DEBUG=1 go run main.go
```

### CLI Tools (MVP)

Two interactive CLI tools are available in `/tests` for testing and development:

<details>
<summary>🖥️ Local KCL CLI (tests/cli)</summary>

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
<summary>🌐 API-Connected CLI (tests/cli-api)</summary>

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

## Configuration

- Templates directory (defaults to `internal/claimtemplate/testdata`):

```bash
export TEMPLATES_DIR=/path/to/your/templates
go run main.go
```

- Equivalent via CLI flag (overrides env):

```bash
go run main.go --templates-dir /path/to/your/templates
```

- Additional templates via profile file (merge with directory):

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

Behavior:
* Profile entries (URLs/paths) are validated; if they are unreachable, a warning is issued and the entry is skipped.
* Templates from the profile and the directory are merged; duplicates are deduplicated based on metadata.name (the profile takes precedence).
* On startup, the API displays the loaded sources and the final template names being used.

### Server Port

Set a custom port with the `PORT` environment variable (default `8080`):

```bash
PORT=9090 go run main.go
```

### Logging

- Standard: Text-Logs mit Methode, Pfad, Status, Dauer, Remote-IP und User-Agent
- JSON-Logs aktivieren:

```bash
LOG_FORMAT=json go run main.go
```

### Request-ID & Korrelation

- Eingehende `X-Request-ID` wird übernommen; sonst generiert der Server eine ID.
- Antwort enthält immer Header `X-Request-ID` (CORS: exposed).
- Logs (Text/JSON) enthalten `requestId` zur Korrelation.
- Bei Panics liefert der Server JSON mit `{"error":"internal server error","requestId":"..."}` und loggt strukturiert.

## Documentation

- [SPEC.md](./SPEC.md) - Full technical specification
- [ROADMAP.md](./ROADMAP.md) - Project roadmap and tracking
- [API Examples](./docs/api-examples.md) - API usage examples

## License

Apache 2.0
