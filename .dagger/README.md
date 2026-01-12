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

## 🚀 Quick Start

### Run Tests (Recommended for CI/CD)
```bash
dagger call -m .dagger build-and-test --src . --progress plain
```

### Build Binary Only
```bash
dagger call -m .dagger build --src . export --path=/tmp/go/build/
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

---

## 📝 Module Structure

```
.dagger/
├── main.go          # Module definition
├── build.go         # Build function implementation
├── test.go          # BuildAndTest function implementation
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
- Verify Go version compatibility
- Check that source files are present
- Ensure Go modules are properly configured

---

## 📚 Related Documentation

- [Dagger Documentation](https://docs.dagger.io)
- [Go Build Reference](https://golang.org/doc/install)
- Claim Machinery API: [../README.md](../README.md)
