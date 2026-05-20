## 0.1.44 (2026-05-20)

### Fixes

- update swagger, republish per-key and add is_subdomain

## 0.1.43 (2026-05-19)

### Features

- add ErrNotFound and ErrGone sentinels for errors.Is() matching

## 0.1.42 (2026-05-19)

### Fixes

- update swagger, add nameservers to WebsiteConfigResponse

## 0.1.41 (2026-05-19)

### Fixes

- update swagger, use WebsiteUpdateRequest for partial updates

## 0.1.40 (2026-05-18)

### Fixes

- handle 307/308 redirects in uploadViaPOST and fix error priority
- buffer reader before redirect retry loop in uploadViaPOST
- avoid buffering entire upload for seekable readers on redirect retry

## 0.1.39 (2026-05-18)

### Features

- add ListDirectoryPath and tests for directory path listing

## 0.1.38 (2026-05-17)

### Fixes

- re-export RetryConfig and PollOption from internal/http

## 0.1.37 (2026-05-17)

### Fixes

- update swagger spec and add GetConfig/WebsiteConfigResponse

## 0.1.36 (2026-05-17)

### Features

- expose CAR builder options for configurable chunk size, DAG layout, and max links

## 0.1.35 (2026-04-23)

### Features

- add optional params to IPNS and new DNS methods

## 0.1.34 (2026-04-10)

### Fixes

- FileSize returns correct UnixFS logical file size instead of block size

## 0.1.33 (2026-04-08)

### Features

- optimize metadata queries with block meta API
- track UnixFS and DAG sizes in upload results

### Fixes

- add unixfs_size field to BlockMetaResponse schema
- eliminate unnecessary memory usage in RateLimitedBlockstore
- improve rate limiting accuracy by passing actual block sizes
- avoid incorrect bandwidth charging in Has operation

## 0.1.32 (2026-04-07)

### Fixes

- restore 429 string error detection lost in PR #45 refactor
- expose ErrRateLimitExceeded error for rate limit detection

## 0.1.31 (2026-04-07)

### Features

- refactor and restructure rate limiting backend architecture

### Fixes

- add Stop method to RateLimitedBlockstore to prevent goroutine leaks

## 0.1.30 (2026-04-07)

### Fixes

- rate limiter correctly retries on 429 errors from upstream components

## 0.1.29 (2026-04-06)

### Features

- add rate limiting with worker pool and intelligent retry
- enable download service configuration through client options

### Fixes

- eliminate worker pool resource leak on backend initialization

## 0.1.28 (2026-03-24)

### Fixes

- update swagger.yaml and fix array response handling

## 0.1.27 (2026-03-23)

### Fixes

- WaitForIPNSResolution CID comparison handles both path and plain CID formats

## 0.1.26 (2026-03-23)

### Features

- add client-level gateway secret authentication for internal APIs

## 0.1.25 (2026-03-23)

### Fixes

- update swagger spec and regenerate client code

## 0.1.24 (2026-03-22)

### Features

- add CreateWithOptions for dns_hosting_enabled support
- add UpdateWithOptions method for architectural consistency

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
