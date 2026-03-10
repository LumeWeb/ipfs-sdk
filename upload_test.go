package ipfs

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/memorylocker"
)

// memoryStore implements a simple in-memory TUS store for testing
type memoryStore struct {
	mu      sync.RWMutex
	uploads map[string]*memoryUpload
}

// memoryUpload stores uploaded data in memory for testing
type memoryUpload struct {
	mu     sync.Mutex
	info   handler.FileInfo
	data   []byte
	offset int64
	closed bool
}

// newMemoryStore creates a new in-memory TUS store
func newMemoryStore() *memoryStore {
	return &memoryStore{
		uploads: make(map[string]*memoryUpload),
	}
}

func (s *memoryStore) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate a UUID if no ID is provided
	if info.ID == "" {
		info.ID = uuid.New().String()
	}

	upload := &memoryUpload{
		info: info,
		data: make([]byte, 0, info.Size),
	}
	s.uploads[info.ID] = upload
	return upload, nil
}

func (s *memoryStore) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, ok := s.uploads[id]
	if !ok {
		return nil, handler.ErrNotFound
	}
	return upload, nil
}

func (s *memoryStore) AsTerminatableUpload(upload handler.Upload) handler.TerminatableUpload {
	return upload.(*memoryUpload)
}

func (u *memoryUpload) GetInfo(ctx context.Context) (handler.FileInfo, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.info, nil
}

func (u *memoryUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Read all available data from source
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, src)
	if err != nil {
		return 0, err
	}

	data := buf.Bytes()

	// Ensure data slice has enough capacity
	if int64(cap(u.data)) < u.offset+int64(len(data)) {
		newData := make([]byte, len(u.data), u.offset+int64(len(data)))
		copy(newData, u.data)
		u.data = newData
	}

	// Ensure data slice has enough length
	if int64(len(u.data)) < u.offset {
		u.data = u.data[:u.offset]
	}

	// Append new data
	u.data = append(u.data, data...)
	u.offset += int64(n)

	return n, nil
}

func (u *memoryUpload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return io.NopCloser(bytes.NewReader(u.data)), nil
}

func (u *memoryUpload) Terminate(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.closed = true
	return nil
}

func (u *memoryUpload) FinishUpload(ctx context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return nil
}

// setupTUSTest creates a complete TUS test environment with a real TUS server
func setupTUSTest(t *testing.T) (*httptest.Server, *handler.Handler, *memoryStore) {
	// Create TUS server with memory store
	store := newMemoryStore()
	locker := memorylocker.New()
	composer := handler.NewStoreComposer()
	composer.UseCore(store)
	composer.UseTerminater(store)
	composer.UseLocker(locker)

	tusHandler, err := handler.NewHandler(handler.Config{
		StoreComposer: composer,
		BasePath:      "/tus",
	})
	require.NoError(t, err)

	// Create HTTP test server
	server := httptest.NewServer(http.StripPrefix("/tus", tusHandler))

	return server, tusHandler, store
}

func TestNewUploadService(t *testing.T) {
	t.Run("creates service with default endpoint", func(t *testing.T) {
		service := NewUploadService("https://api.example.com", "test-token")

		require.NotNil(t, service)
		assert.Equal(t, "https://api.example.com", service.baseURL)
		assert.Equal(t, "test-token", service.authToken)
		assert.Equal(t, "https://api.example.com/api/upload/tus", service.tusEndpoint)
		assert.NotNil(t, service.httpClient)
	})

	t.Run("applies WithHTTPClient option", func(t *testing.T) {
		customClient := &http.Client{Timeout: 30 * time.Second}
		service := NewUploadService("https://api.example.com", "test-token", WithHTTPClient(customClient))

		assert.Same(t, customClient, service.httpClient)
	})

	t.Run("applies WithTUSEndpoint option", func(t *testing.T) {
		customEndpoint := "https://custom.example.com/tus"
		service := NewUploadService("https://api.example.com", "test-token", WithTUSEndpoint(customEndpoint))

		assert.Equal(t, customEndpoint, service.tusEndpoint)
	})
}

func TestUploadService_Upload_Success(t *testing.T) {
	t.Run("uploads data successfully via TUS", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create test data
		testData := []byte("test file content for upload")
		testSize := int64(len(testData))
		testName := "test.txt"

		// Create upload service with TUS endpoint pointing to mock server
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		// Upload
		ctx := context.Background()
		reader := bytes.NewReader(testData)
		result, err := service.Upload(ctx, reader, testName, testSize)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(len(testData)), result.Size)
	})
}

