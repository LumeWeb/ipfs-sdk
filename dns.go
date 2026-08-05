package ipfs

import (
	"context"
	"encoding/json"
	"net/http"

	dnsreq "go.lumeweb.com/ipfs-sdk/internal/dnsreq"
	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// DNS types from dnsreq package to avoid import cycles
type ZoneListResponse = dnsreq.ZoneListResponse
type ZoneResponse = dnsreq.ZoneResponse
type ZoneRequest = dnsreq.ZoneRequest
type RecordRequest = dnsreq.RecordRequest
type RecordResponse = dnsreq.RecordResponse
type RecordIdentifier = dnsreq.RecordIdentifier
type RecordResult = dnsreq.RecordResult
type BulkRecordRequest = dnsreq.BulkRecordRequest
type BulkDeleteRequest = dnsreq.BulkDeleteRequest

// DNS cert/TLSA types from generated client
type CertPushRequest = internalclient.CertPushRequest
type CertPushResponse = internalclient.CertPushResponse
type CertGetResponse = internalclient.CertGetResponse
type TLSAUpdateRequest = internalclient.TLSAUpdateRequest

// DNSConfig holds configuration for DNS service operations
type DNSConfig struct {
	Retry  RetryConfig
	Client DNSClientWithResponsesInterface
}

// DefaultDNSConfig returns default configuration for DNS service
func DefaultDNSConfig() DNSConfig {
	return DNSConfig{
		Retry: DefaultRetryConfig(),
	}
}

// DNSOption applies configuration to DNSConfig
type DNSOption func(*DNSConfig)

// WithDNSRetry sets the retry configuration for DNS operations
func WithDNSRetry(cfg RetryConfig) DNSOption {
	return func(c *DNSConfig) {
		c.Retry = cfg
	}
}

// WithDNSClient sets the client interface for DNS operations
// This allows overriding the internal generated client for testing or mocking
func WithDNSClient(client DNSClientWithResponsesInterface) DNSOption {
	return func(c *DNSConfig) {
		c.Client = client
	}
}

// Type aliases for DNS validation response
type ValidationResponse = internalclient.ValidationResponse

// DNSService provides DNS management functionality
type DNSService interface {
	// Zone management
	ListZones(ctx context.Context) ([]ZoneListResponse, error)
	GetZone(ctx context.Context, id string) (*ZoneResponse, error)
	CreateZone(ctx context.Context, domain string, nameservers []string) (*ZoneResponse, error)
	DeleteZone(ctx context.Context, id string) error
	GetZoneStatus(ctx context.Context, id string) (*ZoneResponse, error)
	ValidateZone(ctx context.Context, id string) (*ValidationResponse, error)

	// Record management
	ListRecords(ctx context.Context, zoneID string) ([]RecordResponse, error)
	CreateRecord(ctx context.Context, zoneID string, record RecordRequest) (*RecordResponse, error)
	GetRecord(ctx context.Context, zoneID string, name string, recordType string) (*RecordResponse, error)
	UpdateRecord(ctx context.Context, zoneID string, name string, recordType string, record RecordRequest) (*RecordResponse, error)
	DeleteRecord(ctx context.Context, zoneID string, name string, recordType string) error

	// Bulk operations
	BulkCreateRecords(ctx context.Context, zoneID string, records []RecordRequest) ([]RecordResponse, error)
	BulkDeleteRecords(ctx context.Context, zoneID string, identifiers []RecordIdentifier) ([]RecordResult, error)

	// Polling/wait methods

	// Certificate/TLSA management (internal endpoints for Caddy)
	PushCert(ctx context.Context, req CertPushRequest) (*CertPushResponse, error)
	UpdateTLSA(ctx context.Context, req TLSAUpdateRequest) (*CertPushResponse, error)

	// GetCert fetches an existing DANE certificate + key for a domain. Returns a
	// not-found error when the portal has no stored identity yet.
	GetCert(ctx context.Context, domain string, namespace string) (*CertGetResponse, error)
}

// dnsService implements DNSService using the generated internal client
type dnsService struct {
	client DNSClientWithResponsesInterface
	config DNSConfig
}

// NewDNSService creates a DNS service from a client interface with options
func NewDNSService(genClient DNSClientWithResponsesInterface, opts ...DNSOption) DNSService {
	cfg := DefaultDNSConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &dnsService{client: genClient, config: cfg}
}

// NewDNSServiceFromClient creates a DNS service from a ClientWithResponses instance
func NewDNSServiceFromClient(genClient *internalclient.ClientWithResponses, opts ...DNSOption) DNSService {
	return NewDNSService(genClient, opts...)
}

// ListZones retrieves all DNS zones for the authenticated user
func (s *dnsService) ListZones(ctx context.Context) ([]ZoneListResponse, error) {
	var result []ZoneListResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListZones, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Data == nil {
			result = []ZoneListResponse{}
			return nil
		}

		result = resp.JSON200.Data
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetZone retrieves a specific DNS zone by ID
func (s *dnsService) GetZone(ctx context.Context, id string) (*ZoneResponse, error) {
	var result *ZoneResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetZone, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetZone) + " no response data for zone " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateZone creates a new DNS zone
func (s *dnsService) CreateZone(ctx context.Context, domain string, nameservers []string) (*ZoneResponse, error) {
	var result *ZoneResponse
	req := ZoneRequest{
		Domain:      domain,
		Nameservers: &nameservers,
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateZone, []int{http.StatusOK, http.StatusCreated}); err != nil {
			return err
		}

		var apiErr error
		result, apiErr = handleCreateResponse(resp.Body, nil, resp.JSON201, OpCreateZone)
		return apiErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteZone deletes a DNS zone by ID
func (s *dnsService) DeleteZone(ctx context.Context, id string) error {
	return httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiDnsZonesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		return handleResponse(resp.StatusCode(), resp.Body, OpDeleteZone, []int{http.StatusOK, http.StatusNoContent})
	})
}

// GetZoneStatus retrieves the status of a specific DNS zone by ID
func (s *dnsService) GetZoneStatus(ctx context.Context, id string) (*ZoneResponse, error) {
	var result *ZoneResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesIdStatusWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetZoneStatus, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetZoneStatus) + " no response data for zone status " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ValidateZone validates a DNS zone by ID
func (s *dnsService) ValidateZone(ctx context.Context, id string) (*ValidationResponse, error) {
	var result *ValidationResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesIdValidateWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpValidateZone, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpValidateZone) + " no response data for zone validation " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListRecords retrieves all DNS records for a zone
func (s *dnsService) ListRecords(ctx context.Context, zoneID string) ([]RecordResponse, error) {
	var result []RecordResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesIdRecordsWithResponse(ctx, zoneID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListRecords, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Data == nil {
			result = []RecordResponse{}
			return nil
		}

		result = resp.JSON200.Data
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleCreateResponse handles responses for create operations that may return either 200 or 201
func handleCreateResponse[T any](body []byte, json200 any, json201 *T, op int) (*T, error) {
	if json201 != nil {
		return json201, nil
	}
	// If 200 was returned, re-parse body as the response type
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ErrBadRequest(opsString(op) + " failed to parse response")
	}
	return &result, nil
}

// CreateRecord creates a new DNS record
func (s *dnsService) CreateRecord(ctx context.Context, zoneID string, record RecordRequest) (*RecordResponse, error) {
	var result *RecordResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesIdRecordsWithResponse(ctx, zoneID, record)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateRecord, []int{http.StatusOK, http.StatusCreated}); err != nil {
			return err
		}

		var apiErr error
		result, apiErr = handleCreateResponse(resp.Body, nil, resp.JSON201, OpCreateRecord)
		return apiErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetRecord retrieves a specific DNS record
func (s *dnsService) GetRecord(ctx context.Context, zoneID string, name string, recordType string) (*RecordResponse, error) {
	var result *RecordResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesIdRecordsNameTypeWithResponse(ctx, zoneID, name, recordType)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetRecord, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetRecord) + " no response data for record " + name)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateRecord updates an existing DNS record
func (s *dnsService) UpdateRecord(ctx context.Context, zoneID string, name string, recordType string, record RecordRequest) (*RecordResponse, error) {
	var result *RecordResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PutApiDnsZonesIdRecordsNameTypeWithResponse(ctx, zoneID, name, recordType, record)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpUpdateRecord, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpUpdateRecord) + " no response data for update " + name)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteRecord deletes a DNS record
