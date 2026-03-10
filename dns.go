package ipfs

import (
	"context"
	"net/http"

	dnsreq "go.lumeweb.com/ipfs-sdk/internal/dnsreq"
	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// DNS types from dnsreq package to avoid import cycles
type ZoneListResponse = dnsreq.ZoneListResponse
type ZoneResponse = dnsreq.ZoneResponse
type ZoneRequest = dnsreq.ZoneRequest
type RecordResponse = dnsreq.RecordResponse
type RecordRequest = dnsreq.RecordRequest
type RecordIdentifier = dnsreq.RecordIdentifier
type RecordResult = dnsreq.RecordResult
type BulkRecordRequest = dnsreq.BulkRecordRequest
type BulkDeleteRequest = dnsreq.BulkDeleteRequest
type ErrorResponse = dnsreq.ErrorResponse

// DNSConfig holds configuration for DNS service operations
type DNSConfig struct {
	Retry   httputil.RetryConfig
	Client  DNSClientWithResponsesInterface
}

// DefaultDNSConfig returns default configuration for DNS service
func DefaultDNSConfig() DNSConfig {
	return DNSConfig{
		Retry: httputil.DefaultRetryConfig(),
	}
}

// DNSOption applies configuration to DNSConfig
type DNSOption func(*DNSConfig)

// WithDNSRetry sets the retry configuration for DNS operations
func WithDNSRetry(cfg httputil.RetryConfig) DNSOption {
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

// DNSService provides DNS management functionality
type DNSService interface {
	// Zone management
	ListZones(ctx context.Context) ([]ZoneListResponse, error)
	GetZone(ctx context.Context, id string) (*ZoneResponse, error)
	CreateZone(ctx context.Context, domain string, nameservers []string) (*ZoneResponse, error)
	DeleteZone(ctx context.Context, id string) error

	// Record management
	ListRecords(ctx context.Context, zoneID string) ([]RecordResponse, error)
	CreateRecord(ctx context.Context, zoneID string, record RecordRequest) (*RecordResponse, error)
	GetRecord(ctx context.Context, zoneID string, name string, recordType string) (*RecordResponse, error)
	UpdateRecord(ctx context.Context, zoneID string, name string, recordType string, record RecordRequest) (*RecordResponse, error)
	DeleteRecord(ctx context.Context, zoneID string, name string, recordType string) error

	// Bulk operations
	BulkCreateRecords(ctx context.Context, zoneID string, records []RecordRequest) ([]RecordResponse, error)
	BulkDeleteRecords(ctx context.Context, zoneID string, identifiers []RecordIdentifier) ([]RecordResult, error)
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
	var result []internalclient.ZoneListResponse
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDnsZonesWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListZones, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			result = []internalclient.ZoneListResponse{}
			return nil
		}

		result = *resp.JSON200
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
			result = nil
			return ErrBadRequest(opsString(OpGetZone) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateZone creates a new DNS zone for a domain
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

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateZone, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			result = nil
			return ErrBadRequest(opsString(OpCreateZone) + " no response data")
		}

		result = resp.JSON201
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteZone removes a DNS zone and all its records
func (s *dnsService) DeleteZone(ctx context.Context, id string) error {
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiDnsZonesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDeleteZone, []int{http.StatusOK, http.StatusNoContent}); err != nil {
			return err
		}

		return nil
	})
	return err
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

		if resp.JSON200 == nil {
			result = []RecordResponse{}
			return nil
		}

		// Return the single record from the response wrapped in a slice
		// The API returns one record per call with Total indicating pagination support
		result = []RecordResponse{resp.JSON200.Data}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateRecord creates a new DNS record in a zone
func (s *dnsService) CreateRecord(ctx context.Context, zoneID string, record RecordRequest) (*RecordResponse, error) {
	var result *RecordResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiDnsZonesIdRecordsWithResponse(ctx, zoneID, record)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateRecord, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			result = nil
			return ErrBadRequest(opsString(OpCreateRecord) + " no response data")
		}

		result = resp.JSON201
		return nil
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
			result = nil
			return ErrBadRequest(opsString(OpGetRecord) + " no response data")
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
			result = nil
			return ErrBadRequest(opsString(OpUpdateRecord) + " no response data")
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
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiDnsZonesIdRecordsNameTypeWithResponse(ctx, zoneID, name, recordType)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDeleteRecord, []int{http.StatusOK, http.StatusNoContent}); err != nil {
			return err
		}

		return nil
	})
	return err
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
