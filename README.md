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

**[Documentation](https://stuttgart-things.github.io/claim-machinery-api/)**

## Features

| Feature | Description |
|---------|-------------|
| Template Discovery | Browse and search KCL-based Crossplane claim templates |
| Template Details | Get schema information including parameters, validation rules, and UI hints |
| Claim Rendering | Render claims with custom parameters using KCL |
| Backstage Integration | Native support for Backstage Software Catalog |
| OCI Support | Load templates from OCI registries |
| Parameter Schema | Exposes parameter metadata (types, enums, patterns, defaults) for client-side validation |
| Interactive CLI | Terminal UI for rendering claims with forms, dropdowns, and live YAML preview |

> **Note:** Authentication and authorization are not built into the API. These are expected to be handled by an upstream API gateway or service mesh.

## Architecture

```
┌──────────────────────────────────────────┐
│           HTTP Request                   │
├──────────────────────────────────────────┤
│  Middleware (CORS, logging, request ID,  │
│             error/panic recovery)        │
├──────────────────────────────────────────┤
│  API Handlers (list, get, order claims)  │
├──────────────────────────────────────────┤
│  Application Layer                       │
│  - Parameter merging with defaults       │
│  - Template loading (dir + profile)      │
├──────────────────────────────────────────┤
│  KCL Rendering Engine                    │
│  - OCI-based (kcl run oci://...)         │
│  - File-based (Go SDK)                   │
├──────────────────────────────────────────┤
│  External: OCI Registries (ghcr.io)      │
└──────────────────────────────────────────┘
```

**Tech stack:** Go, Gorilla Mux, KCL SDK, Cobra, Charmbracelet (bubbletea/huh)

## Quick Start

```bash
git clone https://github.com/stuttgart-things/claim-machinery-api.git
cd claim-machinery-api
go mod download

# Run API server
go run main.go server

# Or interactive CLI (default when no subcommand is given)
go run main.go render
```

The server loads templates from `tests/profile.yaml` by default and listens on port `8080`.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/claim-templates` | GET | List all available claim templates |
| `/api/v1/claim-templates/{name}` | GET | Get template details with parameter schema |
| `/api/v1/claim-templates/{name}/order` | POST | Render a claim with custom parameters |
| `/health` | GET | Health check |
| `/version` | GET | Build version metadata |
| `/openapi.yaml` | GET | OpenAPI specification |
| `/docs` | GET | OpenAPI documentation (Redoc UI) |

### List Templates

```bash
curl http://localhost:8080/api/v1/claim-templates
```

Response:

```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "ClaimTemplateList",
  "items": [
    {
      "apiVersion": "resources.stuttgart-things.com/v1alpha1",
      "kind": "ClaimTemplate",
      "metadata": {
        "name": "volumeclaim",
        "title": "Crossplane Volume Claim",
        "description": "Creates a persistent volume claim via Crossplane",
        "tags": ["storage", "crossplane"]
      },
      "spec": { "type": "volumeclaim", "source": "oci://...", "parameters": [...] }
    }
  ]
}
```

### Get Template Details

```bash
curl http://localhost:8080/api/v1/claim-templates/volumeclaim
```

Returns the full `ClaimTemplate` object including all parameter definitions (type, enum, default, required, pattern, etc.).

### Render a Claim

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "100Gi"}}'
```

Response:

```json
{
  "apiVersion": "api.claim-machinery.io/v1alpha1",
  "kind": "OrderResponse",
  "metadata": {
    "name": "volumeclaim",
    "timestamp": "2025-01-10T12:00:00Z"
  },
  "rendered": "apiVersion: sthings.io/v1alpha1\nkind: VolumeClaim\n..."
}
```

Extract just the YAML:

```bash
curl -s -X POST http://localhost:8080/api/v1/claim-templates/volumeclaim/order \
  -H "Content-Type: application/json" \
  -d '{"parameters": {"namespace": "production", "storage": "100Gi"}}' | jq -r '.rendered'
```

Passing an empty body `{}` renders with all default values.

<details>
<summary><strong>More Examples (HarborProject)</strong></summary>

```bash
# With default parameters
curl -X POST http://localhost:8080/api/v1/claim-templates/harborproject/order \
  -H "Content-Type: application/json" \
  -d '{}'

# With custom parameters
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

</details>

### Health & Version

```bash
curl http://localhost:8080/health
# {"status":"healthy","timestamp":"2025-01-10T12:00:00Z"}

curl http://localhost:8080/version
# {"version":"dev","commit":"none","buildDate":"unknown"}
```

## Usage

### API Server

```bash
claim-machinery-api server
```

### Interactive CLI (Default)

```bash
# Default when no subcommand is given
claim-machinery-api

# Or explicitly
claim-machinery-api render
```

Features: template selection, dynamic parameter forms, enum dropdowns, random value selection, default pre-fill, type/pattern validation, live YAML preview, and optional file save.

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
export ENABLE_TEMPLATES_DIR=true  # enable loading from templates directory
export PORT=9090
export LOG_FORMAT=json   # default: text
```

**Priority:** Flag > Environment Variable > Default

<details>
<summary><strong>Template Profile</strong></summary>

Add additional templates via a profile file (merged with the templates directory):

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

