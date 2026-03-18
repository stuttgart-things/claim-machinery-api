# Claim Machinery API

A Backstage-compatible API for discovering, managing, and rendering KCL-based Crossplane claim templates.

## Overview

The Claim Machinery API provides a REST interface for working with Crossplane claim templates, enabling:

- 📋 Template discovery and browsing
- 🎯 Schema inspection with parameters and validation rules
- 🔧 Claim rendering with custom parameters
- 🐳 OCI registry support

## Quick Start

### Running the API

```bash
# Start the API server (default command)
go run main.go

# API available at
http://localhost:8080
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/claim-templates` | GET | List all templates |
| `/api/v1/claim-templates/{name}` | GET | Get template details |
| `/api/v1/claim-templates/{name}/order` | POST | Render a claim |
| `/version` | GET | API version info |
| `/docs` | GET | OpenAPI documentation |

## Features

### Template Discovery

Browse available claim templates with metadata and descriptions.

### Parameter Validation

Built-in validation ensures parameters meet requirements before rendering.

### KCL Integration

Leverage KCL's type system for safe claim generation.

## Architecture

The API is built with:

- **Go** - Primary language
- **KCL** - Template rendering engine
- **OpenAPI** - API specification
- **Gorilla Mux** - HTTP routing

## Related Documentation

- [API Examples](api-examples.md)
- [Testing Guide](TESTING_GUIDE.md)
- [OpenAPI Specification](openapi.yaml)
