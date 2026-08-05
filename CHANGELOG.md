## 0.1.68 (2026-08-05)

### Features

- add DomainNamespace enum for website domain namespaces

## 0.1.67 (2026-08-05)

### Fixes

- export DNSDelegation and DNSDelegationRecord types

## 0.1.66 (2026-08-05)

### Features

- add GetDomainDNSRequirements returning delegation records

## 0.1.65 (2026-08-04)

### Features

- expose domain status in DomainResponse

## 0.1.64 (2026-08-04)

### Fixes

- tag dnsname nested module in knope
- single package so root module keeps unprefixed tags

## 0.1.63 (2026-08-04)

### Features

- add SDK infrastructure
- add core services (DNS, IPNS, Pinning, Websites)
- add CAR upload and main SDK client
- add archive package with format detection and extraction
- add archive processing configuration
- add UploadBytes API and fix upload limit routing
- add host override cascade support and switch to generated pinning client
- add io.Seeker support to BytesFS
- add trustless download service using boxo gateway patterns
- add UploadFile convenience method with SingleFileFS
- add StatFS support to bytesfs and singlefilefs
- add UnixFS-aware download APIs
- add FileSize API and fix UnixFS handling
- optimize FileSize with dual-strategy approach
- add DNS validation, SSL status updates, and polling methods
- add CreateWithOptions for dns_hosting_enabled support
- add UpdateWithOptions method for architectural consistency
- add client-level gateway secret authentication for internal APIs
- add rate limiting with worker pool and intelligent retry
- enable download service configuration through client options
- refactor and restructure rate limiting backend architecture
- optimize metadata queries with block meta API
- track UnixFS and DAG sizes in upload results
- add optional params to IPNS and new DNS methods
- expose CAR builder options for configurable chunk size, DAG layout, and max links
- add ListDirectoryPath and tests for directory path listing
- add ErrNotFound and ErrGone sentinels for errors.Is() matching
- add WebsiteValidationReason enum and helpers, fix flaky test
- update swagger spec with upload result endpoint
- add PingService + update swagger from live portal
- regenerate IPFS SDK from deployed swagger
- update swagger spec and regenerate client
- add DNS cert/TLSA endpoint wrappers to DNSService
- update swagger spec with namespace field on zone response
- add DAGService with ResolveDAG endpoint wrapper
- add website domain binding methods to WebsitesService
- add MetaService with ExportSiaObject and ExportDAG endpoints
- add shared DNS name normalization module

### Fixes

