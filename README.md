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

</details>

### Installation

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download
go run main.go
```

### API Endpoints

```bash
# List all available claim templates
GET /api/v1/claim-templates

# Get template details with schema
GET /api/v1/claim-templates/{name}

# Render a claim with parameters
POST /api/v1/claim-templates/{name}/order
```

## Documentation

- [SPEC.md](./SPEC.md) - Full technical specification
- [ROADMAP.md](./ROADMAP.md) - Project roadmap and tracking
- [API Examples](./docs/api-examples.md) - API usage examples

## License

Apache 2.0