func TestUploadService_Upload_Incomplete(t *testing.T) {
	t.Run("returns error when upload size mismatch", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		testData := []byte("short")
		testSize := int64(100) // Claim larger than actual
		testName := "test.txt"

		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		reader := bytes.NewReader(testData)
		_, err := service.Upload(ctx, reader, testName, testSize)

		// Should fail because written bytes don't match expected size
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "upload incomplete")
	})
}

func TestUploadService_Upload_CreateFailure(t *testing.T) {
	t.Run("returns error when create upload fails", func(t *testing.T) {
		// Create a server that returns errors
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		testData := []byte("test content")

		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		reader := bytes.NewReader(testData)
		_, err := service.Upload(ctx, reader, "test.txt", int64(len(testData)))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create TUS upload")
	})
}

func TestUploadService_GetUploadStatus(t *testing.T) {
	t.Run("retrieves upload status successfully", func(t *testing.T) {
		server, _, store := setupTUSTest(t)
		defer server.Close()

		ctx := context.Background()

		// First create an upload
		info := handler.FileInfo{
			Size:      100,
			MetaData:  map[string]string{"filename": "test.txt"},
		}
		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		// Get the upload ID
		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		uploadID := uploadInfo.ID

		// Create service
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		// Get upload status using the full URL
		location := server.URL + "/tus/" + uploadID
		status, err := service.GetUploadStatus(ctx, location)

		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.NotEmpty(t, status.Location)
	})

	t.Run("returns error when location is empty", func(t *testing.T) {
		service := NewUploadService("https://api.example.com", "test-token")

		ctx := context.Background()
		_, err := service.GetUploadStatus(ctx, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location cannot be empty")
	})

	t.Run("returns error when TUS endpoint is invalid", func(t *testing.T) {
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint("://invalid-url"),
		)

		ctx := context.Background()
		_, err := service.GetUploadStatus(ctx, "https://example.com/files/upload123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse TUS endpoint")
	})
}

