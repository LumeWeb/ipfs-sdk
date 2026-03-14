// Package http provides shared HTTP utilities for IPFS SDK clients.
//
// This package implements retry logic and error handling for HTTP requests
// across all IPFS SDK services. It provides a configurable retry mechanism
// with sensible defaults for handling transient failures.
//
// Retry Configuration
//
// The package uses a configurable retry strategy with the following components:
//
//   - Attempts: Number of retry attempts (default: 3)
//   - MaxDelay: Maximum delay between retries (default: 30s)
//   - MaxJitter: Maximum random jitter (default: 5s)
//   - LastErrorOnly: Return only the last error from retry attempts
//
// Retry Behavior
//
// HTTP requests are retried for certain status codes:
//
//   - Always retried: 429 (Too Many Requests)
//   - Never retried (unrecoverable): 400, 401, 403, 404, 405, 409, 422
//   - Never retried (all 4xx): Other client errors
//   - Potentially retried: Server errors (5xx)
//
// Unrecoverable Status Codes
//
// The following HTTP status codes are considered unrecoverable and will not
// trigger retry attempts:
//
//   - 400 (Bad Request): Invalid request format or parameters
//   - 401 (Unauthorized): Authentication failed
//   - 403 (Forbidden): Insufficient permissions
//   - 404 (Not Found): Resource does not exist
//   - 405 (Method Not Allowed): HTTP method not supported
//   - 409 (Conflict): Resource state conflicts
//   - 422 (Unprocessable Entity): Semantically incorrect request
//
// Rate Limiting
//
// HTTP 429 (Too Many Requests) status codes are always retried, as they
// typically indicate temporary rate limiting rather than fatal errors.
//
// Usage
//
// Retry context-aware operations:
//
//   cfg := http.DefaultRetryConfig()
//   err := http.RetryContext(ctx, cfg, func() error {
//       return client.DoRequest()
//   })
//
// For standard retry options, use the exported Retry function with options:
//
//   err := http.Retry(
//       func() error { return client.DoRequest() },
//       http.RetryOptions(ctx)...,
//   )
package http
