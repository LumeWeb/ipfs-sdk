package ipfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/avast/retry-go/v4"
	client "go.lumeweb.com/ipfs-sdk/internal/client"
	backend "go.lumeweb.com/ipfs-sdk/internal/download"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
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
	OpGetZoneStatus
	OpValidateZone
	OpPushCert
	OpUpdateTLSA
	OpGetCert

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
	OpUpdateSSLStatusInternal
	OpGetGatewayWebsite
	OpGetGatewayWebsiteStatus
	OpGetWebsiteConfig

	// Website domain binding operations
	OpListWebsiteDomains
	OpBindWebsiteDomain
	OpUpdateWebsiteDomain
	OpUnbindWebsiteDomain
	OpVerifyWebsiteDomain
	OpDNSRequirementsWebsiteDomain
	OpRepublishDomainDANE
	OpListPlatformDomains
	OpCheckPlatformDomainAvailability

	// Pinning operations
	OpListPins
	OpAddPin
	OpGetPin
	OpRemovePin

	// Ping operations
	OpPing

	// DAG operations
	OpResolveDAG
)

// operationString maps operation IDs to human-readable names.
var operationString = map[int]string{
	// DNS operations
	OpListZones:         "list DNS zones",
	OpCreateZone:        "create DNS zone",
	OpGetZone:           "get DNS zone",
	OpDeleteZone:        "delete DNS zone",
	OpListRecords:       "list DNS records",
	OpGetRecord:         "get DNS record",
	OpCreateRecord:      "create DNS record",
	OpUpdateRecord:      "update DNS record",
	OpDeleteRecord:      "delete DNS record",
	OpBulkCreateRecords: "bulk create DNS records",
	OpBulkDeleteRecords: "bulk delete DNS records",
	OpGetZoneStatus:     "get DNS zone status",
	OpValidateZone:      "validate DNS zone",
	OpPushCert:          "push DNS certificate",
	OpUpdateTLSA:        "update DNS TLSA record",
	OpGetCert:           "get DNS certificate",

	// IPNS operations
	OpListIPNSKeys:  "list IPNS keys",
	OpGetIPNSKey:    "get IPNS key",
	OpCreateIPNSKey: "create IPNS key",
	OpDeleteIPNSKey: "delete IPNS key",
	OpPublishIPNS:   "publish to IPNS",
	OpRepublishIPNS: "republish to IPNS",
	OpResolveIPNS:   "resolve IPNS name",

	// Websites operations
	OpListWebsites:            "list websites",
	OpGetWebsite:              "get website",
	OpCreateWebsite:           "create website",
	OpUpdateWebsite:           "update website",
	OpDeleteWebsite:           "delete website",
	OpValidateWebsite:         "validate website",
	OpGetSSLStatus:            "get SSL status",
	OpUpdateSSLStatusInternal: "update SSL status internal",
	OpGetGatewayWebsite:       "get gateway website",
	OpGetGatewayWebsiteStatus: "get gateway website status",
	OpGetWebsiteConfig:        "get website config",

	// Website domain binding operations
	OpListWebsiteDomains:              "list website domains",
	OpBindWebsiteDomain:               "bind website domain",
	OpUpdateWebsiteDomain:             "update website domain",
	OpUnbindWebsiteDomain:             "unbind website domain",
	OpVerifyWebsiteDomain:             "verify website domain",
	OpDNSRequirementsWebsiteDomain:    "get domain DNS requirements",
	OpRepublishDomainDANE:             "republish domain DANE records",
	OpListPlatformDomains:             "list platform domains",
	OpCheckPlatformDomainAvailability: "check platform domain availability",

	// Pinning operations
	OpListPins:  "list pins",
	OpAddPin:    "add pin",
	OpGetPin:    "get pin",
	OpRemovePin: "remove pin",

	// Ping operations
	OpPing: "ping",

	// DAG operations
	OpResolveDAG: "resolve DAG",
}

