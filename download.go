package ipfs

import (
	"context"
	"fmt"
	"io"
	"net/http"

	blockstore "github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/gateway"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// DownloadService provides functionality for downloading IPFS blocks and content
// from the gateway using the boxo gateway patterns.
type DownloadService struct {
	blockstore blockstore.Blockstore
	httpClient *http.Client
	authToken  string
	baseURL    string
}

// DownloadServiceOption configures the DownloadService.
type DownloadServiceOption func(*DownloadService)

// WithDownloadHTTPClient sets a custom HTTP client for the download service.
// This allows injecting a custom HTTP client with specific timeout and transport settings.
func WithDownloadHTTPClient(client *http.Client) DownloadServiceOption {
	return func(s *DownloadService) {
		s.httpClient = client
	}
}

// NewDownloadService creates a new DownloadService.
// baseURL is the base URL of the API server (e.g., "https://api.example.com").
// authToken is the authentication bearer token.
func NewDownloadService(baseURL, authToken string, opts ...DownloadServiceOption) (*DownloadService, error) {
	s := &DownloadService{
		baseURL:   baseURL,
		authToken: authToken,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create HTTP client with auth injection if not provided
	if s.httpClient == nil {
		s.httpClient = &http.Client{
			Transport: httputil.NewAuthRoundTripper(http.DefaultTransport, authToken),
		}
	}

	// Create remote blockstore using boxo's NewRemoteBlockstore
	// This will use the HTTP client to fetch blocks via /ipfs/{cid}?format=raw
	remoteStore, err := gateway.NewRemoteBlockstore([]string{baseURL}, s.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote blockstore: %w", err)
	}

	s.blockstore = remoteStore

	return s, nil
}

// Block downloads a single IPFS block by CID.
// Returns the block data with CID validation.
func (s *DownloadService) Block(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	return s.blockstore.Get(ctx, c)
}

// Has checks if a block exists in the blockstore.
func (s *DownloadService) Has(ctx context.Context, c cid.Cid) (bool, error) {
	return s.blockstore.Has(ctx, c)
}

// BlockSize returns the size of a block by CID without fetching the block data.
// Returns -1 if the block does not exist.
func (s *DownloadService) BlockSize(ctx context.Context, c cid.Cid) (int, error) {
	return s.blockstore.GetSize(ctx, c)
}

// Raw downloads an IPFS block and returns the raw byte data.
// Useful when you only need the data without the CID wrapper.
func (s *DownloadService) Raw(ctx context.Context, c cid.Cid) ([]byte, error) {
	block, err := s.Block(ctx, c)
	if err != nil {
		return nil, err
	}
	return block.RawData(), nil
}

// CopyBlock writes a block to an io.Writer.
// Useful for streaming the block data directly to a file or network connection.
func (s *DownloadService) CopyBlock(ctx context.Context, c cid.Cid, w io.Writer) error {
	block, err := s.Block(ctx, c)
	if err != nil {
		return err
	}
	_, err = w.Write(block.RawData())
	return err
}

// SetAuthToken updates the authentication token used for requests.
// This is thread-safe and can be called at any time.
func (s *DownloadService) SetAuthToken(token string) {
	s.authToken = token
	if art, ok := s.httpClient.Transport.(*httputil.AuthRoundTripper); ok {
		art.SetAuthToken(token)
	}
}

// AuthToken returns the current authentication token.
func (s *DownloadService) AuthToken() string {
	return s.authToken
}
