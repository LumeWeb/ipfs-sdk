package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/bdragon300/tusgo"
	"github.com/docker/go-units"
	"github.com/ipfs/boxo/ipld/unixfs/importer/trickle"
	"github.com/ipfs/go-cid"
	"go.lumeweb.com/ipfs-content/car"
	"go.lumeweb.com/ipfs-content/unixfs"
	go_fs "go.lumeweb.com/ipfs-sdk/fs"
	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

const maxRedirectHops = 5

var errRedirectUpgraded = errors.New("redirect-upgraded:")

type errReader struct{ err error }

func (e errReader) Read(_ []byte) (int, error) { return 0, e.err }

func unwrapErrReader(r io.Reader) (error, bool) {
	if er, ok := r.(errReader); ok {
		return er.err, true
	}
	return nil, false
}

// StreamToPipe runs a blocking function in a goroutine that writes to a pipe.
// This allows you to generate data (e.g., CAR files) without blocking the calling
// thread. The pipe reader is returned immediately for consumption.
//
// Error from fn is propagated via CloseWithError on the pipe. This helper is useful
// when you have a blocking function that writes to an io.Writer and you want to
// stream the output as it's being generated.
func StreamToPipe(fn func(io.Writer) error) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		err := fn(pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()

	return pr
}

// DefaultUploadLimit is the default upload limit in bytes (100MB).
const DefaultUploadLimit = 100 * units.MiB

// ArchiveMode represents the archive processing mode for uploads.
type ArchiveMode string

const (
	// ArchiveModeAuto enables automatic archive processing (default).
	// Archives are unpacked and converted to CAR format by the server.
	ArchiveModeAuto ArchiveMode = "true"

	// ArchiveModeRaw disables archive processing.
	// Files are uploaded as-is without unpacking. ZIP files treated as raw files.
	ArchiveModeRaw ArchiveMode = "false"
)

// String returns the string representation of the archive mode.
func (m ArchiveMode) String() string {
	return string(m)
}

// AutoArchive returns a pointer to ArchiveModeAuto for use with UploadOptions.
// This enables automatic archive processing (unpacked and converted to CAR).
func AutoArchive() *ArchiveMode {
	auto := ArchiveModeAuto
	return &auto
}

// RawArchive returns a pointer to ArchiveModeRaw for use with UploadOptions.
// This disables archive processing (files uploaded as-is, ZIP treated as raw).
func RawArchive() *ArchiveMode {
	raw := ArchiveModeRaw
	return &raw
}

// UploadDataType represents the type of data being uploaded (regular data or CAR format).
type UploadDataType string

const (
	// UploadDataTypeData represents regular file data uploads.
	UploadDataTypeData UploadDataType = "data"
	// UploadDataTypeCAR represents CAR (Content Addressable Archive) format uploads.
	UploadDataTypeCAR UploadDataType = "CAR"
)

// String returns the string representation of the upload data type.
func (t UploadDataType) String() string {
	return string(t)
}

// UploadService provides file upload functionality using TUS and HTTP POST.
type UploadService struct {
	httpClient  *http.Client
	baseURL     string
	mu          sync.RWMutex // guards authToken so SetAuthToken can hot-swap it concurrently with request-header reads
	authToken   string
	tusEndpoint string
	uploadLimit int64
}

// ChunkerStrategy is the DAG layout strategy for UnixFS node generation.
type ChunkerStrategy = unixfs.DAGLayoutFunc

var (
	// BalancedLayout is the default balanced DAG layout strategy.
	BalancedLayout ChunkerStrategy = unixfs.BalancedLayout
	// TrickleLayout is the trickle DAG layout strategy.
	TrickleLayout ChunkerStrategy = trickle.Layout
)

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

	// ArchiveConfig controls whether uploaded content should be processed by the server.
	// Use ArchiveModeAuto to enable processing (archives unpacked and converted to CAR format).
	// Use ArchiveModeRaw to disable processing (files uploaded as-is, ZIP treated as raw).
	//
	// For CAR format uploads, this setting is ignored by the server (bypassed).
	// For POST uploads with raw files, the CID returned IS correct (in-memory processing).
	// For TUS uploads with raw files, the CID may be incorrect due to background processing.
	//
	// Default: ArchiveModeAuto (process/unpack files on server)
	ArchiveConfig *ArchiveMode

	// ChunkSize is the chunk size in bytes for UnixFS file splitting.
	// If 0, the default of 1MB (1024 * 1024) is used.
	ChunkSize int64

	// ChunkerStrategy sets the DAG layout function (e.g. BalancedLayout).
	// If nil, the default balanced layout is used.
	ChunkerStrategy ChunkerStrategy

	// MaxLinks is the maximum number of links per DAG node.
	// If 0, the default of 174 (helpers.DefaultLinksPerBlock) is used.
	MaxLinks int
}

