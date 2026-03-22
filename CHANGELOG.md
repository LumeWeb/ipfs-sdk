## 0.1.23 (2026-03-22)

### Fixes

- update WebsiteItemResponse to array type

## 0.1.22 (2026-03-22)

### Fixes

- expose SSLStatusUpdateRequest type to public API

## 0.1.21 (2026-03-22)

### Features

- add DNS validation, SSL status updates, and polling methods

### Fixes

- update swagger.yaml to match current API
- preserve context error unwrapping and improve polling behavior

## 0.1.20 (2026-03-21)

### Fixes

- handle HTTP 200 and 201 status codes for create operations

## 0.1.19 (2026-03-21)

### Fixes

- update OpenAPI spec to match running portal service and regenerate client
- update DNS service to handle array response types

## 0.1.18 (2026-03-21)

### Fixes

- update ipfs-content

## 0.1.17 (2026-03-20)

### Fixes

- exclude dot paths and return only immediate children
- close skipped dot-path nodes to prevent resource leak

## 0.1.16 (2026-03-20)

### Features

- optimize FileSize with dual-strategy approach

### Fixes

- use fileSize parameter in createChunkedBlock

## 0.1.15 (2026-03-20)

### Features

- add FileSize API and fix UnixFS handling

## 0.1.14 (2026-03-20)

### Features

- add UnixFS-aware download APIs

### Fixes

- address PR review security and test issues

## 0.1.13 (2026-03-19)

### Features

- add StatFS support to bytesfs and singlefilefs

### Fixes

- normalize error handling in SingleFileFS.Stat

## 0.1.12 (2026-03-17)

### Features

- add UploadFile convenience method with SingleFileFS

### Fixes

- address PR review feedback from kody-ai[bot]

## 0.1.11 (2026-03-17)

### Fixes

- update ipfs-content

## 0.1.10 (2026-03-17)

### Features

- add trustless download service using boxo gateway patterns

### Fixes

- address all code review feedback from PR #23
- use package-level auth token variables instead of os.Getenv calls

## 0.1.9 (2026-03-16)

### Fixes

- TUS upload size validation and memory store implementation
- address PR review feedback
- resolve test timeout by removing deadlock in NewUpload

## 0.1.8 (2026-03-16)

### Fixes

- return file from root, not directory
- add authorization header support to TUS uploads
- extract AuthSchemeBearer constant to eliminate duplication
- add mutex synchronization to authRoundTripper for thread safety

## 0.1.7 (2026-03-15)

### Features

- add io.Seeker support to BytesFS

### Fixes

- update car API usage for compatibility

## 0.1.6 (2026-03-15)

### Fixes

- initialize multipart upload with initial data
- remove double-upload bug in TUS upload

## 0.1.5 (2026-03-14)

### Features

- add host override cascade support and switch to generated pinning client

### Fixes

- address PR review feedback

## 0.1.4 (2026-03-14)

### Fixes

- use url.JoinPath for proper URL construction in upload service
- normalize base URL in NewClient for consistent URL handling
- return error instead of constructing malformed URL on parse failure

## 0.1.3 (2026-03-14)

### Features

- add UploadBytes API and fix upload limit routing

### Fixes

- return actual file size in DirEntry.Info

## 0.1.2 (2026-03-14)

### Features

- add archive processing configuration

### Fixes

- extract StreamToPipe helper to simplify CAR generation streaming
- address PR review feedback

## 0.1.1 (2026-03-14)

### Features

- add archive package with format detection and extraction

### Fixes

- add upload data type enum and comprehensive upload route tests
- migrate pkg modules to ipfs-content library
- use DefaultMemoryLimit constant instead of hardcoded value

## 0.1.0 (2026-03-12)

### Breaking Changes

- Initial release

### Features

- add SDK infrastructure
- add core services (DNS, IPNS, Pinning, Websites)
- add CAR upload and main SDK client

### Fixes

- address PR review feedback
- add retry logic to remaining DNS service methods
- resolve compilation errors in CAR upload package
- consolidate retry policy to single source of truth
- address all PR review feedback from kody-ai bot
- resolve test failures after PR review changes
- address critical PR review feedback
- address PR review feedback for critical bug fixes