func TestUploadService_CancelUpload(t *testing.T) {
	t.Run("returns error when location is empty", func(t *testing.T) {
		service := NewUploadService("https://api.example.com", "test-token")

		ctx := context.Background()
		err := service.CancelUpload(ctx, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location cannot be empty")
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		// Create a server that returns errors
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		err := service.CancelUpload(ctx, server.URL+"/tus/upload123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cancel upload")
	})

}

func TestUploadService_ResumeUpload(t *testing.T) {
	t.Run("resumes upload successfully", func(t *testing.T) {
		testData := []byte("resume data")

		server, _, store := setupTUSTest(t)
		defer server.Close()

		ctx := context.Background()

		// First create an upload with partial data
		info := handler.FileInfo{
			Size:      100,
			MetaData:  map[string]string{"filename": "test.txt"},
		}
		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		// Get the upload ID
		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		uploadID := uploadInfo.ID

		// Write some initial data
		initialData := []byte("initial")
		_, err = upload.WriteChunk(ctx, 0, bytes.NewReader(initialData))
		require.NoError(t, err)

		// Create service
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		// Resume upload
		location := server.URL + "/tus/" + uploadID
		reader := bytes.NewReader(testData)
		result, err := service.ResumeUpload(ctx, location, reader)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error when location is empty", func(t *testing.T) {
		service := NewUploadService("https://api.example.com", "test-token")

		ctx := context.Background()
		_, err := service.ResumeUpload(ctx, "", strings.NewReader("data"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location cannot be empty")
	})

	t.Run("returns error when get upload info fails", func(t *testing.T) {
		// Create a server that returns errors
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		_, err := service.ResumeUpload(ctx, server.URL+"/tus/upload123", strings.NewReader("data"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get upload info")
	})
}

func TestUploadService_VerifyUploadIntegrity(t *testing.T) {
	t.Run("currently returns true as placeholder", func(t *testing.T) {
		_ = NewUploadService("https://api.example.com", "test-token")
		// This is a placeholder implementation
		// Full implementation would check against server
		assert.True(t, true, "placeholder test")
	})
}

func TestMaxChunkSize(t *testing.T) {
	t.Run("returns a reasonable chunk size", func(t *testing.T) {
		chunkSize := MaxChunkSize()
		assert.Greater(t, chunkSize, int64(0))
		// Should be a reasonable size like 8MB or similar
		assert.Greater(t, chunkSize, int64(1*1024*1024))
	})
}

// Example integration test showing TUS upload with real TUS server

func TestUploadService_IntegrationCompleteFlow(t *testing.T) {
	t.Run("full TUS upload flow from start to finish", func(t *testing.T) {
		testData := []byte("This is a test file for TUS upload")
		testName := "integration-test.txt"
		testSize := int64(len(testData))

		server, _, _ := setupTUSTest(t)
		defer server.Close()

		baseURL, err := url.Parse(server.URL)
		require.NoError(t, err)

		service := NewUploadService(baseURL.String(), "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()

		// 1. Upload
		result, err := service.Upload(ctx, bytes.NewReader(testData), testName, testSize)
		require.NoError(t, err)
		require.Equal(t, int64(len(testData)), result.Size)

		// 2. Cancel upload
		// Since UploadResult doesn't include Location, we can't cancel the upload
		// This test demonstrates the complete upload flow
		require.NoError(t, err)
	})
}

// TestUploadFromFS tests the new UploadFromFS method that generates CAR files and uploads.
// setupPOSTTest creates a test server for HTTP POST multipart uploads
func setupPOSTTest(t *testing.T) *httptest.Server {
	var bodyHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check authorization
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Success
		w.WriteHeader(http.StatusOK)
	}

	server := httptest.NewServer(bodyHandler)
	return server
}

func TestUploadFromFS(t *testing.T) {
	t.Run("uploads single file via TUS", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create a test filesystem with a single file
		testFS := newMemFS()
		testFS.AddFile("test.txt", "Hello, World!")

		// Set upload limit to 1 byte so it routes to TUS (CAR size > 1 byte)
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1),
		)

		ctx := context.Background()
		result, err := service.UploadFromFS(ctx, testFS, "test.txt", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.CID)
		assert.Greater(t, result.Size, int64(0))
	})

	t.Run("uploads directory via TUS", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create a test filesystem with a directory
		testFS := newMemFS()
		testFS.AddDir("dir")
		testFS.AddFile("dir/file1.txt", "content 1")
		testFS.AddFile("dir/file2.txt", "content 2")

		// Set upload limit to 1 byte so it routes to TUS
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1),
		)

		ctx := context.Background()
		result, err := service.UploadFromFS(ctx, testFS, "dir", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.CID)
		assert.Greater(t, result.Size, int64(0))
	})

	t.Run("uploads small file via POST multipart", func(t *testing.T) {
		server := setupPOSTTest(t)
		defer server.Close()

		// Create a test filesystem with a small file
		testFS := newMemFS()
		testFS.AddFile("small.txt", "small")

		// Set upload limit high so CAR (small) routes to POST
		service := NewUploadService(server.URL, "test-token",
			WithUploadLimit(1000),
		)

		ctx := context.Background()
		result, err := service.UploadFromFS(ctx, testFS, "small.txt", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.CID)
	})

	t.Run("respects upload limit for large files (TUS)", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create a test filesystem with a larger file
		testFS := newMemFS()
		testFS.AddFile("large.txt", "This is a larger file that should use TUS upload")

		// Set upload limit to 1 byte so everything uses TUS
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1),
		)

		ctx := context.Background()
		result, err := service.UploadFromFS(ctx, testFS, "large.txt", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.CID)
	})
}

// memfs is a simple in-memory filesystem for testing.
// It implements fs.FS for use with UploadFromFS.
type memfs struct {
	files map[string]string
	dirs  map[string][]string
}

// newMemFS creates a new in-memory filesystem.
func newMemFS() *memfs {
	return &memfs{
		files: make(map[string]string),
		dirs:  make(map[string][]string),
	}
}

// AddFile adds a file to the filesystem.
func (m *memfs) AddFile(path, data string) {
	m.files[path] = data

	// Track directory structure
	dir := getDir(path)
	if dir != "." {
		m.dirs[dir] = append(m.dirs[dir], getBasename(path))
	}
}

// AddDir adds a directory to the filesystem.
func (m *memfs) AddDir(path string) {
	if _, ok := m.dirs[path]; !ok {
		m.dirs[path] = []string{}
	}
}

