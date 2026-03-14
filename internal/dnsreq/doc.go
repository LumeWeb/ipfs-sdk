// Package dnsreq provides DNS request and response types for the IPFS SDK.
//
// This package contains type aliases for DNS-related types from the generated
// API client. These types are isolated in a separate internal package to allow
// mockery to generate mocks without creating import cycles in test files.
//
// Package Structure
//
// All types in this package are direct aliases to types generated from the
// OpenAPI specification (internal/client). This isolation allows:
//
//   - Mock generation without import cycles
//   - Clean separation between generated code and service logic
//   - Type safety while maintaining generated type compatibility
//
// Types
//
// The following types are defined as aliases to generated client types:
//   - ZoneListResponse, ZoneResponse, ZoneRequest: DNS zone management
//   - RecordResponse, RecordRequest, RecordIdentifier: DNS record operations
//   - RecordResult, BulkRecordRequest, BulkDeleteRequest: Bulk DNS operations
//   - ErrorResponse: Error response structures
//
// Usage
//
// These types are used by the DNSService to provide type-safe DNS operations.
// Direct usage outside of the DNS service is not intended.
package dnsreq
