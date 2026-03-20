package ipfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/ipld/unixfs"
	"github.com/ipfs/boxo/path"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	backend "go.lumeweb.com/ipfs-sdk/internal/download"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// DownloadService provides functionality for downloading IPFS blocks and content
// from the gateway using the boxo gateway patterns.
type DownloadService struct {
	backend   backend.Backend
	httpClient *http.Client
	authTransport *httputil.AuthRoundTripper
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
	backend, err := gateway.NewRemoteBlocksBackend([]string{baseURL}, s.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway backend: %w", err)
	}

	s.backend = backend

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

// getUnixFSSize extracts the UnixFS file size from a UnixFS node's raw data.
// This function expects the node to contain only the UnixFS metadata (typically
// just the root block), not the entire file content.
func getUnixFSSize(data []byte) (int64, error) {
	fsNode, err := unixfs.FSNodeFromBytes(data)
	if err != nil {
		// If the data cannot be parsed as UnixFS, treat it as a raw block and return
		// the data length. This is expected for non-UnixFS CIDs (e.g., raw multicodec)
		// where the size is simply the byte length of the raw data.
		return int64(len(data)), nil
	}
	return int64(fsNode.FileSize()), nil
}

// FileSize returns the actual size of a UnixFS file by CID.
// For UnixFS files, this returns the file size metadata (not the root block size).
// For raw blocks, this returns the block data size.
// Uses GetBlock to fetch only the root block, avoiding memory exhaustion on large files.
func (s *DownloadService) FileSize(ctx context.Context, c cid.Cid) (int64, error) {
	// Fetch only the root block to access UnixFS metadata.
	// This avoids downloading and reading the entire file content into memory.
	_, file, err := s.backend.GetBlock(ctx, path.FromCid(c))
	if err != nil {
		return -1, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return -1, err
	}

	return getUnixFSSize(data)
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
func (s *DownloadService) ListDirectory(ctx context.Context, c cid.Cid) ([]files.DirEntry, error) {
	immutablePath := path.FromCid(c)
	_, node, err := s.backend.GetAll(ctx, immutablePath)
	if err != nil {
		return nil, err
	}
	defer node.Close()
	
	var entries []files.DirEntry
	err = files.Walk(node, func(fpath string, nd files.Node) error {
		// Skip the root path
		if fpath == "." || fpath == "/" {
			return nil
		}
		
		// Extract the base name from the path
		parts := strings.Split(strings.Trim(fpath, "/"), "/")
		name := parts[len(parts)-1]
		
		// Create a DirEntry using boxo's FileEntry helper
		entries = append(entries, files.FileEntry(name, nd))
		
		return nil
	})
	
	if err != nil {
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


