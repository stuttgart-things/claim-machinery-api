# CI/CD Pattern — Go/KCL Microservice

Standard CI/CD pattern used by `claim-machinery-api` and reusable across Go/KCL microservices in the stuttgart-things organization.

## Pipeline Overview

```
Push to main
  │
  ├─► Lint Repo ──────────────────► lint report
  │
  ├─► Build & Test ───────────────► pass / fail + Dagger logs
  │
  └─► Build, Push & Scan Image ──► ghcr.io/.../app:main + Trivy scan
        │
        └─► Release (on success) ─► GitHub Release + binaries + OCI kustomize base
              │
              └─► Deploy Pages (on success) ─► docs site with release history
```

On **pull requests**, the same lint, build, test, and image-scan steps run but images are pushed to `ttl.sh` (ephemeral, 1-hour TTL) instead of `ghcr.io`.

## Stages

| Stage | Trigger | Workflow | Artifacts |
|-------|---------|----------|-----------|
| **Lint** | push / PR to main | `lint-repo.yaml` | Lint report (text) |
| **Build & Test** | push / PR to main | `build-test.yaml` | Pass/fail, Dagger logs on failure |
| **Build & Scan Image** | push / PR / dispatch | `build-scan-image.yaml` | Container image, Trivy scan report |
| **Release** | build-scan-image success / dispatch | `release.yaml` | GitHub release, binaries, OCI kustomize artifact |
| **Deploy Pages** | release success / dispatch | `pages.yaml` | GitHub Pages site with release history |

## Artifact Matrix

| Artifact | Registry | Tag Pattern | Produced By |
|----------|----------|-------------|-------------|
| Container image (dev) | `ttl.sh/<project>-<random>` | `:latest` | Build & Scan (PR) |
| Container image (main) | `ghcr.io/stuttgart-things/<project>` | `:main` | Build & Scan (main push) |
| Container image (release) | `ghcr.io/stuttgart-things/<project>` | `:v<semver>` | Release (image staging) |
| Go binaries | GitHub Release assets | `v<semver>` | Release (GoReleaser) |
| Kustomize base | `ghcr.io/stuttgart-things/<project>-kustomize` | `:v<semver>` | Release (KCL render + OCI push) |
| Docs site | GitHub Pages | latest | Deploy Pages |

### Binary Targets (GoReleaser)

| OS | Arch |
|----|------|
| linux | amd64, arm64 |
| darwin | amd64, arm64 |
| windows | amd64, arm64 |

## Dagger Functions

