# 🧪 Claim Machinery API - Dagger Module

This Dagger module provides automated build, test, and deployment pipelines for the Claim Machinery API.

## 📋 Available Functions

### `build-and-test`
Builds the API binary and runs integration tests against a running instance.

**Features:**
- ✅ Compiles the Go binary with configurable options
- ✅ Starts the API service in a containerized environment
- ✅ Runs integration tests from a separate test container
- ✅ Validates `/health` and `/api/v1/claim-templates` endpoints

**Usage:**
```bash
dagger call -m .dagger build-and-test \
  --src . \
  --progress plain
```

**Optional Parameters:**
```bash
dagger call -m .dagger build-and-test \
  --src . \
  --goVersion 1.25.5 \
  --os linux \
  --arch amd64 \
  --port 8080 \
  --progress plain
```

---

### `build`
Compiles the Go binary and exports it to the host filesystem.

**Usage:**
```bash
dagger call -m .dagger build \
  --src "." \
  export --path=/tmp/go/build/claim-machinery-api/ \
  --progress plain
```

**Optional Parameters:**
```bash
dagger call -m .dagger build \
  --src "." \
  --goVersion 1.25.5 \
  --os linux \
  --arch amd64 \
  --goMainFile main.go \
  --binName claim-machinery-api \
  --variant bookworm \
  export --path=/tmp/go/build/claim-machinery-api/ \
  --progress plain
```

---

### `lint`
Run static Go lint and export the report.

**Usage:**
```bash
dagger call -m .dagger lint \
  --src . \
  export --path=/tmp/lint-report.txt \
  --progress plain
```

---

### `test`
Run `go test` with configurable `goVersion` and `testArg`. Returns the raw output as a string.

**Usage:**
```bash
dagger call -m .dagger test \
  --src . \
  --go-version 1.24.4 \
  --test-arg "./..." \
  --progress plain
```

### `test-report`
Run `go test` and export the output as a file (useful for CI artifacts).

**Usage:**
```bash
dagger call -m .dagger test-report \
  --src . \
  --go-version 1.24.4 \
  --test-arg "./..." \
  export --path=/tmp/test-output.txt \
  --progress plain
```

---

### `build-image`
Builds a container image using ko and optionally scans it for vulnerabilities.

**Features:**
- ✅ Builds optimized Go container images with ko
- ✅ Pushes to any OCI-compliant registry (including ttl.sh, ghcr.io)
- ✅ Optional integrated Trivy security scanning
- ✅ No Docker daemon required
- ✅ Multi-arch support via ko

**Usage (Build Only):**
```bash
# Build without pushing (local OCI layout)
dagger call -m .dagger build-image \
  --src . \
  --repo ko.local/claim-machinery-api \
  --push false \
  --progress plain
```

**Usage (Build & Push):**
```bash
# Push to ttl.sh (no auth required, 1h TTL)
dagger call -m .dagger build-image \
  --src . \
  --repo ttl.sh/claim-machinery-api-test \
  --push true \
  --progress plain

# Push to GitHub Container Registry
dagger call -m .dagger build-image \
  --src . \
  --repo ghcr.io/stuttgart-things/claim-machinery-api \
  --push true \
  --token env:GITHUB_TOKEN \
  --progress plain
```

**Usage (Build, Push & Scan):**
```bash
# Build, push and scan for vulnerabilities
dagger call -m .dagger build-image \
  --src . \
  --repo ttl.sh/claim-machinery-api-test \
  --push true \
  --scan true \
  --scan-severity "HIGH,CRITICAL" \
  --progress plain
```

**Parameters:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `repo` | `ko.local` | Image repository (e.g., `ttl.sh/myapp`, `ghcr.io/org/app`) |
| `push` | `true` | Push to registry (`true`) or local build only (`false`) |
| `buildArg` | `.` | Package to build (usually current directory) |
| `koVersion` | `v0.18.1` | Ko version to use |
| `tokenName` | `GITHUB_TOKEN` | Environment variable name for registry auth |
| `token` | - | Secret for registry authentication (optional for public registries) |
| `scan` | `false` | Enable Trivy vulnerability scanning |
| `scanSeverity` | `HIGH,CRITICAL` | Severity levels to scan for |

---

### `scan-image`
Scans a container image for vulnerabilities using Trivy.

**Features:**
- ✅ Scans any OCI-compliant image
- ✅ Configurable severity levels
- ✅ JSON report output
- ✅ Supports private registries with authentication

**Usage:**
```bash
# Scan a public image
dagger call -m .dagger scan-image \
  --image-ref ttl.sh/claim-machinery-api:latest \
  --severity "HIGH,CRITICAL" \
  export --path /tmp/scan-report.json \
  --progress plain

# Scan a private image
dagger call -m .dagger scan-image \
  --image-ref ghcr.io/org/private-app:v1.0.0 \
  --registry-user env:GITHUB_USER \
  --registry-password env:GITHUB_TOKEN \
  --severity "MEDIUM,HIGH,CRITICAL" \
  export --path /tmp/scan-report.json \
  --progress plain
```

**Parameters:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `imageRef` | - | Full image reference (e.g., `ttl.sh/app:tag`) |
| `registryUser` | - | Username for private registry authentication |
| `registryPassword` | - | Password/token for private registry authentication |
| `severity` | `HIGH,CRITICAL` | Comma-separated severity levels to report |
| `trivyVersion` | `0.64.1` | Trivy scanner version |