Or as container:

```bash
docker run --rm \
  -v $PWD/tests/profile.yaml:/tmp/profile.yaml \
  -e TEMPLATE_PROFILE_PATH=/tmp/profile.yaml \
  -e TEMPLATES_DIR="/tmp" \
  -p 8080:8080 \
  ghcr.io/stuttgart-things/claim-machinery-api:v0.5.6
```

**Behavior:**
- Profile entries (URLs/paths) are validated; unreachable entries trigger a warning and are skipped
- Templates from the profile and directory are merged; duplicates are deduplicated based on `metadata.name` (profile takes precedence)
- On startup, the API displays loaded sources and final template names

</details>

<details>
<summary><strong>Request ID and Correlation</strong></summary>

- Incoming `X-Request-ID` header is preserved; otherwise the server generates an ID
- Response always includes the `X-Request-ID` header (CORS: exposed)
- Logs (text/JSON) include `requestId` for correlation
- On panics, the server returns JSON with `{"error":"internal server error","requestId":"..."}` and logs structured output

</details>

<details>
<summary><strong>Debug Mode</strong></summary>

Enable debug logging to see parameter processing:

```bash
DEBUG=1 go run main.go
```

</details>

## Development

<details>
<summary><strong>Testing</strong></summary>

```bash
go test ./...
```

</details>

<details>
<summary><strong>Legacy Test CLIs (tests/cli & tests/cli-api)</strong></summary>

> **Note:** These are legacy test tools. The integrated CLI via `claim-machinery-api render` is recommended.

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

## Deployment

The release pipeline publishes a kustomize base as an OCI artifact (`ghcr.io/stuttgart-things/claim-machinery-api-kustomize:<version>`) and a container image (`ghcr.io/stuttgart-things/claim-machinery-api:<version>`).

<details>
<summary><strong>Kustomize Overlay</strong></summary>

A kustomize overlay example is provided in `deployment/overlays/example/`. Pull the base and customize:

```bash
oras pull ghcr.io/stuttgart-things/claim-machinery-api-kustomize:v0.5.6 \
  -o deployment/kustomize-base

kubectl kustomize deployment/overlays/example/
```

</details>

<details>
<summary><strong>Flux (GitOps)</strong></summary>

A Flux app definition is available in the [stuttgart-things/flux](https://github.com/stuttgart-things/flux) repository at [`apps/claim-machinery-api/`](https://github.com/stuttgart-things/flux/tree/main/apps/claim-machinery-api). It uses a two-layer Flux reconciliation with `OCIRepository` + Flux `Kustomization` (not Helm) and Gateway API `HTTPRoute`.

**1. Create the GitRepository source:**

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: stuttgart-things-flux
  namespace: flux-system
spec:
  interval: 1m0s
  url: https://github.com/stuttgart-things/flux.git
  ref:
    tag: v1.1.0
```

**2. Create the Kustomization:**

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: claim-machinery-api
  namespace: flux-system
spec:
  interval: 1h
  retryInterval: 1m
  timeout: 5m
  sourceRef:
    kind: GitRepository
    name: stuttgart-things-flux
  path: ./apps/claim-machinery-api
  prune: true
  wait: true
  postBuild:
    substitute:
      CLAIM_MACHINERY_API_NAMESPACE: claim-machinery
      CLAIM_MACHINERY_API_VERSION: v0.6.0
      GATEWAY_NAME: my-gateway
      GATEWAY_NAMESPACE: default
      HOSTNAME: claim-api
      DOMAIN: example.sthings-vsphere.labul.sva.de
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `CLAIM_MACHINERY_API_NAMESPACE` | `claim-machinery` | Target namespace |
| `CLAIM_MACHINERY_API_VERSION` | `v0.5.6` | OCI tag + container image tag |
| `GATEWAY_NAME` | _(required)_ | Gateway parentRef name |
| `GATEWAY_NAMESPACE` | `default` | Gateway parentRef namespace |
| `HOSTNAME` | _(required)_ | HTTPRoute hostname prefix |
| `DOMAIN` | _(required)_ | HTTPRoute domain suffix |

**How it works:** The outer Kustomization reads `./apps/claim-machinery-api` from the GitRepository, substitutes variables, and creates the Namespace + OCIRepository + inner Kustomization + HTTPRoute. The inner Kustomization (`release.yaml`) reconciles the OCI kustomize base from `ghcr.io/stuttgart-things/claim-machinery-api-kustomize`, patches out the Ingress (replaced by HTTPRoute), overrides the container image tag, and applies the resulting manifests.

</details>

## Documentation

| Document | Description |
|----------|-------------|
| [SPEC.md](./docs/SPEC.md) | Full technical specification |
| [ROADMAP.md](./docs/ROADMAP.md) | Project roadmap and tracking |
| [API Examples](./docs/api-examples.md) | API usage examples |
| [Template Schema](./docs/claim-template-schema.md) | Claim template specification |
| [CI/CD Pattern](./docs/CICD.md) | CI/CD pipeline stages, Dagger functions, Taskfile interface |
| [OpenAPI Spec](./docs/openapi.yaml) | OpenAPI / Swagger definition |

## License

Apache 2.0