All CI steps run inside [Dagger](https://dagger.io) containers for reproducibility. The module is defined in `.dagger/`.

| Function | Purpose | Source |
|----------|---------|--------|
| `lint` | Go linting via `dag.Go().Lint()` | `lint.go` |
| `build` | Compile Go binary with ldflags | `build.go` |
| `build-and-test` | Build binary, start API service, run integration tests (health, templates, request-ID, panic recovery) | `test.go` |
| `test` | Run `go test` | `test.go` |
| `build-image` | Build image with ko, optional Trivy scan | `image.go` |
| `build-image-with-kcl` | Build image with KCL base (non-root, static binary) | `container.go` |
| `scan-image` | Trivy vulnerability scan (HIGH/CRITICAL) | `image.go` |
| `run-api` | Run API as Dagger service with profile | `run.go` |

## Taskfile Interface

The `Taskfile.yaml` wraps Dagger calls and provides the local developer interface.

### PR Validation

```bash
task pr          # Full pipeline: lint → build+test → build+scan image → build binary
task pr-git      # pr + create GitHub PR
```

### Individual Steps

| Task | Dagger Function | Description |
|------|----------------|-------------|
| `task lint` | `lint` | Run Go linting |
| `task build-test-api` | `build-and-test` | Build + integration tests |
| `task build-output-binary` | `build` | Build exportable binary |
| `task build-scan-image-kcl` | `build-image-with-kcl` + `scan-image` | Build KCL image, push to ttl.sh, scan |
| `task build-scan-image-ko` | `build-image` + `scan-image` | Build ko image, push to ttl.sh, scan |
| `task scan-image` | `scan-image` | Scan an existing image |
| `task run-api` | `run-api` | Run API as Dagger service |

### Release & Deploy

| Task | Description |
|------|-------------|
| `task release-local` | Full local release (interactive): semantic-release → goreleaser → stage image → push kustomize base |
| `task release-github` | Same as above, non-interactive (for CI) |
| `task trigger-release` | Trigger Release workflow on GitHub Actions via `gh workflow run` |
| `task trigger-pages` | Trigger Deploy Pages workflow on GitHub Actions via `gh workflow run` |

### KCL Deployment

| Task | Description |
|------|-------------|
| `task render-manifests` | Render Kubernetes manifests with KCL (interactive) |
| `task render-manifests-quick` | Render with defaults (non-interactive) |
| `task render-kustomize-base` | Generate kustomize base from KCL output |
| `task push-kustomize-base` | Push kustomize base as OCI artifact |
| `task deploy` | Pull base + select overlay + apply to cluster |

## Workflow Structure

### Shared Workflows

Linting and release use reusable workflows from [stuttgart-things/github-workflow-templates](https://github.com/stuttgart-things/github-workflow-templates):

- `call-repository-linting.yaml` — YAML, Markdown, and Go linting
- `call-go-microservice-release.yaml` — semantic-release, GoReleaser, image staging, kustomize OCI push

### Shared Taskfiles

External task definitions from [stuttgart-things/tasks](https://github.com/stuttgart-things/tasks):

| Include | Taskfile | Purpose |
|---------|----------|---------|
| `git:` | `git/git.yaml` | Git operations (PR creation) |
| `lint:` | `git/linting.yaml` | Pre-commit linting |
| `go:` | `go/lint.yaml` | Go-specific linting |
| `dagger:` | `dagger/modules.yaml` | Dagger module operations |
| `image:` | `docker/copy.yaml` | Image copy between registries |
| `goBuild:` | `go/build.yaml` | Go build operations |
| `k8sMicroservice:` | `go/k8s-microservice.yaml` | Commit with pre-commit validation |
| `release:` | `go/release-workflow.yaml` | Release pipeline (semantic-release, goreleaser, image staging, kustomize push) |

## Pre-commit Hooks

Local checks configured in `.pre-commit-config.yaml`, run via `task commit`:

| Hook | Purpose |
|------|---------|
| `trailing-whitespace` | Remove trailing whitespace |
| `end-of-file-fixer` | Ensure files end with newline |
| `check-added-large-files` | Prevent large file commits |
| `check-merge-conflict` | Detect merge conflict markers |
| `check-yaml` | Validate YAML syntax |
| `detect-private-key` | Detect accidentally committed keys |
| `detect-secrets` | High-entropy password detection |
| `shellcheck` | Shell script linting |
| `hadolint-docker` | Dockerfile linting |
| `check-github-workflows` | Validate GitHub Actions schema |

## KCL Deployment Module

The `deployment/` directory contains a KCL module that generates all Kubernetes manifests:

```
deployment/
├── kcl.mod              # Module definition (deploy-claim-machinery-api v0.3.0)
├── main.k               # Entry point
├── schema.k             # Configuration schema
├── labels.k             # Common labels
├── deploy.k             # Deployment
├── service.k            # Service
├── serviceaccount.k     # ServiceAccount
├── configmap.k          # ConfigMap
├── secret.k             # Secret
├── ingress.k            # Ingress
├── httproute.k          # Gateway API HTTPRoute
├── namespace.k          # Namespace
└── overlays/example/    # Kustomize overlay example
```

The Release workflow renders this module with `tests/kcl-deploy-profile.yaml` parameters, produces a kustomize base, and pushes it as an OCI artifact to `ghcr.io/stuttgart-things/claim-machinery-api-kustomize:<version>`.

## Applying This Pattern

To apply this CI/CD pattern to a new Go/KCL microservice:

1. **Copy workflows** from `.github/workflows/` — update image repo and project name
2. **Copy `.dagger/`** module — adjust build, test, and image functions as needed
3. **Copy `Taskfile.yaml`** — update vars (`GO_MODULE`, `KUSTOMIZE_OCI_REPO`, etc.)
4. **Copy `.goreleaser.yaml`** — update binary name and ldflags paths
5. **Copy `.pre-commit-config.yaml`** and `.releaserc.json`
6. **Create `deployment/`** KCL module for Kubernetes manifests
7. **Create `mkdocs.yml`** + `docs/` for documentation