---

## 🚀 Quick Start

### Run Tests (Recommended for CI/CD)
```bash
dagger call -m .dagger build-and-test --src . --progress plain
```

### Build Binary Only
```bash
dagger call -m .dagger build --src . export --path=/tmp/go/build/
```

### Build & Push Container Image
```bash
# Quick test with ttl.sh (expires in 1h)
dagger call -m .dagger build-image \
  --src . \
  --repo ttl.sh/claim-machinery-api-$(openssl rand -hex 4) \
  --push true \
  --scan true \
  --progress plain

# Production release to GHCR
dagger call -m .dagger build-image \
  --src . \
  --repo ghcr.io/stuttgart-things/claim-machinery-api \
  --push true \
  --scan true \
  --token env:GITHUB_TOKEN \
  --progress plain
```

---

## 📊 Test Output

The `build-and-test` function provides colored, formatted output:

```
========================================
   Claim Machinery API Test Suite
========================================

⏳ Waiting for API to be ready...
✓ API is ready

[1/2] Testing /health endpoint
✓ Health check passed
  Response: {"status":"healthy","timestamp":"2026-01-12T13:46:30Z"}

[2/2] Testing /api/v1/claim-templates endpoint
✓ Templates endpoint passed
  Found 2 templates

========================================
  All tests passed! 🎉
========================================
```

---

## 🔧 Parameters Reference

### Build Parameters
| Parameter | Default | Description |
|-----------|---------|-------------|
| `goVersion` | `1.25.5` | Go version for compilation |
| `os` | `linux` | Target operating system |
| `arch` | `amd64` | Target architecture |
| `goMainFile` | `main.go` | Entry point file |
| `binName` | `claim-machinery-api` | Output binary name |
| `variant` | `bookworm` | Debian variant for build environment |
| `ldflags` | `` | Linker flags for build |
| `packageName` | `` | Package name override |
| `port` | `8080` | API service port (build-and-test only) |

### Container Image Parameters
| Parameter | Default | Description |
|-----------|---------|-------------|
| `repo` | `ko.local` | Image repository URL |
| `push` | `true` | Push to registry |
| `buildArg` | `.` | Package to build |
| `koVersion` | `v0.18.1` | Ko version |
| `scan` | `false` | Enable vulnerability scanning |
| `scanSeverity` | `HIGH,CRITICAL` | Severity levels to scan |

### Security Scan Parameters
| Parameter | Default | Description |
|-----------|---------|-------------|
| `imageRef` | - | Full image reference (required) |
| `severity` | `HIGH,CRITICAL` | Severity levels to report |
| `trivyVersion` | `0.64.1` | Trivy version |

---

## 📝 Module Structure

```
.dagger/
├── main.go          # Module definition
├── build.go         # Binary build functions
├── image.go         # Container image build & scan functions
├── test.go          # Test & integration functions
├── lint.go          # Linting functions
├── dagger.gen.go    # Generated code (auto-generated)
├── go.mod           # Go module definition
└── README.md        # This file
```

---

## 🎯 Common Tasks

### Build for Different Architectures
```bash
# ARM64 build
dagger call -m .dagger build --src . --arch arm64 \
  export --path=/tmp/go/build/

# Windows executable
dagger call -m .dagger build --src . --os windows --arch amd64 \
  export --path=/tmp/go/build/
```

### Container Image Workflows

**Development/Testing:**
```bash
# Quick build & push to ttl.sh (1h expiration, no auth)
dagger call -m .dagger build-image \
  --src . \
  --repo ttl.sh/test-$(openssl rand -hex 4) \
  --push true \
  --scan true \
  --progress plain
```

**Production Release:**
```bash
# Build, scan and push to GHCR
dagger call -m .dagger build-image \
  --src . \
  --repo ghcr.io/stuttgart-things/claim-machinery-api \
  --push true \
  --scan true \
  --scan-severity "CRITICAL" \
  --token env:GITHUB_TOKEN \
  --progress plain
```

**Scan Existing Image:**
```bash
# Audit a deployed image
dagger call -m .dagger scan-image \
  --image-ref ghcr.io/stuttgart-things/claim-machinery-api:v1.0.0 \
  --severity "MEDIUM,HIGH,CRITICAL" \
  export --path /tmp/audit-report.json \
  --progress plain
```

### Custom Build with Linker Flags
```bash
dagger call -m .dagger build --src . \
  --ldflags "-X main.Version=v1.0.0 -X main.BuildTime=$(date)" \
  export --path=/tmp/go/build/
```

---

## 🐛 Troubleshooting

**API fails to start in tests:**
- Ensure all required template files are included in the source directory
- Check that port 8080 is available
- Review API logs in Dagger output

**Build fails:**
- Verify Go version compatibility (requires Go 1.25.5+)
- Check that source files are present
- Ensure Go modules are properly configured

**Image push fails:**
- Verify registry credentials for private registries
- Check network connectivity to registry
- For ttl.sh: No authentication needed, check image name format
- For GHCR: Ensure `GITHUB_TOKEN` has `packages:write` permission

**Scan fails:**
- Verify image exists and is accessible
- Check Trivy version compatibility
- For private images, provide `registry-user` and `registry-password`

---

## 📚 Related Documentation

- [Dagger Documentation](https://docs.dagger.io)
- [Go Build Reference](https://golang.org/doc/install)
- Claim Machinery API: [../README.md](../README.md)
