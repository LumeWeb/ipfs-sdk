package ipfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/path"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	backend "go.lumeweb.com/ipfs-sdk/internal/download"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
	"go.lumeweb.com/ipfs-content/car"
	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
)

// RateLimiter defines an interface for controlling download rate and availability.
// Implementations check if a download should proceed and optionally enforce rate limits.
type RateLimiter = backend.RateLimiter

// RateLimiterFunc is a function type that implements RateLimiter.
// This allows using simple functions as rate limiters without defining a type.
type RateLimiterFunc = backend.RateLimiterFunc

// DownloadService provides functionality for downloading IPFS blocks and content
// from the gateway using the boxo gateway patterns.
type DownloadService struct {
	backend        backend.Backend
	httpClient     *http.Client
	authTransport  *httputil.AuthRoundTripper
	mu             sync.RWMutex // guards authToken so SetAuthToken can hot-swap it concurrently with AuthToken reads
	authToken      string
	baseURL        string
	rateLimiter    backend.RateLimiter
	workerPoolSize int
	retryConfig    httputil.RetryConfig
	blockMeta      BlockMetaClient
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

// WithDownloadRateLimiter sets a rate limiter for the download service.
// This enables rate-limited downloading with intelligent backoff and retry.
// The rate limiter is called before and during downloads to control download rate.
// All block fetches (including recursive fetches during directory/file iteration) are rate-limited.
func WithDownloadRateLimiter(rl backend.RateLimiter) DownloadServiceOption {
	return func(s *DownloadService) {
		s.rateLimiter = rl
	}
}

// WithDownloadWorkerPoolSize sets the maximum number of concurrent download workers.
// Only applies when a rate limiter is configured.
func WithDownloadWorkerPoolSize(size int) DownloadServiceOption {
	return func(s *DownloadService) {
		s.workerPoolSize = size
	}
}

// WithDownloadRetryConfig sets the retry configuration for download operations.
// Only applies when a rate limiter is configured.
func WithDownloadRetryConfig(cfg httputil.RetryConfig) DownloadServiceOption {
	return func(s *DownloadService) {
		s.retryConfig = cfg
	}
}

// WithInternalGen sets the internal generated client for the download service.
// This provides access to the REST API endpoints.
func WithInternalGen(client *internalclient.ClientWithResponses) DownloadServiceOption {
	return func(s *DownloadService) {
		s.blockMeta = client
	}
}

// blockMetaAdapter adapts BlockMetaClient to download.BlockMetaClient interface.
type blockMetaAdapter struct {
	client BlockMetaClient
}

func (a *blockMetaAdapter) GetBlockSize(ctx context.Context, cid string) (int, error) {
	response, err := a.client.GetApiBlockMetaCidWithResponse(ctx, cid)
	if err != nil {
		return 0, err
	}
	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return 0, fmt.Errorf("block meta API returned status %d", response.StatusCode())
	}
	if response.JSON200 == nil {
		return 0, fmt.Errorf("block meta API returned no data")
	}
	return response.JSON200.BlockSize, nil
}

// WithBlockMetaClient sets a custom block meta client for the download service.
// This is useful for testing with mock implementations.
func WithBlockMetaClient(client BlockMetaClient) DownloadServiceOption {
	return func(s *DownloadService) {
		s.blockMeta = client
	}
}

// blockMetaBackendAdapter adapts BlockMetaClient to backend.BlockMetaClient interface.
type blockMetaBackendAdapter struct {
	client BlockMetaClient
}

