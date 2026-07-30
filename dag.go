package ipfs

import (
	"context"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// DAG types from the generated client
type DAGResponse = internalclient.DAGResponse
type DAGBlockNodeResponse = internalclient.DAGBlockNodeResponse

// DAGClientWithResponsesInterface defines the methods needed from the generated internal client for DAG
type DAGClientWithResponsesInterface interface {
	GetApiDagCidWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDagCidResponse, error)
}

// internalClientToDAGAdapter adapts internalclient.ClientWithResponses to DAGClientWithResponsesInterface
type internalClientToDAGAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToDAGAdapter) GetApiDagCidWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDagCidResponse, error) {
	return a.client.GetApiDagCidWithResponse(ctx, cid, reqEditors...)
}

// ConvertClientToDAG converts a ClientWithResponses to DAGClientWithResponsesInterface
func ConvertClientToDAG(client *internalclient.ClientWithResponses) DAGClientWithResponsesInterface {
	return &internalClientToDAGAdapter{client: client}
}

// DAGConfig holds configuration for DAG service operations
type DAGConfig struct {
	Client DAGClientWithResponsesInterface
	Retry  RetryConfig
}

// DAGOption configures the DAGConfig
type DAGOption func(*DAGConfig)

// WithDAGClient sets a custom client for the DAG service
func WithDAGClient(client DAGClientWithResponsesInterface) DAGOption {
	return func(c *DAGConfig) {
		c.Client = client
	}
}

// WithDAGRetry sets the retry configuration for the DAG service
func WithDAGRetry(rc httputil.RetryConfig) DAGOption {
	return func(c *DAGConfig) {
		c.Retry = rc
	}
}

// DefaultDAGConfig returns the default DAG configuration
func DefaultDAGConfig() DAGConfig {
	return DAGConfig{
		Retry: DefaultRetryConfig(),
	}
}

// DAGService provides DAG resolution functionality
type DAGService interface {
	// ResolveDAG resolves the complete block graph (DAG) for a root CID.
	// Returns all blocks reachable from the root CID, including their sizes and ordered child relationships.
	ResolveDAG(ctx context.Context, cid string) (*DAGResponse, error)
}

// dagService implements DAGService using the generated internal client
type dagService struct {
	client DAGClientWithResponsesInterface
	config DAGConfig
}

// NewDAGService creates a DAG service from a client interface with options
func NewDAGService(genClient DAGClientWithResponsesInterface, opts ...DAGOption) DAGService {
	cfg := DefaultDAGConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &dagService{
		client: genClient,
		config: cfg,
	}
}

// ResolveDAG resolves the complete block graph (DAG) for a root CID in a single query.
// Returns all blocks reachable from the root CID, including their sizes and ordered child relationships.
// Only blocks with ready status are included. Blocks appearing in multiple paths are deduplicated.
func (s *dagService) ResolveDAG(ctx context.Context, cid string) (*DAGResponse, error) {
	var result *DAGResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiDagCidWithResponse(ctx, cid)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpResolveDAG, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpResolveDAG) + " no response data for CID " + cid)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
