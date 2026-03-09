# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claim Machinery API is a Go REST API and interactive CLI for managing Crossplane claim templates. It discovers ClaimTemplate YAML definitions (from local directories and remote URLs), exposes them via a REST API, and renders them through KCL (a configuration language). The project belongs to the stuttgart-things organization.

## Build & Development Commands

**Run locally:**
```bash
go run main.go server                                    # Start API server (default port 8080)
go run main.go render                                    # Start interactive TUI for template rendering
go run main.go --templates-dir internal/claimtemplate/testdata --template-profile-path tests/profile.yaml
```

**Tests:**
```bash
go test ./...                                            # Run all tests
go test -v ./internal/claimtemplate/...                   # Run tests for a specific package
go test -run TestHandlerName ./internal/api/...           # Run a single test
```

**Lint & Build (via Dagger):**
```bash
dagger call -m .dagger lint --src . export --path=/tmp/lint-report.txt --progress plain
dagger call -m .dagger build-and-test --src . --progress plain
dagger call -m .dagger build --src . --bin-name=claim-machinery-api export --path=/tmp/go/build --progress plain
```

**Task runner (Taskfile.yaml):**
```bash
task do                          # Interactive task selector
task run-local-go                # Run API locally
task lint                        # Lint via Dagger
task build-test-api              # Build + test via Dagger
task pr                          # Full PR workflow: lint, build, test, scan
task commit                      # Commit with pre-commit validation
```

## Architecture

```
cmd/                     Cobra CLI commands (server, render, version)
internal/
  api/                   HTTP layer: Gorilla Mux router, handlers, middleware (CORS, logging, request ID, error recovery)
  app/                   Business logic: template loading, profile parsing, parameter merging, rendering orchestration
  render/                KCL execution engine (OCI-based via CLI, file-based via Go SDK)
  claimtemplate/         Domain model: ClaimTemplate struct, YAML deserialization
  notify/                Homerun2 (omni-pitcher) notification sender
  version/               Build metadata (set via ldflags)
```

**Request flow:** HTTP handler -> `app.BuildParameterValues` (merge defaults + request params) -> `app.RenderTemplate` -> `render.KCL` execution -> JSON response -> (async) `notify.SendHomerunMessage`

**Template loading:** Templates come from two sources merged at startup:
1. Directory scan (`TEMPLATES_DIR` env var / `--templates-dir` flag) - loads all YAML files
2. Profile file (`TEMPLATE_PROFILE_PATH` / `--template-profile-path`) - YAML file listing template URLs and local paths; profile entries override directory entries by `metadata.name`

## API Endpoints

- `GET /api/v1/claim-templates` - List all templates
- `GET /api/v1/claim-templates/{name}` - Get specific template
- `POST /api/v1/claim-templates/{name}/order` - Render template with `{"parameters": {...}}` body
- `GET /health`, `GET /version`, `GET /docs`, `GET /openapi.yaml`

Responses use Kubernetes-style format with `apiVersion: api.claim-machinery.io/v1alpha1` and `kind` fields.

## Key Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `PORT` | HTTP server port | `8080` |
| `TEMPLATES_DIR` | Local template directory | - |
| `TEMPLATE_PROFILE_PATH` | Profile YAML path | - |
| `DEBUG` | Enable debug logging (`1`/`true`/`yes`) | off |
| `ENABLE_TEST_ROUTES` | Enable `/__test/*` endpoints | off |
| `LOG_FORMAT` | Log format (`json` or text) | text |
| `ENABLE_HOMERUN` | Enable homerun2 notifications (`true`/`1`/`yes`) | off |
| `HOMERUN_URL` | Omni-pitcher base URL (e.g. `https://pitcher.example.com`) | - |
| `HOMERUN_AUTH_TOKEN` | Bearer token for pitcher `/pitch` endpoint | - |

## Testing Conventions

- Uses `testify` for assertions (`assert.*`, `require.*`)
- Test files live alongside source (`*_test.go`)
- Test fixtures in `internal/claimtemplate/testdata/` (YAML ClaimTemplate files)
- HTTP handler tests use `httptest.NewRequest` / `httptest.NewRecorder`

## CI/CD

- **Dagger module** in `.dagger/` provides: `build`, `build-and-test`, `lint`, `test`, `build-image`, `scan-image`, `run-api`
- **GitHub Actions** in `.github/workflows/` for PR validation and image builds
- **GoReleaser** (`.goreleaser.yaml`) for multi-platform binary releases
- **Pre-commit hooks** configured in `.pre-commit-config.yaml` (trailing whitespace, YAML validation, secret detection, shellcheck, hadolint)

## ClaimTemplate Schema

Templates are YAML files with `apiVersion`, `kind: ClaimTemplate`, `metadata` (name, description, tags), and `spec` containing:
- `kclModuleRef` - OCI reference for the KCL module to execute
- `parameters[]` - Each has: name, type (string/boolean/integer), default, enum, pattern, hidden, multiselect, allowRandom
