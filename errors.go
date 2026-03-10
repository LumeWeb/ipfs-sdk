package ipfs

import (
	"fmt"
	"net/http"

	"github.com/avast/retry-go/v4"
)

// Operation identifiers for error message mapping.
// Defined as iota constants for automatic incrementing IDs.
const (
	// DNS operations
	OpListZones = iota
	OpCreateZone
	OpGetZone
	OpDeleteZone
	OpListRecords
	OpGetRecord
	OpCreateRecord
	OpUpdateRecord
	OpDeleteRecord
	OpBulkCreateRecords
	OpBulkDeleteRecords

	// IPNS operations
	OpListIPNSKeys
	OpGetIPNSKey
	OpCreateIPNSKey
	OpDeleteIPNSKey
	OpPublishIPNS
	OpRepublishIPNS
	OpResolveIPNS

	// Websites operations
	OpListWebsites
	OpGetWebsite
	OpCreateWebsite
	OpUpdateWebsite
	OpDeleteWebsite
	OpValidateWebsite
	OpGetSSLStatus

	// Pinning operations (boxo - may not use this pattern)
	// Note: Boxo client has its own error handling
)

// operationString maps operation IDs to human-readable names.
var operationString = map[int]string{
	// DNS operations
	OpListZones:        "list DNS zones",
	OpCreateZone:       "create DNS zone",
	OpGetZone:          "get DNS zone",
	OpDeleteZone:       "delete DNS zone",
	OpListRecords:      "list DNS records",
	OpGetRecord:        "get DNS record",
	OpCreateRecord:     "create DNS record",
	OpUpdateRecord:     "update DNS record",
	OpDeleteRecord:     "delete DNS record",
	OpBulkCreateRecords: "bulk create DNS records",
	OpBulkDeleteRecords: "bulk delete DNS records",

	// IPNS operations
	OpListIPNSKeys:  "list IPNS keys",
	OpGetIPNSKey:    "get IPNS key",
	OpCreateIPNSKey: "create IPNS key",
	OpDeleteIPNSKey: "delete IPNS key",
	OpPublishIPNS:   "publish to IPNS",
	OpRepublishIPNS: "republish to IPNS",
	OpResolveIPNS:   "resolve IPNS name",

	// Websites operations
	OpListWebsites:   "list websites",
	OpGetWebsite:     "get website",
	OpCreateWebsite:  "create website",
	OpUpdateWebsite:  "update website",
	OpDeleteWebsite:  "delete website",
	OpValidateWebsite: "validate website",
	OpGetSSLStatus:   "get SSL status",
}

// Named error types for error comparison.
var (
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = fmt.Errorf("unauthorized")
)

// errorFactory is a helper for creating errors with optional wrapping.
type errorFactory struct {
	wrapErr  bool
	message  string
}

// Error creates the actual error.
func (ef errorFactory) Error() error {
	if ef.wrapErr {
		return fmt.Errorf("%w: %s", ErrUnauthorized, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// authErr creates an error factory that wraps with ErrUnauthorized.
func authErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, message: msg}
}

// plainErr creates an error factory without wrapping.
func plainErr(msg string) errorFactory {
	return errorFactory{wrapErr: false, message: msg}
}

const defaultOperationName = "operation"

// opsString returns the operation name for the given operation ID.
func opsString(op int) string {
	if opName, ok := operationString[op]; ok {
		return opName
	}
	return defaultOperationName
}

// ErrBadRequest creates a bad request error with the given message.
func ErrBadRequest(msg string) error {
	return fmt.Errorf("%s", msg)
}

// httpErrorMessages maps operation IDs to their custom status code error messages.
// This provides a centralized, DRY way to handle HTTP error responses.
var httpErrorMessages = map[int]map[int]errorFactory{
	// DNS operations - all use similar patterns
	OpListZones: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpCreateZone: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid zone data"),
		http.StatusConflict:     plainErr("zone already exists"),
	},
	OpGetZone: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("zone not found"),
	},
	OpDeleteZone: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("zone not found"),
	},
	OpListRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("record not found"),
	},
	OpCreateRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record data"),
	},
	OpUpdateRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record data"),
		http.StatusNotFound:     plainErr("record not found"),
	},
	OpDeleteRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("record not found"),
	},
	OpBulkCreateRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid records data"),
	},
	OpBulkDeleteRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record identifiers"),
	},

	// IPNS operations
	OpListIPNSKeys: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("IPNS key not found"),
	},
	OpCreateIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid key data"),
	},
	OpDeleteIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("IPNS key not found"),
	},
	OpPublishIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid publish data"),
	},
	OpRepublishIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("no keys to republish"),
	},
	OpResolveIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("IPNS name not found"),
	},

	// Websites operations
	OpListWebsites: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("website not found"),
	},
	OpCreateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid website data"),
		http.StatusConflict:     plainErr("website already exists"),
	},
	OpUpdateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid website data"),
		http.StatusNotFound:     plainErr("website not found"),
	},
	OpDeleteWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("website not found"),
	},
	OpValidateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("website not found"),
		http.StatusBadRequest:   plainErr("validation failed"),
	},
	OpGetSSLStatus: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     plainErr("website not found"),
	},
}

// isUnrecoverable returns true if the given status code should not be retried.
// Client errors (4xx) are typically unrecoverable, except for 429 (rate limiting).
func isUnrecoverable(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusConflict,
		http.StatusUnprocessableEntity:
		return true
	default:
		return statusCode >= 400 && statusCode < 500 && statusCode != http.StatusTooManyRequests
	}
}

// handleResponse processes an HTTP response using the global error message map.
//
// op: the operation ID (used to lookup custom error messages)
// statusCode: the HTTP status code from the response
// body: the response body
// successCodes: status codes that indicate success (e.g., []int{http.StatusOK})
//
// Returns nil for success codes, unrecoverable error for client errors (to stop retries),
// or retryable error for server errors.
func handleResponse(statusCode int, body []byte, op int, successCodes []int) error {
	// Check if the status code indicates success
	for _, code := range successCodes {
		if statusCode == code {
			return nil
		}
	}

	// Check for custom error message in global map
	var err error
	if errorMessages, ok := httpErrorMessages[op]; ok {
		if factory, ok := errorMessages[statusCode]; ok {
			err = factory.Error()
		}
	}

	if err == nil {
		// Get operation name for generic error
		opName := operationString[op]
		if opName == "" {
			opName = defaultOperationName
		}

		// Generic error with body
		err = fmt.Errorf("%s failed with status %d: %s", opName, statusCode, string(body))
	}

	// Mark client errors as unrecoverable to prevent retries
	if isUnrecoverable(statusCode) {
		return retry.Unrecoverable(err)
	}

	return err
}