// Named error types for error comparison.
var (
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrForbidden is returned when the caller is authenticated but not
	// authorized for the requested resource (e.g. another domain's private key).
	ErrForbidden = errors.New("forbidden")

	// ErrGone is returned when a resource exists but is in a deleted or broken state.
	ErrGone = errors.New("gone")

	// ErrRateLimitExceeded is returned when a download rate limit is exceeded.
	// This error is returned by download operations when the configured rate limiter
	// denies a request due to too many concurrent operations or excessive bandwidth usage.
	// Re-exported from internal/download package for error matching with errors.Is().
	ErrRateLimitExceeded = backend.ErrRateLimitExceeded
)

// errorFactory is a helper for creating errors with optional wrapping.
type errorFactory struct {
	wrapErr  bool
	sentinel error
	message  string
}

// Error creates the actual error.
func (ef errorFactory) Error() error {
	if ef.wrapErr && ef.sentinel != nil {
		return fmt.Errorf("%w: %s", ef.sentinel, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// authErr creates an error factory that wraps with ErrUnauthorized.
func authErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, sentinel: ErrUnauthorized, message: msg}
}

// notFoundErr creates an error factory that wraps with ErrNotFound.
func notFoundErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, sentinel: ErrNotFound, message: msg}
}

// forbiddenErr creates an error factory that wraps with ErrForbidden.
func forbiddenErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, sentinel: ErrForbidden, message: msg}
}

// goneErr creates an error factory that wraps with ErrGone.
func goneErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, sentinel: ErrGone, message: msg}
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

// Backend error reason codes returned by the API in ErrorDetail.Reason.
// These distinguish otherwise-identical HTTP error responses so callers can
// handle each failure mode explicitly.
const (
	// ErrorCodeCIDNotPinned is returned when a request targets a CID that is
	// not pinned on the gateway.
	ErrorCodeCIDNotPinned = "CID_NOT_PINNED"

	// ErrorCodeIPNSKeyNotFound is returned when a request targets an IPNS key
	// that does not exist.
	ErrorCodeIPNSKeyNotFound = "IPNS_KEY_NOT_FOUND"

	// ErrorCodeDNSValidationFailed is returned when domain DNS validation fails
	// (e.g. verification TXT record not published or domain does not resolve).
	ErrorCodeDNSValidationFailed = "DNS_VALIDATION_FAILED"
)

// APIError carries the machine-readable reason code and human-readable details
// returned by the backend alongside the underlying error. It preserves the
// underlying error chain so errors.Is/errors.As still match sentinels like
// ErrNotFound.
type APIError struct {
	// Reason is the backend machine-readable error code (e.g. CID_NOT_PINNED).
	Reason string

	// Details is the optional human-readable detail message from the backend.
	Details string

	// Err is the wrapped underlying error.
	Err error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("api error (reason: %s)", e.Reason)
	}
	if e.Reason != "" {
		return fmt.Sprintf("%s (reason: %s)", e.Err, e.Reason)
	}
	return e.Err.Error()
}

// Unwrap returns the wrapped underlying error for errors.Is/As support.
func (e *APIError) Unwrap() error {
	return e.Err
}

// ErrorReasonOf extracts the backend reason code from err, or returns "" if the
// error does not carry one.
func ErrorReasonOf(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Reason
	}
	return ""
}

