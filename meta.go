package ipfs

import (
	"context"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Meta types from the generated client
type CIDExportResponse = internalclient.CIDExportResponse
type SharedObject = internalclient.SharedObject
type SlabSlice = internalclient.SlabSlice
type PinnedSector = internalclient.PinnedSector
type DAGExportResponse = internalclient.DAGExportResponse
type DAGBlock = internalclient.DAGBlock
type DAGLink = internalclient.DAGLink

// MetaClientWithResponsesInterface defines the methods needed from the generated internal client for Meta
type MetaClientWithResponsesInterface interface {
	GetApiExportCidCidSiaObjectWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiExportCidCidSiaObjectResponse, error)
	GetApiExportCidCidDagWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiExportCidCidDagResponse, error)
}

// internalClientToMetaAdapter adapts internalclient.ClientWithResponses to MetaClientWithResponsesInterface
type internalClientToMetaAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToMetaAdapter) GetApiExportCidCidSiaObjectWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiExportCidCidSiaObjectResponse, error) {
	return a.client.GetApiExportCidCidSiaObjectWithResponse(ctx, cid, reqEditors...)
}

func (a *internalClientToMetaAdapter) GetApiExportCidCidDagWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiExportCidCidDagResponse, error) {
	return a.client.GetApiExportCidCidDagWithResponse(ctx, cid, reqEditors...)
}

// ConvertClientToMeta converts a ClientWithResponses to MetaClientWithResponsesInterface
func ConvertClientToMeta(client *internalclient.ClientWithResponses) MetaClientWithResponsesInterface {
	return &internalClientToMetaAdapter{client: client}
}

// MetaConfig holds configuration for Meta service operations
type MetaConfig struct {
	Client MetaClientWithResponsesInterface
	Retry  RetryConfig
}

// MetaOption configures the MetaConfig
type MetaOption func(*MetaConfig)

// WithMetaClient sets a custom client for the Meta service
func WithMetaClient(client MetaClientWithResponsesInterface) MetaOption {
	return func(c *MetaConfig) {
		c.Client = client
	}
}

// WithMetaRetry sets the retry configuration for the Meta service
func WithMetaRetry(rc httputil.RetryConfig) MetaOption {
	return func(c *MetaConfig) {
		c.Retry = rc
	}
}

// DefaultMetaConfig returns the default Meta configuration
func DefaultMetaConfig() MetaConfig {
	return MetaConfig{
		Retry: DefaultRetryConfig(),
	}
}

// MetaService provides export functionality for CID metadata
type MetaService interface {
	// ExportSiaObject exports the Sia object for a CID, including the shared object structure
	// with slab and sector details.
	ExportSiaObject(ctx context.Context, cid string) (*CIDExportResponse, error)

	// ExportDAG exports a complete DAG for a root CID, including all blocks, their sizes,
	// links, and associated Sia objects.
	ExportDAG(ctx context.Context, cid string) (*DAGExportResponse, error)
}

// metaService implements MetaService using the generated internal client
type metaService struct {
	client MetaClientWithResponsesInterface
	config MetaConfig
}

// NewMetaService creates a Meta service from a client interface with options
func NewMetaService(genClient MetaClientWithResponsesInterface, opts ...MetaOption) MetaService {
	cfg := DefaultMetaConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &metaService{
		client: genClient,
		config: cfg,
	}
}

// ExportSiaObject exports the Sia object for a CID.
func (s *metaService) ExportSiaObject(ctx context.Context, cid string) (*CIDExportResponse, error) {
	var result *CIDExportResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiExportCidCidSiaObjectWithResponse(ctx, cid)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpExportSiaObject, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpExportSiaObject) + " no response data for CID " + cid)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExportDAG exports a complete DAG for a root CID.
func (s *metaService) ExportDAG(ctx context.Context, cid string) (*DAGExportResponse, error) {
	var result *DAGExportResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiExportCidCidDagWithResponse(ctx, cid)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpExportDAG, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpExportDAG) + " no response data for CID " + cid)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