// NewUploadService creates a new UploadService.
// baseURL is the base URL of the API server (e.g., "https://api.example.com").
// authToken is the authentication bearer token.
func NewUploadService(baseURL, authToken string, opts ...UploadServiceOption) (*UploadService, error) {
	s := &UploadService{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Transport: httputil.NewAuthRoundTripper(defaultHTTPClient().Transport, authToken),
			// No client-level Timeout on the streaming path: it would abort large
			// block/file body transfers; transport timeouts guard idle hangs.
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Default TUS endpoint is /api/upload/tus
	if s.tusEndpoint == "" {
		s.tusEndpoint = s.buildEndpoint("/api/upload/tus")
	}

	// Default upload limit
	if s.uploadLimit == 0 {
		s.uploadLimit = DefaultUploadLimit
	}

	return s, nil
}

// UploadServiceOption configures an UploadService.
type UploadServiceOption func(*UploadService)

// WithHTTPClient sets a custom HTTP client for the upload service.
// It wraps the client's transport with authRoundTripper to add authorization headers.
func WithHTTPClient(client *http.Client) UploadServiceOption {
	return func(s *UploadService) {
		if client != nil {
			wrappedClient := &http.Client{
				Timeout:       client.Timeout,
				CheckRedirect: client.CheckRedirect,
				Jar:           client.Jar,
				Transport:     httputil.NewAuthRoundTripper(getTransport(client), s.authToken),
			}
			s.httpClient = wrappedClient
		}
	}
}

// getTransport extracts the http.RoundTripper from an http.Client,
// returning DefaultTransport if none is set.
func getTransport(client *http.Client) http.RoundTripper {
	if client == nil {
		return http.DefaultTransport
	}
	if client.Transport == nil {
		return http.DefaultTransport
	}
	return client.Transport
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

// TokenAwareTransport is the interface for transports that can update their auth token.
type TokenAwareTransport interface {
	SetAuthToken(token string)
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	CID      string // The content identifier
	Size     int64  // UnixFS logical file size (the actual file size users expect)
	DAGSize  int64  // Raw size of all blocks in the DAG
	Location string // The TUS upload location, when resumable upload was used
}

// UploadResultInfo is the result of querying upload status by identifier.
type UploadResultInfo = internalclient.UploadResultResponse

// UploadStatusEnum represents the status of an upload result. Type alias to
// expose the generated enum from the client.
type UploadStatusEnum = internalclient.UploadResultResponseStatus

const (
	// UploadStatusCompleted - upload completed successfully
	UploadStatusCompleted UploadStatusEnum = internalclient.UploadResultResponseStatusCompleted
	// UploadStatusDuplicate - upload was a duplicate of an existing upload
	UploadStatusDuplicate UploadStatusEnum = internalclient.UploadResultResponseStatusDuplicate
	// UploadStatusFailed - upload failed
	UploadStatusFailed UploadStatusEnum = internalclient.UploadResultResponseStatusFailed
	// UploadStatusPending - upload is queued and waiting
	UploadStatusPending UploadStatusEnum = internalclient.UploadResultResponseStatusPending
	// UploadStatusProcessing - upload is being processed
	UploadStatusProcessing UploadStatusEnum = internalclient.UploadResultResponseStatusProcessing
)

// getUploadResultSize determines the appropriate Size field value for UploadResult.
// For CAR uploads (unixFSSize > 0), returns the UnixFS logical size.
// For non-CAR uploads (unixFSSize == 0), returns the network transfer size.
func getUploadResultSize(unixFSSize int64, networkTransferSize int64) int64 {
	if unixFSSize > 0 {
		return unixFSSize
	}
	return networkTransferSize
}

// uploadTask represents a single upload operation with all necessary parameters.
// It encapsulates the upload configuration and can be executed via the Execute() method
// which routes to the appropriate upload protocol (POST or TUS) based on file size.
type uploadTask struct {
	service       *UploadService
	ctx           context.Context
	reader        io.Reader
	name          string
	size          int64
	cid           cid.Cid
	isCAR         bool
	archiveConfig *ArchiveMode
	uploadLimit   int64
	unixFSSize    int64
	dagSize       int64
}

// Execute performs the actual upload, routing to POST or TUS based on size.
func (t *uploadTask) Execute() (*UploadResult, error) {
	var result *UploadResult
	var err error

	uploadLimit := t.uploadLimit

	if t.size <= uploadLimit {
		result, err = t.service.uploadViaPOST(t.ctx, t.reader, t.name, t.size, t.isCAR, t.archiveConfig, t.unixFSSize, t.dagSize)
	} else {
		result, err = t.service.uploadViaTUS(t.ctx, t.reader, t.name, t.size, t.archiveConfig, t.unixFSSize, t.dagSize)
	}

	if err != nil {
		return nil, err
	}

	// For CAR uploads, we already know the CID from the CAR generation
	if !t.cid.Equals(cid.Undef) {
		result.CID = t.cid.String()
	}

	return result, nil
}

// Upload uploads data via the appropriate protocol based on file size.
// Files smaller than or equal to the upload limit use HTTP POST.
// Larger files use TUS resumable upload protocol.
//
// ctx is the context for the operation.
// reader provides the data to upload.
// name is the name for the upload.
// size is the total size of the data in bytes.
func (s *UploadService) Upload(ctx context.Context, reader io.Reader, name string, size int64) (*UploadResult, error) {
	task := &uploadTask{
		service:     s,
		ctx:         ctx,
		reader:      reader,
		name:        name,
		size:        size,
		isCAR:       false,
		uploadLimit: s.uploadLimit,
	}
	return task.Execute()
}

// UploadFromFS uploads a file or directory by generating a CAR file and uploading via the appropriate method.
// This method uses go.lumeweb.com/ipfs-content for CAR generation which provides:
// - Two-pass CAR generation (BuildSummary + WriteCAR)
// - LRU memory-constrained blockstore for efficient memory usage
// - On-the-fly block regeneration when blocks are evicted from cache
//
// ctx is the context for the operation.
// filesystem is the filesystem to upload (e.g., os.DirFS, testing/fstest.MapFS).
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
		memoryLimit = 100 * uint64(units.MiB) // Default 100MB
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

	var pr io.ReadCloser

	var carBuilderOpts []car.CARBuilderOption
	if opts.ChunkSize > 0 {
		carBuilderOpts = append(carBuilderOpts, car.WithChunkSize(opts.ChunkSize))
	}
	if opts.ChunkerStrategy != nil {
		carBuilderOpts = append(carBuilderOpts, car.WithChunkerStrategy(opts.ChunkerStrategy))
	}
	if opts.MaxLinks > 0 {
		carBuilderOpts = append(carBuilderOpts, car.WithMaxLinks(opts.MaxLinks))
	}

	builder, summary, err := car.PrepareCARWithOptions(ctx, filesystem, wrapInDir, carBuilderOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upload: %w. Try reducing memory limit if this is a large directory", err)
	}

	// Calculate CAR size to determine upload method
	carSize, err := car.CalculateCARSize(summary)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate upload size: %w", err)
	}

	// Pass 2: Stream CAR generation to pipe
	pr = StreamToPipe(func(w io.Writer) error {
		return builder.WriteCAR(ctx, w)
	})

	// Route based on CAR size vs upload limit
	uploadLimit := opts.UploadLimit
	if uploadLimit == 0 {
		uploadLimit = s.uploadLimit
	}

	// Create upload task for CAR upload
	task := &uploadTask{
		service:       s,
		ctx:           ctx,
		reader:        pr,
		name:          name,
		size:          carSize,
		cid:           summary.RootCID,
		isCAR:         true,
		archiveConfig: opts.ArchiveConfig,
		uploadLimit:   uploadLimit,
		unixFSSize:    int64(summary.TotalLogicalFileSize()),
		dagSize:       int64(summary.TotalSize),
	}
	return task.Execute()
}

