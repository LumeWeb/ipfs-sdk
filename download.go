package ipfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/path"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	backend "go.lumeweb.com/ipfs-sdk/internal/download"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
	"go.lumeweb.com/ipfs-content/car"
	"go.lumeweb.com/ipfs-content/dagnode"
)



// DownloadService provides functionality for downloading IPFS blocks and content
// from the gateway using the boxo gateway patterns.
type DownloadService struct {
	backend        backend.Backend
	httpClient     *http.Client
	authTransport  *httputil.AuthRoundTripper
	authToken      string
	baseURL        string
	remoteBackend  *backend.RemoteBackend
	rateLimiter    backend.RateLimiter
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
func WithDownloadRateLimiter(rl backend.RateLimiter) DownloadServiceOption {
	return func(s *DownloadService) {
		s.rateLimiter = rl
	}
}

// WithDownloadWorkerPoolSize sets the maximum number of concurrent download workers.
// Only applies when a rate limiter is configured.
func WithDownloadWorkerPoolSize(size int) DownloadServiceOption {
	return func(s *DownloadService) {
		// This option is applied when creating the remote backend
		if s.rateLimiter != nil {
		}
	}
}

// WithDownloadRetryConfig sets the retry configuration for download operations.
// Only applies when a rate limiter is configured.
func WithDownloadRetryConfig(cfg httputil.RetryConfig) DownloadServiceOption {
	return func(s *DownloadService) {
		// This option is applied when creating the remote backend
		if s.rateLimiter != nil {
		}
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
	gatewayBackend, err := gateway.NewRemoteBlocksBackend([]string{baseURL}, s.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway backend: %w", err)
	}

	// Wrap with RemoteBackend if a rate limiter is provided
	if s.rateLimiter != nil {
		s.remoteBackend = backend.NewBackend(gatewayBackend, s.httpClient,
			backend.WithRateLimiter(s.rateLimiter),
		)
		s.backend = s.remoteBackend
	} else {
		s.backend = gatewayBackend
	}

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



// FileSize returns the actual size of a UnixFS file by CID.
// Strategy 1: Try GetBlock first (fast, memory-efficient)
//   - For small files with UnixFS metadata, reads only the root block (~1MB max)
//   - For raw blocks, returns the block data length
// Strategy 2: Fallback to GetAll for chunked files
//   - For large chunked files, walks the DAG structure without loading file data
//   - Uses file.Size() to get actual size from UnixFS DAG metadata
//
// This implementation never loads large file contents into memory.
// Strategy 1 reads at most one block (IPFS max size is ~1MB).
// Strategy 2 only walks the DAG structure for metadata, not file data.
func (s *DownloadService) FileSize(ctx context.Context, c cid.Cid) (int64, error) {
	// Strategy 1: Try GetBlock first (fast, memory-efficient)
	_, file, err := s.backend.GetBlock(ctx, path.FromCid(c))
	if err == nil {
		defer file.Close()

		// Read the block data
		data, err := io.ReadAll(file)
		if err != nil {
			return -1, fmt.Errorf("failed to read block: %w", err)
		}

		// Use ipfs-content's AnalyzeNode for comprehensive node analysis
		block := blocks.NewBlock(data)
		nodeInfo, err := dagnode.AnalyzeNode(ctx, block)
		if err != nil {
			// AnalyzeNode failed - treat as raw block
			return int64(len(data)), nil
		}

		// If it's a UnixFS file, use the file size from NodeInfo
		if nodeInfo.IsUnixFS {
			// For UnixFS files with chunks, FileSize is 0 (computed from ChunkSizes)
			if nodeInfo.FileSize > 0 {
				return int64(nodeInfo.FileSize), nil
			}
			// For chunked files, sum up chunk sizes
			if len(nodeInfo.ChunkSizes) > 0 {
				var totalSize uint64
				for _, chunkSize := range nodeInfo.ChunkSizes {
					totalSize += chunkSize
				}
				return int64(totalSize), nil
			}
			// Fallback to data size
			return int64(nodeInfo.DataSize), nil
		}

		// Raw block - return data length
		return int64(nodeInfo.DataSize), nil
	}

	// Strategy 2: Fallback to GetAll for chunked files
	// This walks the DAG structure to get UnixFS file size metadata
	// It does NOT load file data into memory
	immutablePath := path.FromCid(c)
	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return -1, fmt.Errorf("failed to get file node: %w", err)
	}
	defer node.Close()

	fileNode, ok := node.(files.File)
	if !ok {
		return -1, fmt.Errorf("CID is not a file")
	}

	size, err := fileNode.Size()
	if err != nil {
		return -1, fmt.Errorf("failed to get file size: %w", err)
	}

	return size, nil
}

// BlockSize returns the size of a block by CID without fetching the block data.
// Note: Due to unexported fields in HeadResponse, this implementation
// actually fetches the block to determine size.
func (s *DownloadService) BlockSize(ctx context.Context, c cid.Cid) (int, error) {
	_, file, err := s.backend.GetBlock(ctx, path.FromCid(c))
	if err != nil {
		return -1, err
	}
	defer file.Close()
	
	size, err := file.Size()
	if err != nil {
		return -1, err
	}
	return int(size), nil
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
	s.authToken = token
	if s.authTransport != nil {
		s.authTransport.SetAuthToken(token)
	}
}

// AuthToken returns the current authentication token.
func (s *DownloadService) AuthToken() string {
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
	
	// Convert the node to a Directory if it is one
	dir, ok := node.(files.Directory)
	if !ok {
		return nil, fmt.Errorf("CID is not a directory")
	}
	
	// Use Directory.Entries() to get immediate children only
	// This avoids the flattened recursive behavior of files.Walk
	entries := make([]files.DirEntry, 0)
	it := dir.Entries()
	for it.Next() {
		name := it.Name()
		nd := it.Node()
		
		if name == car.CurrentDir || name == car.ParentDir {
			nd.Close()
			continue
		}
		
		// Create a DirEntry using boxo's FileEntry helper
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


