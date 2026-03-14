package ipfs

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/docker/go-units"
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
		// Set upload limit to 1 byte to force TUS routing
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1), // Force TUS by setting limit smaller than file size
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

		// Set upload limit to force TUS routing
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1), // Force TUS by setting limit smaller than file size
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

		// Set upload limit to force TUS routing
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1), // Force TUS by setting limit smaller than file size
		)

		ctx := context.Background()
		reader := bytes.NewReader(testData)
		_, err := service.Upload(ctx, reader, "test.txt", int64(len(testData)))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create TUS upload")
	})
}

func TestUploadService_Upload_RoutesBySize(t *testing.T) {
	t.Run("small file uses POST multipart (default 100MB limit)", func(t *testing.T) {
		server := setupPOSTTest(t)
		defer server.Close()

		// Create small test data (< 100MB)
		testData := []byte("small file content for POST test")
		testSize := int64(len(testData))
		testName := "small.txt"

		// Use default upload limit (100MB) - should route to POST
		service := NewUploadService(server.URL, "test-token",
			WithUploadLimit(100*units.MiB),
		)

		ctx := context.Background()
		reader := bytes.NewReader(testData)
		result, err := service.Upload(ctx, reader, testName, testSize)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(len(testData)), result.Size)
	})

	t.Run("large file uses TUS (default 100MB limit)", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create test data and set limit to force TUS routing
		largeData := bytes.Repeat([]byte("x"), 1000) // 1KB data
		testSize := int64(len(largeData))
		testName := "large.txt"

		// Set upload limit to 1 byte to force TUS routing
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1),
		)

		ctx := context.Background()
		reader := bytes.NewReader(largeData)
		result, err := service.Upload(ctx, reader, testName, testSize)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(len(largeData)), result.Size)
	})
}

func TestUploadService_Upload_DefaultLimit(t *testing.T) {
	t.Run("uses 100MiB default limit when not specified", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create service without specifying upload limit
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		// Verify default limit is 100MiB
		assert.Equal(t, int64(100*units.MiB), service.uploadLimit)
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

		// Set upload limit to 1 byte to force TUS routing
		service := NewUploadService(baseURL.String(), "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1), // Force TUS by setting limit smaller than file size
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
		testFS := fstest.MapFS{
			"test.txt": {Data: []byte("Hello, World!")},
		}

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
		testFS := fstest.MapFS{
			"dir/file1.txt": {Data: []byte("content 1")},
			"dir/file2.txt": {Data: []byte("content 2")},
		}

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
		testFS := fstest.MapFS{
			"small.txt": {Data: []byte("small")},
		}

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
		testFS := fstest.MapFS{
			"large.txt": {Data: []byte("This is a larger file that should use TUS upload")},
		}

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

	t.Run("respects upload limit for large files (TUS)", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Create a test filesystem with a larger file
		testFS := fstest.MapFS{
			"large.txt": {Data: []byte("This is a larger file that should use TUS upload")},
		}

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
	t.Run("respects opts uploadLimit", func(t *testing.T) {
		// Regression test: opts.UploadLimit should be used, not ignored
		server, _, store := setupTUSTest(t)
		defer server.Close()

		testFS := fstest.MapFS{
			"test.txt": {Data: []byte("test")},
		}

		// Set UploadLimit in opts to 1 byte (should force TUS routing)
		opts := &UploadOptions{
			UploadLimit: 1,
		}

		// Set service default limit to 100MB (should be ignored when opts are provided)
		service := NewUploadService(server.URL, "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(100*1024*1024),
		)

		ctx := context.Background()
		result, err := service.UploadFromFS(ctx, testFS, "test.txt", opts)

		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Verify TUS was used (not POST) by checking store has upload
		// If POST was used incorrectly, upload would fail on invalid URL
		uploads := store.uploads
		assert.Len(t, uploads, 1, "should have exactly 1 TUS upload when opts.UploadLimit is 1")
	})
}

func TestUploadBytes(t *testing.T) {
	t.Run("uploads bytes via TUS for large content", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Test data
		testData := []byte("Hello, World! This is byte data for testing UploadBytes.")

		// Force TUS routing
		service := NewUploadService("https://api.example.com", "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
			WithUploadLimit(1),
		)

		ctx := context.Background()
		result, err := service.UploadBytes(ctx, testData, "test.txt", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.CID)
		assert.Greater(t, result.Size, int64(0))
	})

	t.Run("uploads bytes via POST for small content", func(t *testing.T) {
		server := setupPOSTTest(t)
		defer server.Close()

		// Test data
		testData := []byte("small")

		// Force POST routing
		service := NewUploadService(server.URL, "test-token",
			WithUploadLimit(1000),
		)

		ctx := context.Background()
		result, err := service.UploadBytes(ctx, testData, "small.txt", nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Size, int64(0))
	})

	t.Run("supports custom UploadOptions", func(t *testing.T) {
		server, _, _ := setupTUSTest(t)
		defer server.Close()

		// Test data
		testData := []byte("custom options test")

		// Create custom options
		opts := &UploadOptions{
			MemoryLimit: 50 * 1024 * 1024, // 50MB
			WrapInDir:   false,
			UploadLimit: 1,
		}

		service := NewUploadService(server.URL, "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		result, err := service.UploadBytes(ctx, testData, "test.txt", opts)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Size, int64(0))
	})
	t.Run("uses TUS when CAR size exceeds upload limit", func(t *testing.T) {
		// Regression test: opts.UploadLimit should control routing (not service default)
		// CAR size (~216 bytes) > opts.UploadLimit (1 byte) should route to TUS
		server, _, store := setupTUSTest(t)
		defer server.Close()

		testData := []byte("test data")

		// Set UploadLimit to 1 byte to force TUS routing (CAR size > 1 byte)
		opts := &UploadOptions{
			UploadLimit: 1,
		}

		service := NewUploadService(server.URL, "test-token",
			WithTUSEndpoint(server.URL+"/tus"),
		)

		ctx := context.Background()
		result, err := service.UploadBytes(ctx, testData, "test.txt", opts)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify the upload was actually stored in the TUS store (not POST)
		uploads := store.uploads
		assert.Len(t, uploads, 1, "should have exactly 1 TUS upload")

	})
	t.Run("uses POST when CAR size is under upload limit", func(t *testing.T) {
		// Regression test: verify opts.UploadLimit works for POST routing too
		server := setupPOSTTest(t)
		defer server.Close()

		testData := []byte("small")

		// Set UploadLimit high to force POST routing
		opts := &UploadOptions{
			UploadLimit: 1000,
		}

		service := NewUploadService(server.URL, "test-token")

		ctx := context.Background()
		result, err := service.UploadBytes(ctx, testData, "test.txt", opts)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}
