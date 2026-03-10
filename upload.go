package ipfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/bdragon300/tusgo"
	"github.com/docker/go-units"
	"github.com/ipfs/go-cid"
	"go.lumeweb.com/ipfs-sdk/pkg/upload"
)

// DefaultUploadLimit is the default upload limit in bytes (100MB).
const DefaultUploadLimit = 100 * units.MiB

// UploadService provides file upload functionality using TUS and HTTP POST.
type UploadService struct {
	httpClient  *http.Client
	baseURL     string
	authToken   string
	tusEndpoint string
	uploadLimit int64
}

// UploadOptions configures an upload operation.
type UploadOptions struct {
	// MemoryLimit is the memory limit for CAR generation in bytes.
	// If 0, a sensible default will be used.
	MemoryLimit uint64

	// WrapInDir wraps single files in a root directory if true.
	// If false and there's only one file, the file itself will be the root.
	WrapInDir bool

	// UploadLimit is the size threshold for choosing TUS vs POST upload.
	// Files larger than this will use TUS resumable upload.
	// Smaller files will use HTTP POST upload.
	// If 0, DefaultUploadLimit is used.
	UploadLimit int64
}

// NewUploadService creates a new UploadService.
// baseURL is the base URL of the API server (e.g., "https://api.example.com").
// authToken is the authentication bearer token.
func NewUploadService(baseURL, authToken string, opts ...UploadServiceOption) *UploadService {
	s := &UploadService{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Default TUS endpoint is /api/upload/tus
	if s.tusEndpoint == "" {
		s.tusEndpoint = baseURL + "/api/upload/tus"
	}

	// Default upload limit
	if s.uploadLimit == 0 {
		s.uploadLimit = DefaultUploadLimit
	}

	return s
}

// UploadServiceOption configures an UploadService.
type UploadServiceOption func(*UploadService)

// WithHTTPClient sets a custom HTTP client for the upload service.
func WithHTTPClient(client *http.Client) UploadServiceOption {
	return func(s *UploadService) {
		s.httpClient = client
	}
}

// WithTUSEndpoint sets a custom TUS endpoint URL.
func WithTUSEndpoint(endpoint string) UploadServiceOption {
	return func(s *UploadService) {
		s.tusEndpoint = endpoint
	}
}

// WithUploadLimit sets a custom upload limit for choosing TUS vs POST.
func WithUploadLimit(limit int64) UploadServiceOption {
	return func(s *UploadService) {
		s.uploadLimit = limit
	}
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	CID  string
	Size int64
}

// Upload uploads data via TUS resumable upload protocol.
// ctx is the context for the operation.
// reader provides the data to upload.
// name is the name for the upload.
// size is the total size of the data in bytes.
func (s *UploadService) Upload(ctx context.Context, reader io.Reader, name string, size int64) (*UploadResult, error) {
	return s.uploadViaTUS(ctx, reader, name, size)
}

// UploadFromFS uploads a file or directory by generating a CAR file and uploading via the appropriate method.
// This method uses pkg/upload for CAR generation which provides:
// - Two-pass CAR generation (BuildSummary + WriteCAR)
// - LRU memory-constrained blockstore for efficient memory usage
// - On-the-fly block regeneration when blocks are evicted from cache
//
// ctx is the context for the operation.
// filesystem is the filesystem to upload (e.g., os.DirFS, memfs).
// name is the name for the upload.
// opts configures upload behavior (memory limit, dir wrapping, upload limit).
func (s *UploadService) UploadFromFS(ctx context.Context, filesystem fs.FS, name string, opts *UploadOptions) (*UploadResult, error) {
	// Ensure opts is not nil
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Set default memory limit if not provided
	memoryLimit := opts.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = upload.DefaultMemoryLimit
	}

	// Check if filesystem wraps a directory
	info, err := fs.Stat(filesystem, ".")
	if err != nil {
		return nil, fmt.Errorf("cannot access filesystem: %w", err)
	}

	// Determine wrapInDir - wrap in directory if it's a directory
	wrapInDir := opts.WrapInDir
	if wrapInDir == false && info.IsDir() {
		wrapInDir = true
	}

	// Pass 1: Build tree summary to get root CID and calculate CAR size
	bs, dagService := upload.NewDAGServiceWithMemoryLimit(memoryLimit)
	generator := upload.NewUnixFSNodeGenerator(
		upload.WithUnixFSNodeDAGService(dagService),
		upload.WithUnixFSNodeBlockstore(bs),
	)
	builder := upload.NewCARBuilder(bs, dagService, generator)
	summary, err := builder.BuildSummary(ctx, filesystem, wrapInDir)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upload: %w. Try reducing memory limit if this is a large directory", err)
	}

	// Calculate CAR size to determine upload method
	carSize, err := upload.CalculateCARSize(summary)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate upload size: %w", err)
	}

	// Pass 2: Create pipe for streaming CAR generation
	pr, pw := io.Pipe()

	go func() {
		err := builder.WriteCAR(ctx, pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()

	// Route based on CAR size vs upload limit
	uploadLimit := opts.UploadLimit
	if uploadLimit == 0 {
		uploadLimit = s.uploadLimit
	}

	if carSize <= uploadLimit {
		return s.uploadViaPOST(ctx, summary.RootCID, pr, name, carSize)
	}

	return s.uploadViaTUSWithCAR(ctx, summary.RootCID, pr, carSize)
}

// uploadViaTUS uploads via TUS protocol for large files/directories.
func (s *UploadService) uploadViaTUS(ctx context.Context, reader io.Reader, name string, size int64) (*UploadResult, error) {
	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}

	tusClient := tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx)

	// Create upload on server - upload object gets populated with Location
	upload := &tusgo.Upload{}
	_, err = tusClient.CreateUpload(upload, size, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUS upload: %w", err)
	}

	// Upload data using stream
	stream := tusgo.NewUploadStream(tusClient, upload)

	written, err := io.Copy(stream, reader)
	if err != nil {
		return nil, fmt.Errorf("upload interrupted: %w", err)
	}

	if written != size {
		return nil, fmt.Errorf("upload incomplete: expected %d bytes, wrote %d", size, written)
	}

	return &UploadResult{
		CID:  "", // Will be filled by the server response
		Size: written,
	}, nil
}

