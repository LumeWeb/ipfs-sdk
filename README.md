# IPFS SDK for Go

[![Go Version](https://img.shields.io/badge/Go-1.25.0-blue)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://github.com/lumeweb/ipfs-sdk/actions/workflows/go.yml/badge.svg)](https://github.com/lumeweb/ipfs-sdk/actions/workflows/go.yml)
[![Coverage Status](https://img.shields.io/badge/Coverage-Report-green)](https://github.com/lumeweb/ipfs-sdk/actions)

A Go SDK for interacting with IPFS HTTP gateway services. Provides a clean, idiomatic interface for common IPFS operations including pinning, DNS management, IPNS, website deployment, and file uploads.

## Features

- **Pinning Service** - Pin and unpin content using.boxo client
- **DNS Service** - Manage DNS zones and records with DNSLink support
- **IPNS Service** - Inter-Planetary Naming System key management
- **Websites Service** - Deploy and manage gateway websites
- **Upload Service** - TUS resumable uploads and HTTP POST with memory-efficient CAR generation
- **OpenAPI-driven** - Code generated from `swagger.yaml` for type safety
- **Configurable retry** - Exponential backoff with customizable attempts

## Installation

```bash
go get go.lumeweb.com/ipfs-sdk
```

## Quick Start

```go
import (
    "context"
    "log"

    "go.lumeweb.com/ipfs-sdk"
)

func main() {
    client, err := ipfs.NewClient("https://api.example.com", "your-bearer-token")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // List all pins
    pins, err := client.Pinning().ListPins(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found %d pins", len(pins))
}
```

## Services

### Pinning Service

```go
// List, add, get status, and remove pins
pins, _ := client.Pinning().ListPins(ctx)
c, _ := cid.Decode("QmYourCidHere")
pin, _ := client.Pinning().AddPin(ctx, c)
status, _ := client.Pinning().GetPin(ctx, pin.GetRequestId())
client.Pinning().RemovePin(ctx, pin.GetRequestId())
```

### DNS Service

```go
// Manage DNS zones and DNSLink records
zones, _ := client.DNS().ListZones(ctx)
zone, _ := client.DNS().CreateZone(ctx, "example.com", nameservers)
record, _ := client.DNS().CreateRecord(ctx, zone.Id, request)
records, _ := client.DNS().ListRecords(ctx, zone.Id)
```

### IPNS Service

```go
// List, create, publish, and resolve IPNS keys
keys, _ := client.IPNS().ListKeys(ctx)
key, _ := client.IPNS().CreateKey(ctx, "my-key")
publishResult, _ := client.IPNS().Publish(ctx, key.Id, cid)
resolveResult, _ := client.IPNS().Resolve(ctx, name)
```

### Websites Service

```go
// Deploy and manage gateway websites
websites, _ := client.Websites().List(ctx)
site, _ := client.Websites().Create(ctx, "domain.com", cid, "ipfs")
updatedSite, _ := client.Websites().Update(ctx, site.Id, domain, cid, protocol)
```

### Upload Service

```go
// Upload files with automatic TUS/POST selection
file, _ := os.Open("path/to/file.txt")
result, _ := client.Upload().Upload(ctx, file, "file.txt", ipfs.UploadOptions{
    MemoryLimit: 100 * 1024 * 1024,
    WrapInDir:   true,
})
// result.CID contains the uploaded content CID
```

## Configuration

```go
import httputil "go.lumeweb.com/ipfs-sdk/internal/http"

retryConfig := httputil.RetryConfig{
    Attempts:      5,
    MaxJitter:     10 * 1000000000,
    MaxDelay:      60 * 1000000000,
}

client, _ := ipfs.NewClient(
    "https://api.example.com",
    "your-token",
    ipfs.WithRetryConfig(retryConfig),
)
```

### Retry Behavior

Default retry configuration:
- **3 attempts** with exponential backoff
- **Unrecoverable codes** (not retried): 400, 401, 403, 404, 405, 409, 422
- **Rate limit (429)** always retried

## Architecture

The SDK uses a layered architecture:

```
Client Entry Point (Pinning, DNS, IPNS, Websites, Upload)
    ↓
Service Interfaces (Type-safe adapters)
    ↓
Generated OpenAPI Client (swagger.yaml → oapi-codegen)
    ↓
HTTP Infrastructure (Retry, Auth, net/http)
```

### Upload Package

Implements two-pass CAR generation for memory-efficient large file handling:

1. **Pass 1**: Build summary with metadata (CIDs, sizes, tree structure)
2. **Pass 2**: Generate CARv1 on demand using LRU blockstore (default 100MB limit)
3. Enables processing of content larger than available RAM

## Development

```bash
# Build
go build -v ./...

# Test with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# Generate code
go generate ./internal/client
mockery

# Dependencies
go mod download
go mod tidy
```

## Contributing

See [AGENTS.md](AGENTS.md) for development guidelines and project architecture details.

## License

MIT License – see [LICENSE](LICENSE) for details.

Copyright (c) 2026 Hammer Technologies LLC