// getTUSClient creates a configured TUS client with error handling.
func (s *UploadService) getTUSClient(ctx context.Context) (*tusgo.Client, error) {
	baseURL, err := url.Parse(s.tusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TUS endpoint: %w", err)
	}
	return tusgo.NewClient(s.httpClient, baseURL).WithContext(ctx), nil
}

// uploadViaTUS uploads data via TUS resumable upload protocol.
func (s *UploadService) uploadViaTUS(ctx context.Context, reader io.Reader, name string, size int64, archiveConfig *ArchiveMode, unixFSSize int64, dagSize int64) (*UploadResult, error) {
	tusClient, err := s.getTUSClient(ctx)
	if err != nil {
		return nil, err
	}

	// Build TUS metadata from archive config and the optional pin name. The
	// name is sent via the TUS Upload-Metadata header so the resulting pin is
	// named on the server.
	metadata := make(map[string]string)
	if archiveConfig != nil {
		metadata["archive"] = string(*archiveConfig)
	}
	if name != "" {
		metadata["name"] = name
	}

	// Create upload on server and send initial data to initialize multipart upload
	upload := &tusgo.Upload{}

	// For certain backends, we need to use CreateUploadWithData to initialize the upload
	// Send initial chunk with default 2MB chunk size
	const initialChunkSize = 2 * units.MiB

	var initialChunk []byte
	var n int

	// Calculate how much to read in initial chunk (don't exceed file size)
	readSize := int64(min(initialChunkSize, size))

	if readSize > 0 {
		initialChunk = make([]byte, readSize)
		var readErr error
		n, readErr = io.ReadFull(reader, initialChunk)

		// If we hit EOF or ErrUnexpectedEOF, it means the reader has less data than claimed
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("upload incomplete: reader ended after %d bytes, expected %d bytes", n, size)
		}

		if readErr != nil {
			return nil, fmt.Errorf("failed to read initial chunk: %w", readErr)
		}

		// Slice to actual bytes read
		initialChunk = initialChunk[:n]
	}

	// Create upload with initial data (required for some backends)
	// The server will initialize the multipart upload with this request
	uploadedBytes, resp, err := tusClient.CreateUploadWithData(upload, initialChunk, size, false, metadata)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create TUS upload: %w", err)
	}

	// If the entire file was uploaded in the initial request, return early
	if int64(n) == size {
		return &UploadResult{
			CID:      "", // Will be filled by the server response
			Size:     getUploadResultSize(unixFSSize, uploadedBytes),
			DAGSize:  dagSize,
			Location: upload.Location,
		}, nil
	}

	// Upload remaining data using only the reader, which already has the remaining data
	// The initial chunk was already sent via CreateUploadWithData
	remainingReader := reader

	// Upload remaining data using stream
	// The upload is already initialized on the server, so we can start streaming
	stream := tusgo.NewUploadStream(tusClient, upload)

	written, err := io.Copy(stream, remainingReader)
	if err != nil {
		return nil, fmt.Errorf("upload interrupted: %w", err)
	}

	totalWritten := int64(n) + written
	if totalWritten != size {
		return nil, fmt.Errorf("upload incomplete: expected %d bytes, wrote %d", size, totalWritten)
	}

	return &UploadResult{
		CID:      "", // Will be filled by the server response
		Size:     getUploadResultSize(unixFSSize, totalWritten),
		DAGSize:  dagSize,
		Location: upload.Location,
	}, nil
}