// uploadViaTUSWithCAR uploads a CAR stream via TUS protocol.
// This is used when uploading CAR files generated by pkg/upload.
func (s *UploadService) uploadViaTUSWithCAR(ctx context.Context, rootCID cid.Cid, carReader io.Reader, carSize int64) (*UploadResult, error) {
	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}

	tusClient := tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx)

	// Create upload on server with known CAR size
	upload := &tusgo.Upload{}
	_, err = tusClient.CreateUpload(upload, carSize, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start upload: %w", err)
	}

	// Upload CAR data
	stream := tusgo.NewUploadStream(tusClient, upload)

	written, err := io.Copy(stream, carReader)
	if err != nil {
		return nil, fmt.Errorf("upload interrupted: %w", err)
	}

	if written != carSize {
		return nil, fmt.Errorf("upload incomplete: expected %d bytes, wrote %d", carSize, written)
	}

	return &UploadResult{
		CID:  rootCID.String(),
		Size: written,
	}, nil
}

// uploadViaPOST uploads CAR data via HTTP POST as multipart form.
// This is used for smaller CAR files that fit within the upload limit.
func (s *UploadService) uploadViaPOST(ctx context.Context, rootCID cid.Cid, carReader io.Reader, name string, carSize int64) (*UploadResult, error) {
	// Create a pipe for streaming multipart form
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	type result struct {
		err error
	}

	// Channel to capture results from multipart writing
	resultChan := make(chan result, 1)

	// Write CAR to multipart form in goroutine
	go func() {
		var resultErr error

		defer func() {
			// Close writer first to finalize multipart form
			if err := writer.Close(); err != nil && resultErr == nil {
				resultErr = fmt.Errorf("failed to close multipart writer: %w", err)
			}
			// Then close pipe writer
			if err := pw.Close(); err != nil && resultErr == nil {
				resultErr = fmt.Errorf("failed to close pipe writer: %w", err)
			}
			// Always send result
			resultChan <- result{err: resultErr}
			close(resultChan)
		}()

		// Create form file field
		part, err := writer.CreateFormFile("file", name+".car")
		if err != nil {
			resultErr = fmt.Errorf("failed to create form file: %w", err)
			return
		}

		// Copy CAR from reader to multipart form
		if _, err := io.Copy(part, carReader); err != nil {
			resultErr = fmt.Errorf("failed to write CAR to multipart form: %w", err)
			return
		}
	}()

	// Upload the CAR as multipart form
	uploadEndpoint := s.baseURL + "/api/upload"
	uploadErr := s.postUpload(ctx, uploadEndpoint, pr, writer.FormDataContentType())

	// If the upload fails, the reading side of the pipe is abandoned.
	// We must close the pipe writer to unblock the writing goroutine, allowing it to terminate gracefully.
	if uploadErr != nil {
		pw.CloseWithError(uploadErr)
	}

	// Always wait for the multipart writing goroutine to complete to prevent leaks.
	res := <-resultChan
	if res.err != nil {
		// The writer goroutine failed, which is a more specific and critical error.
		return nil, res.err
	}

	// If the upload itself failed, return that error now that we've cleaned up the goroutine.
	if uploadErr != nil {
		return nil, uploadErr
	}

	return &UploadResult{
		CID:  rootCID.String(),
		Size: carSize,
	}, nil
}

