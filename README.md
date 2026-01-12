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
<summary>3️⃣ Get Single Template Details</summary>

```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim
```

</details>

<details open>
<summary>4️⃣ Render Template </summary>

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

## DEV

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download
go run main.go
```


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

## Documentation

- [SPEC.md](./SPEC.md) - Full technical specification
- [ROADMAP.md](./ROADMAP.md) - Project roadmap and tracking
- [API Examples](./docs/api-examples.md) - API usage examples

## License

Apache 2.0