func (a *blockMetaBackendAdapter) GetBlockSize(ctx context.Context, c cid.Cid) (int, error) {
	response, err := a.client.GetApiBlockMetaCidWithResponse(ctx, c.String())
	if err != nil {
		return 0, err
	}
	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return 0, fmt.Errorf("block meta API returned status %d", response.StatusCode())
	}
	if response.JSON200 == nil {
		return 0, fmt.Errorf("block meta API returned no data")
	}
	return response.JSON200.BlockSize, nil
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

	// Create or wrap HTTP client with auth injection
	var authTransport *httputil.AuthRoundTripper
	if s.httpClient == nil {
		authTransport = httputil.NewAuthRoundTripper(http.DefaultTransport, authToken)
		s.httpClient = &http.Client{
			Transport: authTransport,
		}
	} else {
		// Wrap existing transport
		authTransport = httputil.NewAuthRoundTripper(s.httpClient.Transport, authToken)
		s.httpClient.Transport = authTransport
	}
	s.authTransport = authTransport

	// Create gateway backend for UnixFS operations
	// Use our rate-limited implementation when a rate limiter is provided
	// Pass the blockMeta client adapter for size queries without rate limiting
	var gatewayBackend gateway.IPFSBackend
	var err error
	
	// Prepare block meta adapter for rate-limited backend
	var metaClient backend.BlockMetaClient
	if s.blockMeta != nil {
		metaClient = &blockMetaBackendAdapter{client: s.blockMeta}
	}
	
	if s.rateLimiter != nil {
		gatewayBackend, err = backend.NewBlocksBackendWithRateLimit([]string{baseURL}, s.httpClient, s.rateLimiter, s.workerPoolSize, s.retryConfig, metaClient)
	} else {
		gatewayBackend, err = gateway.NewRemoteBlocksBackend([]string{baseURL}, s.httpClient)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway backend: %w", err)
	}

	s.backend = gatewayBackend

	return s, nil
}

// Block downloads a single IPFS block by CID.
// Returns the block data with CID validation.
func (s *DownloadService) Block(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	_, file, err := s.backend.GetBlock(ctx, path.FromCid(c))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	
	
	block := blocks.NewBlock(data)
	if !block.Cid().Equals(c) {
		return nil, fmt.Errorf("CID mismatch: expected %s, got %s", c, block.Cid())
	}
	
	return block, nil
}

// Has checks if a block exists in the blockstore.
func (s *DownloadService) Has(ctx context.Context, c cid.Cid) (bool, error) {
	_, headResponse, err := s.backend.Head(ctx, path.FromCid(c))
	_ = headResponse
	if err != nil {
		return false, err
	}
	return true, nil
}


// queryBlockMeta queries the block meta REST API for metadata.
func (s *DownloadService) queryBlockMeta(ctx context.Context, c cid.Cid) (*internalclient.GetApiBlockMetaCidResponse, error) {
	if s.blockMeta == nil {
		return nil, fmt.Errorf("block meta client not initialized")
	}

	// Use the block meta REST API to get UnixFS metadata
	response, err := s.blockMeta.GetApiBlockMetaCidWithResponse(ctx, c.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query block meta API: %w", err)
	}

	if response.StatusCode() < 200 || response.StatusCode() >= 300 {
		return nil, fmt.Errorf("block meta API returned status %d: %s", response.StatusCode(), string(response.Body))
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("block meta API returned no data")
	}

	return response, nil
}

// FileSize returns the actual size of a UnixFS file by CID.
// Uses the block meta REST API to query UnixFS metadata without loading file contents.
// For chunked files, the API returns the total file size by summing all chunk sizes.
// For inline files, returns the actual data size.
func (s *DownloadService) FileSize(ctx context.Context, c cid.Cid) (int64, error) {
	response, err := s.queryBlockMeta(ctx, c)
	if err != nil {
		return -1, err
	}

	return int64(response.JSON200.UnixfsSize), nil
}

// BlockSize returns the size of a block by CID.
// Uses the block meta REST API to query UnixFS metadata without loading file contents.
func (s *DownloadService) BlockSize(ctx context.Context, c cid.Cid) (int, error) {
	response, err := s.queryBlockMeta(ctx, c)
	if err != nil {
		return -1, err
	}

	return response.JSON200.BlockSize, nil
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
	_, file, err := s.backend.GetBlock(ctx, path.FromCid(c))
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = io.Copy(w, file)
	return err
}

// SetAuthToken updates the authentication token used for requests.
// This is thread-safe and can be called at any time.
func (s *DownloadService) SetAuthToken(token string) {
	s.mu.Lock()
	s.authToken = token
	s.mu.Unlock()
	if s.authTransport != nil {
		s.authTransport.SetAuthToken(token)
	}
}

// AuthToken returns the current authentication token.
func (s *DownloadService) AuthToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authToken
}

// DownloadFile downloads a full file from IPFS by CID.
// This method is UnixFS-aware and handles chunked files.
// Returns an io.ReadCloser that should be closed when done.
func (s *DownloadService) DownloadFile(ctx context.Context, c cid.Cid) (io.ReadCloser, error) {
	immutablePath := path.FromCid(c)
	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return nil, err
	}

	return wrapFileNodeAsReadCloser(node, "CID is not a file")
}

// fileReadCloser wraps a files.File to implement io.ReadCloser
type fileReadCloser struct {
	files.File
}