func (s *dnsService) DeleteRecord(ctx context.Context, zoneID string, name string, recordType string) error {
	return httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiDnsZonesIdRecordsNameTypeWithResponse(ctx, zoneID, name, recordType)
		if err != nil {
			return err
		}

		return handleResponse(resp.StatusCode(), resp.Body, OpDeleteRecord, []int{http.StatusOK, http.StatusNoContent})
	})
}

// BulkCreateRecords creates multiple DNS records at once
func (s *dnsService) BulkCreateRecords(ctx context.Context, zoneID string, records []RecordRequest) ([]RecordResponse, error) {
	if len(records) == 0 {
		return []RecordResponse{}, nil
	}

	var result []RecordResponse
	bulkReq := BulkRecordRequest{
		Records: records,
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesIdRecordsBulkWithResponse(ctx, zoneID, bulkReq)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpBulkCreateRecords, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Records == nil {
			result = []RecordResponse{}
			return nil
		}

		result = resp.JSON200.Records
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// BulkDeleteRecords deletes multiple DNS records at once
func (s *dnsService) BulkDeleteRecords(ctx context.Context, zoneID string, identifiers []RecordIdentifier) ([]RecordResult, error) {
	if len(identifiers) == 0 {
		return []RecordResult{}, nil
	}

	var result []RecordResult
	bulkReq := BulkDeleteRequest{
		Records: identifiers,
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesIdRecordsBulkDeleteWithResponse(ctx, zoneID, bulkReq)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpBulkDeleteRecords, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Results == nil {
			result = []RecordResult{}
			return nil
		}

		result = resp.JSON200.Results
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// PushCert pushes a certificate to the DNS service and receives a computed TLSA record.
// This is an internal endpoint typically called by Caddy.
func (s *dnsService) PushCert(ctx context.Context, req CertPushRequest) (*CertPushResponse, error) {
	var result *CertPushResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostInternalDnsCertWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpPushCert, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpPushCert) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetCert fetches an existing DANE certificate and key for a domain.
// This is an internal endpoint typically called by Caddy so it can re-issue a
// certificate around a persisted key (stable SPKI). Returns ErrNotFound when
// the portal has no stored identity for the domain yet.
func (s *dnsService) GetCert(ctx context.Context, domain string, namespace string) (*CertGetResponse, error) {
	var result *CertGetResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		params := internalclient.GetInternalDnsCertDomainParams{}
		if namespace != "" {
			params.Namespace = &namespace
		}

		resp, err := s.client.GetInternalDnsCertDomainWithResponse(ctx, domain, &params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetCert, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetCert) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateTLSA pushes a computed TLSA record to the DNS service.
// This is an internal endpoint typically called by Caddy.
func (s *dnsService) UpdateTLSA(ctx context.Context, req TLSAUpdateRequest) (*CertPushResponse, error) {
	var result *CertPushResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostInternalDnsTlsaWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpUpdateTLSA, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpUpdateTLSA) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}