- address PR review feedback
- add retry logic to remaining DNS service methods
- resolve compilation errors in CAR upload package
- consolidate retry policy to single source of truth
- address all PR review feedback from kody-ai bot
- resolve test failures after PR review changes
- address critical PR review feedback
- address PR review feedback for critical bug fixes
- add upload data type enum and comprehensive upload route tests
- migrate pkg modules to ipfs-content library
- use DefaultMemoryLimit constant instead of hardcoded value
- extract StreamToPipe helper to simplify CAR generation streaming
- address PR review feedback
- return actual file size in DirEntry.Info
- use url.JoinPath for proper URL construction in upload service
- normalize base URL in NewClient for consistent URL handling
- return error instead of constructing malformed URL on parse failure
- address PR review feedback
- initialize multipart upload with initial data
- remove double-upload bug in TUS upload
- update car API usage for compatibility
- return file from root, not directory
- add authorization header support to TUS uploads
- extract AuthSchemeBearer constant to eliminate duplication
- add mutex synchronization to authRoundTripper for thread safety
- TUS upload size validation and memory store implementation
- address PR review feedback
- resolve test timeout by removing deadlock in NewUpload
- address all code review feedback from PR #23
- use package-level auth token variables instead of os.Getenv calls
- update ipfs-content
- address PR review feedback from kody-ai[bot]
- normalize error handling in SingleFileFS.Stat
- address PR review security and test issues
- use fileSize parameter in createChunkedBlock
- exclude dot paths and return only immediate children
- close skipped dot-path nodes to prevent resource leak
- update ipfs-content
- update OpenAPI spec to match running portal service and regenerate client
- update DNS service to handle array response types
- handle HTTP 200 and 201 status codes for create operations
- update swagger.yaml to match current API
- preserve context error unwrapping and improve polling behavior
- expose SSLStatusUpdateRequest type to public API
- update WebsiteItemResponse to array type
- update swagger spec and regenerate client code
- WaitForIPNSResolution CID comparison handles both path and plain CID formats
- update swagger.yaml and fix array response handling
- eliminate worker pool resource leak on backend initialization
- rate limiter correctly retries on 429 errors from upstream components
- add Stop method to RateLimitedBlockstore to prevent goroutine leaks
- restore 429 string error detection lost in PR #45 refactor
- expose ErrRateLimitExceeded error for rate limit detection
- add unixfs_size field to BlockMetaResponse schema
- eliminate unnecessary memory usage in RateLimitedBlockstore
- improve rate limiting accuracy by passing actual block sizes
- avoid incorrect bandwidth charging in Has operation
- FileSize returns correct UnixFS logical file size instead of block size
- update swagger spec and add GetConfig/WebsiteConfigResponse
- re-export RetryConfig and PollOption from internal/http
- handle 307/308 redirects in uploadViaPOST and fix error priority
- buffer reader before redirect retry loop in uploadViaPOST
- avoid buffering entire upload for seekable readers on redirect retry
- update swagger, use WebsiteUpdateRequest for partial updates
- update swagger, add nameservers to WebsiteConfigResponse
- update swagger, republish per-key and add is_subdomain
- add active_cid and ipns_key_id to WebsiteItem/WebsiteResponse
- add required reason field to ZoneStatus/WebsiteValidateResponse
- handle 410 Gone on website Get
- handle 410 Gone in WaitForWebsiteStatus poll callback
- update swagger specs and regenerate client
- propagate SetHTTPClient to internal generated client
- update swagger spec to use ErrorWrapper for IPFS error responses
- tag dnsname nested module in knope

## 0.1.62 (2026-08-04)

### Features

- add shared DNS name normalization module

## 0.1.61 (2026-08-04)

### Fixes

#### Remove the duplicated meta export service and refresh the client spec

Removes the meta export service (CID DAG and Sia object endpoints) that
is served by the dedicated meta service and provided by the portal-sdk
module. Refreshes the swagger spec and regenerated client against the
deployed ipfs endpoint definition, including 422 validation responses.

## 0.1.60 (2026-07-31)

### Features

- add MetaService with ExportSiaObject and ExportDAG endpoints for CID metadata export

## 0.1.59 (2026-07-31)

### Features

- add website domain binding methods to WebsitesService (ListDomains, BindDomain, UnbindDomain, VerifyDomain)

## 0.1.58 (2026-07-30)

### Features

- add DAGService with ResolveDAG endpoint wrapper

## 0.1.57 (2026-07-30)

### Features

- update swagger spec with namespace field on zone response

## 0.1.56 (2026-07-30)

### Features

- add DNS cert/TLSA endpoint wrappers to DNSService

## 0.1.55 (2026-07-30)

### Features

- update swagger spec and regenerate client

## 0.1.54 (2026-07-18)

### Features

- regenerate IPFS SDK from deployed swagger

## 0.1.53 (2026-07-17)

### Fixes

- update swagger spec to use ErrorWrapper for IPFS error responses

## 0.1.52 (2026-07-02)

### Fixes

- propagate SetHTTPClient to internal generated client

## 0.1.51 (2026-07-01)

### Fixes

- update swagger specs and regenerate client

## 0.1.50 (2026-06-19)

### Fixes

- handle 410 Gone on website Get
- handle 410 Gone in WaitForWebsiteStatus poll callback

## 0.1.49 (2026-06-05)

### Features

- add PingService + update swagger from live portal

## 0.1.48 (2026-05-23)

### Features

- update swagger spec with upload result endpoint

## 0.1.47 (2026-05-22)

### Features

- add WebsiteValidationReason enum and helpers, fix flaky test

## 0.1.46 (2026-05-22)

### Fixes

- add required reason field to ZoneStatus/WebsiteValidateResponse

## 0.1.45 (2026-05-22)

### Fixes

- add active_cid and ipns_key_id to WebsiteItem/WebsiteResponse

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