// Close implements io.Closer (delegates to files.File.Close())
func (f *fileReadCloser) Close() error {
	return f.File.Close()
}

// wrapFileNodeAsReadCloser converts a files.Node to io.ReadCloser if it's a file.
// Returns an error if the node is not a file (e.g., it's a directory).
func wrapFileNodeAsReadCloser(node files.Node, errorMessage string) (io.ReadCloser, error) {
	file, ok := node.(files.File)
	if !ok {
		node.Close()
		return nil, fmt.Errorf("%s", errorMessage)
	}
	return &fileReadCloser{File: file}, nil
}

// ListDirectory lists directory entries for a directory CID.
// Returns a slice of directory entries that can be used to access file metadata.
// Only includes immediate children of the directory, not nested descendants.
func (s *DownloadService) ListDirectory(ctx context.Context, c cid.Cid) ([]files.DirEntry, error) {
	immutablePath := path.FromCid(c)
	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return nil, err
	}
	defer node.Close()

	return directoryEntriesFromNode(node)
}

// ListDirectoryPath lists directory entries for a path within a directory CID.
// The dirPath is relative to the directory's root (e.g., "a/b/c").
// Returns a slice of directory entries for the resolved path.
// Only includes immediate children, not nested descendants.
func (s *DownloadService) ListDirectoryPath(ctx context.Context, dirCID cid.Cid, dirPath string) ([]files.DirEntry, error) {
	basePath := path.FromCid(dirCID)

	if strings.Trim(dirPath, "/") == "" {
		return s.ListDirectory(ctx, dirCID)
	}

	trimmedPath := strings.Trim(dirPath, "/")
	pathSegments := strings.Split(trimmedPath, "/")

	var segments []string
	for _, seg := range pathSegments {
		if seg != "" {
			if seg == ".." {
				return nil, fmt.Errorf("invalid path segment: %s", seg)
			}
			segments = append(segments, seg)
		}
	}

	fullPath, err := path.Join(basePath, segments...)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	immutablePath, err := path.NewImmutablePath(fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create immutable path: %w", err)
	}

	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return nil, err
	}
	defer node.Close()

	return directoryEntriesFromNode(node)
}

func directoryEntriesFromNode(node files.Node) ([]files.DirEntry, error) {
	dir, ok := node.(files.Directory)
	if !ok {
		return nil, fmt.Errorf("path is not a directory")
	}

	entries := make([]files.DirEntry, 0)
	it := dir.Entries()
	for it.Next() {
		name := it.Name()
		nd := it.Node()

		if name == car.CurrentDir || name == car.ParentDir {
			nd.Close()
			continue
		}

		entries = append(entries, files.FileEntry(name, nd))
	}

	if err := it.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// GetFile retrieves a specific file from a directory structure using a path.
// The path is relative to the directory's root (e.g., "a/b/c").
// Returns an io.ReadCloser that should be closed when done.
func (s *DownloadService) GetFile(ctx context.Context, dirCID cid.Cid, filePath string) (io.ReadCloser, error) {
	basePath := path.FromCid(dirCID)
	
	// Handle empty file path by using base path with trailing separator
	if strings.Trim(filePath, "/") == "" {
		// Empty path means we're accessing the root CID directly
		immutablePath, err := path.NewImmutablePath(basePath)
		if err != nil {
			return nil, fmt.Errorf("cannot create immutable path: %w", err)
		}
		_, node, err := s.backend.GetAll(ctx, immutablePath)
		if err != nil {
			return nil, err
		}

		return wrapFileNodeAsReadCloser(node, "CID is a directory, not a file")
	}
	
	// Trim and split the file path
	trimmedPath := strings.Trim(filePath, "/")
	pathSegments := strings.Split(trimmedPath, "/")
	
	// Skip empty segments and validate no path traversal
	var segments []string
	for _, seg := range pathSegments {
		if seg != "" {
			if seg == ".." {
				return nil, fmt.Errorf("invalid path segment: %s", seg)
			}
			segments = append(segments, seg)
		}
	}
	
	fullPath, err := path.Join(basePath, segments...)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	
	// Convert to immutable path
	immutablePath, err := path.NewImmutablePath(fullPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create immutable path: %w", err)
	}
	
	// Use GetAll to handle chunked files properly
	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return nil, err
	}

	return wrapFileNodeAsReadCloser(node, "path is a directory, not a file")
}