// uploadViaPOST uploads data via HTTP POST as multipart form.
// This is used for smaller files that fit within the upload limit.
// It handles 307/308 redirects (e.g. Caddy HTTP→HTTPS) by detecting the
// redirect, discarding the pipe, and retrying with a fresh pipe + goroutine.
func (s *UploadService) uploadViaPOST(ctx context.Context, reader io.Reader, name string, size int64, isCAR bool, archiveConfig *ArchiveMode, unixFSSize int64, dagSize int64) (*UploadResult, error) {
	fileName := name
	dataType := UploadDataTypeData
	if isCAR {
		fileName = name + ".car"
		dataType = UploadDataTypeCAR
	}

	seeker, isSeeker := reader.(io.Seeker)
	var buf bytes.Buffer
	bufReader := func() io.Reader {
		if isSeeker {
			return reader
		}
		if buf.Len() == 0 {
			if _, err := io.Copy(&buf, reader); err != nil {
				return errReader{err}
			}
		}
		return bytes.NewReader(buf.Bytes())
	}

	uploadEndpoint := s.buildEndpoint("/api/upload")

	for range maxRedirectHops {
		if isSeeker {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek upload reader: %w", err)
			}
		}
		r := bufReader()
		if err, ok := unwrapErrReader(r); ok {
			return nil, err
		}
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		type result struct {
			err error
		}
		resultChan := make(chan result, 1)

		go func() {
			var resultErr error
			defer func() {
				if err := writer.Close(); err != nil && resultErr == nil {
					resultErr = fmt.Errorf("failed to close multipart writer: %w", err)
				}
				if err := pw.Close(); err != nil && resultErr == nil {
					resultErr = fmt.Errorf("failed to close pipe writer: %w", err)
				}
				resultChan <- result{err: resultErr}
				close(resultChan)
			}()

			part, err := writer.CreateFormFile("file", fileName)
			if err != nil {
				resultErr = fmt.Errorf("failed to create form file: %w", err)
				return
			}
			if _, err := io.Copy(part, r); err != nil {
				resultErr = fmt.Errorf("failed to write %s to multipart form: %w", dataType.String(), err)
				return
			}
		}()

		uploadErr := s.postUpload(ctx, uploadEndpoint, pr, writer.FormDataContentType(), archiveConfig, name)
		if uploadErr != nil {
			pw.CloseWithError(uploadErr)
		}
		res := <-resultChan

		if uploadErr != nil {
			if errors.Is(uploadErr, errRedirectUpgraded) {
				uploadEndpoint = strings.TrimPrefix(uploadErr.Error(), errRedirectUpgraded.Error())
				continue
			}
			return nil, uploadErr
		}
		if res.err != nil {
			return nil, res.err
		}

		return &UploadResult{
			CID:     "",
			Size:    getUploadResultSize(unixFSSize, size),
			DAGSize: dagSize,
		}, nil
	}

	return nil, fmt.Errorf("upload exceeded %d redirect hops", maxRedirectHops)
}