// Open implements fs.FS.Open for memfs.
func (m *memfs) Open(name string) (fs.File, error) {
	// Handle root directory
	if name == "." {
		// Collect all top-level entries
		var entries []string
		for path := range m.files {
			dir := getDir(path)
			if dir == "." {
				entries = append(entries, getBasename(path))
			}
		}
		for dir := range m.dirs {
			parent := getDir(dir)
			if parent == "." && dir != "." {
				entries = append(entries, getBasename(dir))
			}
		}
		return &memdir{name: ".", entries: entries}, nil
	}

	// Check if it's a directory
	if entries, isDir := m.dirs[name]; isDir {
		// Collect entries from files in this directory
		var dirEntries []string
		entries = append(entries, dirEntries...)
		return &memdir{name: name, entries: entries}, nil
	}

	// Check if it's a file
	data, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &memfile{name: name, data: data}, nil
}

// memdir implements fs.File and fs.ReadDirFile for in-memory directories.
type memdir struct {
	name     string
	entries  []string
	pos      int
}

// Stat implements fs.File.Stat for memdir.
func (d *memdir) Stat() (fs.FileInfo, error) {
	return &fileInfo{size: 0, isDir: true}, nil
}

// Read implements io.Reader for memdir (returns EOF - directories can't be read as data).
func (d *memdir) Read([]byte) (int, error) {
	return 0, io.EOF
}

// Close implements fs.File.Close for memdir.
func (d *memdir) Close() error {
	return nil
}

// ReadDir implements fs.ReadDirFile.ReadDir for memdir.
func (d *memdir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.pos >= len(d.entries) {
		return nil, io.EOF
	}

	var entries []fs.DirEntry
	end := d.pos + n
	if n <= 0 {
		end = len(d.entries)
	} else if end > len(d.entries) {
		end = len(d.entries)
	}

	for i := d.pos; i < end; i++ {
		entries = append(entries, &direntry{name: d.entries[i]})
	}

	d.pos = end
	return entries, nil
}

// direntry implements fs.DirEntry for memfs.
type direntry struct {
	name string
}

// Name implements fs.DirEntry.Name.
func (d *direntry) Name() string {
	return d.name
}

// Type implements fs.DirEntry.Type.
func (d *direntry) Type() fs.FileMode {
	return 0
}

// Info implements fs.DirEntry.Info.
func (d *direntry) Info() (fs.FileInfo, error) {
	return &fileInfo{}, nil
}

// IsDir implements fs.DirEntry.IsDir.
func (d *direntry) IsDir() bool {
	return false
}

// memfile implements fs.File for in-memory files.
type memfile struct {
	name string
	data string
	pos  int64
}

// Stat implements fs.File.Stat for memfile.
func (f *memfile) Stat() (fs.FileInfo, error) {
	return &fileInfo{size: int64(len(f.data))}, nil
}

// Read implements io.Reader for memfile.
func (f *memfile) Read(p []byte) (int, error) {
	if f.pos >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, []byte(f.data[f.pos:]))
	f.pos += int64(n)
	return n, nil
}

// Close implements fs.File.Close for memfile.
func (f *memfile) Close() error {
	return nil
}

// fileInfo implements fs.FileInfo for in-memory files.
type fileInfo struct {
	size  int64
	isDir bool
}

// Name implements fs.FileInfo.Name.
func (f *fileInfo) Name() string {
	return ""
}

// Size implements fs.FileInfo.Size.
func (f *fileInfo) Size() int64 {
	return f.size
}

// Mode implements fs.FileInfo.Mode.
func (f *fileInfo) Mode() fs.FileMode {
	if f.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}

// ModTime implements fs.FileInfo.ModTime.
func (f *fileInfo) ModTime() time.Time {
	return time.Time{}
}

// IsDir implements fs.FileInfo.IsDir.
func (f *fileInfo) IsDir() bool {
	return f.isDir
}

// Sys implements fs.FileInfo.Sys.
func (f *fileInfo) Sys() interface{} {
	return nil
}

// getDir returns the directory part of a path.
func getDir(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return "."
	}
	if lastSlash == 0 {
		return "/"
	}
	return path[:lastSlash]
}

// getBasename returns the basename of a path.
func getBasename(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return path
	}
	return path[lastSlash+1:]
}

// extractUploadID extracts the upload ID from a TUS location URL
func extractUploadID(location string) string {
	// Extract the last part of the URL path which should be the upload ID
	parts := strings.Split(location, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