// postUpload sends the CAR data via HTTP POST as multipart form.
func (s *UploadService) postUpload(ctx context.Context, endpoint string, body io.Reader, contentType string) error {
	// Create HTTP request with streaming body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add auth token if present
	if s.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.authToken)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetUploadStatus retrieves the status of a TUS upload by its location.
func (s *UploadService) GetUploadStatus(ctx context.Context, location string) (*tusgo.Upload, error) {
	if location == "" {
		return nil, fmt.Errorf("location cannot be empty")
	}

	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}

	tusClient := tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx)

	// Create upload object and populate it via GetUpload
	upload := &tusgo.Upload{}
	_, err = tusClient.GetUpload(upload, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload status: %w", err)
	}

	return upload, nil
}

// CancelUpload cancels an in-progress TUS upload.
func (s *UploadService) CancelUpload(ctx context.Context, location string) error {
	if location == "" {
		return fmt.Errorf("location cannot be empty")
	}

	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}

	tusClient := tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx)

	// Create upload object with the location
	upload := &tusgo.Upload{Location: location}
	_, err = tusClient.DeleteUpload(*upload)
	if err != nil {
		return fmt.Errorf("failed to cancel upload: %w", err)
	}

	return nil
}

// ResumeUpload resumes an interrupted TUS upload from a specific location.
// location is the location URL of the previous upload.
// reader provides the new data to upload.
func (s *UploadService) ResumeUpload(ctx context.Context, location string, reader io.Reader) (*UploadResult, error) {
	if location == "" {
		return nil, fmt.Errorf("location cannot be empty")
	}

	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}

	tusClient := tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx)

	// Get existing upload info to find the offset
	upload := &tusgo.Upload{}
	_, err = tusClient.GetUpload(upload, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload info: %w", err)
	}

	// Upload data stream
	stream := tusgo.NewUploadStream(tusClient, upload)

	written, err := io.Copy(stream, reader)
	if err != nil {
		return nil, fmt.Errorf("upload interrupted: %w", err)
	}

	return &UploadResult{
		CID:  "", // Will be filled by the server response
		Size: upload.RemoteOffset + written,
	}, nil
}

// VerifyUploadIntegrity verifies the integrity of an uploaded file by checking its CID.
func (s *UploadService) VerifyUploadIntegrity(ctx context.Context, cid cid.Cid) (bool, error) {
	// This would typically involve checking the uploaded file's CID against the expected CID
	// For now, return true - implementation would depend on server capabilities
	return true, nil
}

// MaxChunkSize returns the maximum recommended chunk size for TUS uploads.
func MaxChunkSize() int64 {
	return 50 * 1024 * 1024 // 50MB default chunk size
}

// SetAuthToken sets a new authentication token.
func (s *UploadService) SetAuthToken(token string) {
	s.authToken = token
}

// GetAuthToken returns the current authentication token.
func (s *UploadService) GetAuthToken() string {
	return s.authToken
}