// withReason wraps err with the backend ErrorResponse parsed from body, if the
// body contains a non-empty machine-readable reason code. If err is nil or no
// code is present, err is returned unchanged.
func withReason(err error, body []byte) error {
	if err == nil || len(body) == 0 {
		return err
	}

	var resp client.ErrorResponse
	if e := json.Unmarshal(body, &resp); e != nil || resp.Error.Reason == "" {
		return err
	}

	apiErr := &APIError{
		Reason: resp.Error.Reason,
		Err:    err,
	}
	if resp.Error.Details != nil {
		apiErr.Details = *resp.Error.Details
	}

	return apiErr
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
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpDeleteZone: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpListRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("record not found"),
	},
	OpCreateRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record data"),
	},
	OpUpdateRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record data"),
		http.StatusNotFound:     notFoundErr("record not found"),
	},
	OpDeleteRecord: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("record not found"),
	},
	OpBulkCreateRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid records data"),
	},
	OpBulkDeleteRecords: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid record identifiers"),
	},
	OpGetZoneStatus: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpValidateZone: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpPushCert: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid certificate data"),
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpUpdateTLSA: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid TLSA data"),
		http.StatusNotFound:     notFoundErr("zone not found"),
	},
	OpGetCert: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusForbidden:    forbiddenErr("not authorized for this domain"),
		http.StatusNotFound:     notFoundErr("certificate not found"),
	},

	// IPNS operations
	OpListIPNSKeys: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("IPNS key not found"),
	},
	OpCreateIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid key data"),
	},
	OpDeleteIPNSKey: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("IPNS key not found"),
	},
	OpPublishIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid publish data"),
	},
	OpRepublishIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("no keys to republish"),
	},
	OpResolveIPNS: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("IPNS name not found"),
	},

	// Websites operations
	OpListWebsites: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
		http.StatusGone:         goneErr("website is broken"),
	},
	OpCreateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid website data"),
		http.StatusConflict:     plainErr("website already exists"),
	},
	OpUpdateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid website data"),
		http.StatusNotFound:     notFoundErr("website not found"),
	},
	OpDeleteWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
	},
	OpValidateWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
		http.StatusBadRequest:   plainErr("validation failed"),
	},
	OpGetSSLStatus: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
	},
	OpUpdateSSLStatusInternal: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid SSL status data"),
		http.StatusNotFound:     notFoundErr("website not found"),
	},
	OpGetWebsiteConfig: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website config not found"),
	},

	// Website domain binding operations
	OpListWebsiteDomains: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
	},
	OpBindWebsiteDomain: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid domain data"),
		http.StatusNotFound:     notFoundErr("website not found"),
		http.StatusConflict:     plainErr("domain already bound"),
	},
	OpUpdateWebsiteDomain: {
		http.StatusUnauthorized:        authErr("authentication required"),
		http.StatusBadRequest:          plainErr("invalid request"),
		http.StatusNotFound:            notFoundErr("domain or website not found"),
		http.StatusUnprocessableEntity: plainErr("validation failed"),
	},
	OpUnbindWebsiteDomain: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("domain or website not found"),
	},
	OpVerifyWebsiteDomain: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("domain or website not found"),
		http.StatusBadRequest:   plainErr("verification failed"),
	},
	OpDNSRequirementsWebsiteDomain: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("domain or website not found"),
	},
	OpRepublishDomainDANE: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("domain or website not found"),
		http.StatusBadRequest:   plainErr("invalid request"),
		http.StatusConflict:     plainErr("no DANE records to republish (no stored certificate or managed zone)"),
	},
	OpCheckPlatformDomainAvailability: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusBadRequest:   plainErr("invalid label"),
	},
	OpListPlatformDomains: {
		http.StatusUnauthorized: authErr("authentication required"),
	},
	OpGetGatewayWebsite: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
		http.StatusGone:         goneErr("website is broken or deleted"),
	},
	OpGetGatewayWebsiteStatus: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("website not found"),
		http.StatusGone:         goneErr("website is broken or deleted"),
	},

	// Ping operations
	OpPing: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusForbidden:    plainErr("gateway secret required"),
	},

	// DAG operations
	OpResolveDAG: {
		http.StatusUnauthorized: authErr("authentication required"),
		http.StatusNotFound:     notFoundErr("CID not found"),
	},
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

	// Attach the backend machine-readable reason code (if present) so callers
	// can distinguish otherwise-identical HTTP error responses (e.g.
	// CID_NOT_PINNED vs IPNS_KEY_NOT_FOUND vs DNS_VALIDATION_FAILED).
	err = withReason(err, body)

	// Mark client errors as unrecoverable to prevent retries
	if httputil.IsUnrecoverable(statusCode) {
		return retry.Unrecoverable(err)
	}

	return err
}