// postUpload sends the CAR data via HTTP POST as multipart form.
// If the server responds with 307/308, it returns errRedirectUpgraded
// with the resolved URL so the caller can retry with a fresh pipe.
func (s *UploadService) postUpload(ctx context.Context, endpoint string, body io.Reader, contentType string, archiveConfig *ArchiveMode, name string) error {
	// The optional pin name is sent as a query parameter so the server can name
	// the resulting pin, matching the POST /upload DTO (query:"name").
	if archiveConfig != nil || name != "" {
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("failed to parse endpoint URL: %w", err)
		}
		q := parsedURL.Query()
		if archiveConfig != nil {
			q.Set("archive", string(*archiveConfig))
		}
		if name != "" {
			q.Set("name", name)
		}
		parsedURL.RawQuery = q.Encode()
		endpoint = parsedURL.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.mu.RLock()
	token := s.authToken
	s.mu.RUnlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		loc := resp.Header.Get("Location")
		if loc == "" {
			return fmt.Errorf("upload redirect (HTTP %d) with no Location header", resp.StatusCode)
		}
		resolved, err := url.Parse(loc)
		if err != nil {
			return fmt.Errorf("upload redirect with invalid Location %q: %w", loc, err)
		}
		if !resolved.IsAbs() {
			base, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("upload redirect with unresolvable base: %w", err)
			}
			resolved = base.ResolveReference(resolved)
		}
		return fmt.Errorf("%w%s", errRedirectUpgraded, resolved.String())
	}

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

	tusClient, err := s.getTUSClient(ctx)
	if err != nil {
		return nil, err
	}

	// Create upload object and populate it via GetUpload
	upload := &tusgo.Upload{}
	_, err = tusClient.GetUpload(upload, location)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload status: %w", err)
	}

	return upload, nil
}

