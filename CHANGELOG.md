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
