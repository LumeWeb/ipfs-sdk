# AGENTS.md
This file provides development guidelines and architectural documentation for the IPFS SDK.

## Common Commands

### Building
```bash
# Build all packages
go build -v ./...

# Build specific package
go build -v ./go.lumeweb.com/ipfs-sdk
```

### Testing
```bash
# Run all tests with race detection and coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Run specific package tests
go test -v ./...

# View coverage report
go tool cover -func=coverage.out
```

### Code Generation
```bash
# Generate OpenAPI client from swagger.yaml
go generate ./internal/client

# Generate Pinning Service client from IPFS pinning service spec
go generate ./internal/pinning

# Generate mocks for interfaces
mockery
```

### Dependency Management
```bash
# Download dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy dependencies
go mod tidy
```

## High-Level Architecture

### Core Design
This is a Go SDK for interacting with IPFS HTTP gateway services. The SDK uses a layered architecture with the main `Client` serving as the entry point to specialized services.

### Module Structure
- **Module path**: `go.lumeweb.com/ipfs-sdk`
- **Go version**: 1.25.0

### Key Services
The `Client` struct provides access to five specialized services:
1. **PinningService** - Generated client from IPFS Pinning Service API spec for pinning content
2. **DNSService** - Manage DNS zones and records (DNSLink support)
3. **IPNSService** - Inter-Planetary Naming System key management
4. **WebsitesService** - Gateway website deployment
5. **UploadService** - File upload via TUS resumable upload protocol

### Code Generation Pattern
The SDK uses OpenAPI specifications to generate HTTP client code:

**Main API Client:**
1. **OpenAPI Spec**: `swagger.yaml` defines all API endpoints
2. **Generated Client**: `internal/client/client.gen.go` is generated via `go tool oapi-codegen`
3. **Adapter Interfaces**: Each service (DNS, IPNS, Websites) defines its own interface that wraps the generated client
4. **Mock Generation**: `mockery` generates test mocks from these adapter interfaces

**Pinning Service Client:**
1. **OpenAPI Spec**: `internal/pinning/swagger.yaml` (from official IPFS Pinning Service API)
2. **Generated Client**: `internal/pinning/client.gen.go` is generated via `go tool oapi-codegen`
3. **Type Aliases**: Types are exposed cleanly via type aliases to avoid import cycles
4. **Mock Generation**: `mockery` generates test mocks from the PinningService interface

This pattern isolates the generated code, allows for clean API surfaces on each service, and enables dependency injection for testing.

### Directory Organization
```
├── sdk.go, auth.go, retry.go, errors.go, options.go              # Core SDK infrastructure
├── pinning.go                                                # Pinning service (generated from IPFS pinning service spec)
├── dns.go                                                    # DNS service
├── ipns.go                                                   # IPNS service
├── websites.go                                               # Websites service
├── upload.go                                                 # Upload service (TUS & HTTP POST)
├── internal/
│   ├── client/
│   │   ├── client.gen.go                                   # Generated OpenAPI client
│   │   ├── doc.go                                           # go:generate directive
│   │   └── oai-codegen.yaml                                # Codegen config
│   ├── http/
│   │   └── http.go                                          # Retry utilities (UnrecoverableStatusCodes)
│   ├── pinning/
│   │   ├── client.gen.go                                    # Generated OpenAPI client (IPFS Pinning Service)
│   │   ├── doc.go                                           # go:generate directive
│   │   ├── oai-codegen.yaml                                # Codegen config
│   │   └── swagger.yaml                                     # IPFS pinning service spec
│   ├── dnsreq/
│   │   └── types.go                                        # DNS types isolated to avoid import cycles
│   └── testutil/
│       └── http.go                                          # HTTP test utilities
├── pkg/upload/
│   ├── doc.go                                               # Package docs
│   ├── car.go                                               # CAR streaming utilities
│   ├── car_blockstore.go                                    # LRU blockstore implementation
│   ├── car_dir_builder.go                                   # Directory tree builder
│   ├── unixfs_generator.go                                  # UnixFS node generation
│   └── internal/
│       ├── io/                                              # I/O utilities
│       ├── encoding/                                        # Encoding utilities
│       └── carv1/                                           # CARv1 format handling
├── mocks/                                                    # Generated mocks (DNSClientWithResponsesInterface, etc.)
├── swagger.yaml                                              # OpenAPI specification
├── .mockery.yaml                                             # Mockery configuration
├── git-town.toml                                             # Git workflow config
└── knope.toml                                                # Release management
```

### Memory-Efficient Upload Architecture
The `pkg/upload` package implements a two-pass CAR generation approach for handling large directory trees:

1. **Pass 1**: Walk filesystem and build summary with metadata (CIDs, sizes, tree structure)
2. **Pass 2**: Generate CARv1 file by regenerating blocks from filesystem on demand

This uses an LRU blockstore (`LRUBlockstore`) with configurable memory limits (default 100MB), enabling processing of content larger than available RAM.

### Authentication
All services use Bearer token authentication passed via `Authorization` header. The token is set on the main `Client` and used by internal request editors.

### Retry Strategy
All HTTP requests use configurable retry behavior via `internal/http/http.go`:
- Default: 3 attempts with exponential backoff
- Unrecoverable status codes (not retried): 400, 401, 403, 404, 405, 409, 422
- Rate limit (429) is always retried
- Configurable via `RetryConfig` per service

### Type Alias Pattern
Services use type aliases to expose generated types from the internal client package, avoiding direct imports in external-facing APIs. This keeps the public API clean while leveraging generated models.

## Important Constraints

1. **Code Generation**: Generated files are auto-generated. Do not edit them manually.
   - `internal/client/client.gen.go` - run `go generate ./internal/client` after modifying `swagger.yaml`
   - `internal/pinning/client.gen.go` - run `go generate ./internal/pinning` after modifying `internal/pinning/swagger.yaml`

2. **Mock Generation**: Mocks are generated using mockery. Configure interfaces in `.mockery.yaml`.

3. **Import Cycles**: The `internal/dnsreq` package isolates DNS request types to allow mockery to generate mocks without creating import cycles in test files.

4. **Service Adapters**: Each service (DNS, IPNS, Websites) must define its own `*ClientWithResponsesInterface` and adapter to convert the generated `ClientWithResponses` to the service-specific interface.

5. **IPFS CID Types**: The SDK uses `github.com/ipfs/go-cid` for CID operations. Ensure proper CID validation when calling service methods.

6. **Context Usage**: All service methods accept `context.Context`. Always use timeout contexts for operations that may block.

7. **Error Handling**: Service methods return standard Go errors. Check for `context.Canceled` when operations fail with context cancellation.

8. **Testing**: Use the generated mocks in the `mocks` package with `testify/mock` assertions. See existing test files for patterns. Use `NewPinningService` with `WithPinningHTTPClient` for testing with custom HTTP clients.