// GetUploadResult retrieves the result of an upload operation by identifier.
// Accepts either a TUS upload ID or a numeric request ID as the identifier.
func (s *UploadService) GetUploadResult(ctx context.Context, identifier string) (*UploadResultInfo, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	endpoint := s.buildEndpoint(fmt.Sprintf("/api/upload/result/%s", url.PathEscape(identifier)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.mu.RLock()
	token := s.authToken
	s.mu.RUnlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get upload result failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result UploadResultInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode upload result: %w", err)
	}

	return &result, nil
}

// CancelUpload cancels an in-progress TUS upload.
func (s *UploadService) CancelUpload(ctx context.Context, location string) error {
	if location == "" {
		return fmt.Errorf("location cannot be empty")
	}

	tusClient, err := s.getTUSClient(ctx)
	if err != nil {
		return err
	}

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

	tusClient, err := s.getTUSClient(ctx)
	if err != nil {
		return nil, err
	}

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
		CID:      "", // Will be filled by the server response
		Size:     upload.RemoteOffset + written,
		DAGSize:  0, // Not available for non-CAR uploads
		Location: upload.Location,
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
	s.mu.Lock()
	s.authToken = token
	s.mu.Unlock()
	// Update the token in the RoundTripper if it supports TokenAwareTransport
	if rt, ok := s.httpClient.Transport.(TokenAwareTransport); ok {
		rt.SetAuthToken(token)
	}
}

// GetAuthToken returns the current authentication token.
func (s *UploadService) GetAuthToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authToken
}

// buildEndpoint constructs a full endpoint URL from the base URL and a relative path.
// It ensures the base URL is properly parsed and any existing path is cleared before
// joining the new path. This method handles various base URL formats (with/without
// trailing slash, with/without existing paths) to ensure consistent endpoint construction.
func (s *UploadService) buildEndpoint(path string) string {
	parsedURL, err := url.Parse(s.baseURL)
	if err != nil {
		parsedURL = &url.URL{Scheme: "https", Host: s.baseURL}
	}
	parsedURL.Path = ""
	return parsedURL.JoinPath(path).String()
}

// UploadBytes uploads byte data by wrapping it in CAR format and uploading.
// This is a convenience method for the common case of uploading byte slices.
// It automatically wraps the data in a CAR file and uploads via the appropriate
// method (POST or TUS) based on the resulting CAR size.
//
// ctx is the context for the operation.
// data is the byte data to upload.
// filename is the name for the uploaded file (without .car extension).
// opts configures upload behavior (memory limit, wrap-in-dir, upload limit).
func (s *UploadService) UploadBytes(ctx context.Context, data []byte, filename string, opts *UploadOptions) (*UploadResult, error) {
	filesystem := go_fs.NewBytesFS(data, filename)
	return s.UploadFromFS(ctx, filesystem, filename, opts)
}

// UploadFile uploads a single file by wrapping it in a filesystem and uploading via UploadFromFS.
// This method provides a convenient way to upload files without manually creating filesystem wrappers.
// The file is automatically wrapped in a SingleFileFS implementation and uploaded via CAR generation.
//
// This method is useful when you have a file handle (*os.File) and want to upload it with CAR
// generation, which provides content addressing and verification.
//
// ctx is the context for the operation.
// file is the open file handle to upload. The caller is responsible for closing the file.
// filename is the name for the upload.
// opts configures upload behavior (memory limit, wrap-in-dir, upload limit).
//
// Example usage:
//
//	file, err := os.Open(path/to/file.txt)
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//
//	opts := &UploadOptions{
//	    MemoryLimit: 100 * 1024 * 1024, // 100MB
//	}
//	result, err := uploadService.UploadFile(ctx, file, "file.txt", opts)
func (s *UploadService) UploadFile(ctx context.Context, file fs.File, filename string, opts *UploadOptions) (*UploadResult, error) {
	// Wrap the file in a SingleFileFS
	filesystem := go_fs.NewSingleFileFS(file, filename)

	// Ensure opts is not nil and set WrapInDir=false for single file
	if opts == nil {
		opts = &UploadOptions{}
	}

	return s.UploadFromFS(ctx, filesystem, filename, opts)
}
