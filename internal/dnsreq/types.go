// Package dnsreq defines DNS request and response types for the IPFS SDK.
// These types are isolated in a separate package to allow mockery to generate
// mocks without creating import cycles in test files.
package dnsreq

import (
	"go.lumeweb.com/ipfs-sdk/internal/client"
)

// Type aliases for DNS types from generated client
type ZoneListResponse = client.ZoneListResponse
type ZoneResponse = client.ZoneResponse
type ZoneRequest = client.ZoneRequest
type RecordResponse = client.RecordResponse
type RecordRequest = client.RecordRequest
type RecordIdentifier = client.RecordIdentifier
type RecordResult = client.RecordResult
type BulkRecordRequest = client.BulkRecordRequest
type BulkDeleteRequest = client.BulkDeleteRequest
type ErrorResponse = client.ErrorResponse
