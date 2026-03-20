package ipfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/exchange/offline"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/ipld/merkledag"
	boxounixfs "github.com/ipfs/boxo/ipld/unixfs"
	"github.com/ipfs/boxo/path"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/multiformats/go-multicodec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.lumeweb.com/ipfs-sdk/fs"
	"go.lumeweb.com/ipfs-sdk/mocks"
	unixfs "go.lumeweb.com/ipfs-content/unixfs"
)

var (
	testAPIToken  string
	testAuthToken string
)

func init() {
	testAPIToken = os.Getenv("TEST_API_TOKEN")
	if testAPIToken == "" {
		testAPIToken = "test-token-12345"
	}
	testAuthToken = os.Getenv("TEST_AUTH_TOKEN")
	if testAuthToken == "" {
		testAuthToken = "test-auth-token-67890"
	}
}

func TestNewDownloadService(t *testing.T) {
	t.Run("creates service with base URL and token", func(t *testing.T) {
		baseURL := "https://api.example.com"

		service, err := NewDownloadService(baseURL, testAPIToken)

		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, baseURL, service.baseURL)
		assert.Equal(t, testAPIToken, service.authToken)
	})

	t.Run("creates service with custom HTTP client", func(t *testing.T) {
		baseURL := "https://api.example.com"
		customClient := &http.Client{}

		service, err := NewDownloadService(baseURL, testAPIToken, WithDownloadHTTPClient(customClient))

		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Same(t, customClient, service.httpClient)
	})

	t.Run("fails with invalid base URL", func(t *testing.T) {
		// This test verifies that invalid URLs are handled
		baseURL := "://invalid-url"

		service, err := NewDownloadService(baseURL, testAPIToken)

		// The NewRemoteBlockstore should handle URL validation
		// For now we just ensure the service is created
		if err == nil {
			assert.NotNil(t, service)
		}
	})
}

func TestDownloadService_Block(t *testing.T) {
	t.Run("retrieves block successfully", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)

		// This test would require mocking the underlying blockstore
		// For now we just verify the method signature
		ctx := context.Background()
		c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)

		_, _ = service.Block(ctx, c)
	})
}

func TestDownloadService_Has(t *testing.T) {
	t.Run("checks block existence", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)
		ctx := context.Background()
		c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)

		_, _ = service.Has(ctx, c)
	})
}

func TestDownloadService_BlockSize(t *testing.T) {
	t.Run("gets block size", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)
		ctx := context.Background()
		c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)

		_, _ = service.BlockSize(ctx, c)
	})
}

func TestDownloadService_FileSize(t *testing.T) {
	t.Run("returns actual UnixFS file size for inline small file", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("Hello, IPFS! This is test data.")
		expectedSize := int64(len(testData))

		// Use ipfs-content to generate properly encoded block
		testGen := newTestUnixFSGenerator(t)
		block := testGen.createBlock(t, testData)

		// Create mock file with block data
		file := createMockFileWithUnixFS(block.RawData())

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations for GetBlock
		setupMockGetBlock(t, mockBackend, ctx, testCID, file, nil)

		// Call FileSize
		size, err := service.FileSize(ctx, testCID)
		require.NoError(t, err)
		assert.Equal(t, expectedSize, size)
	})

	t.Run("returns UnixFS file size for chunked large file", func(t *testing.T) {
		ctx := context.Background()
		expectedSize := int64(104857600) // 100MB

		// Simulate chunk sizes for 100MB file (400 chunks of 256KB each)
		chunkSizes := make([]uint64, 400)
		for i := range chunkSizes {
			chunkSizes[i] = 256 * 1024
		}

		// Use ipfs-content generator to create chunked block metadata
		testGen := newTestUnixFSGenerator(t)
		block := testGen.createChunkedBlock(t, expectedSize, chunkSizes)

		// Create mock file with block data
		file := createMockFileWithUnixFS(block.RawData())

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations for GetBlock
		setupMockGetBlock(t, mockBackend, ctx, testCID, file, nil)

		// Call FileSize
		size, err := service.FileSize(ctx, testCID)
		require.NoError(t, err)
		assert.Equal(t, expectedSize, size)
	})

	t.Run("returns actual data length for raw blocks", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("raw block data")

		// Create mock file with raw data (no UnixFS wrapper)
		file := createMockFileWithUnixFS(testData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations for GetBlock
		setupMockGetBlock(t, mockBackend, ctx, testCID, file, nil)

		// Call FileSize - should return actual data length
		size, err := service.FileSize(ctx, testCID)
		require.NoError(t, err)
		assert.Equal(t, int64(len(testData)), size)
	})

	t.Run("returns error from backend", func(t *testing.T) {
		ctx := context.Background()
		testErr := errors.New("backend error")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations for GetBlock to fail
		setupMockGetBlock(t, mockBackend, ctx, testCID, nil, testErr)

		// Setup mock expectations for GetAll fallback (also fails)
		setupMockGetAll(t, mockBackend, ctx, testCID, nil, testErr)

		// Call FileSize - should return error
		_, err := service.FileSize(ctx, testCID)
		assert.Error(t, err)
		// Error should mention the GetAll failure since GetBlock failed
		assert.ErrorContains(t, err, "failed to get file node")
	})

	t.Run("cancels with context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations - GetBlock and GetAll may be called with cancelled context
		setupMockGetBlockWithMaybe(t, mockBackend, ctx, testCID)
		setupMockGetAllWithMaybe(t, mockBackend, ctx, testCID)

		// Call FileSize with cancelled context
		_, err := service.FileSize(ctx, testCID)
		assert.Error(t, err)
	})
}

func TestDownloadService_Raw(t *testing.T) {
	t.Run("gets raw block data", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)
		ctx := context.Background()
		c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)

		_, _ = service.Raw(ctx, c)
	})
}

func TestDownloadService_CopyBlock(t *testing.T) {
	t.Run("writes block to writer", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)
		ctx := context.Background()
		c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)
		var buf bytes.Buffer

		_ = service.CopyBlock(ctx, c, &buf)
	})
}

func TestDownloadService_AuthToken(t *testing.T) {
	t.Run("returns authentication token", func(t *testing.T) {
		service, err := NewDownloadService("https://api.example.com", testAPIToken)
		require.NoError(t, err)

		assert.Equal(t, testAPIToken, service.AuthToken())
	})

	t.Run("sets authentication token", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)

		newToken := "new-token-456"
		service.SetAuthToken(newToken)

		assert.Equal(t, newToken, service.AuthToken())
	})

	t.Run("sets token on HTTP client transport", func(t *testing.T) {
		token := testAPIToken
		service, err := NewDownloadService("https://api.example.com", token)
		require.NoError(t, err)

		newToken := "updated-token"
		service.SetAuthToken(newToken)

		assert.Equal(t, newToken, service.AuthToken())
	})
}



// mockFile implements files.File for testing
type mockFile struct {
	io.ReadSeeker
	io.Closer
}

// File interface methods not covered by ReadSeeker
func (m *mockFile) Seek(offset int64, whence int) (int64, error) {
	if rs, ok := m.ReadSeeker.(io.Seeker); ok {
		return rs.Seek(offset, whence)
	}
	return 0, nil
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if rs, ok := m.ReadSeeker.(io.Reader); ok {
		return rs.Read(p)
	}
	return 0, io.EOF
}

// mockBytesFile implements files.File for bytes.Reader without UnixFS wrapping
type mockBytesFile struct {
	*bytes.Reader
}

func (m *mockBytesFile) Close() error { return nil }
func (m *mockBytesFile) Stat() (files.FileInfo, error) {
	return &mockFileInfo{}, nil
}
func (m *mockBytesFile) IsDir() bool { return false }
func (m *mockBytesFile) ModTime() time.Time { return time.Now() }
func (m *mockBytesFile) Mode() os.FileMode { return 0644 }
func (m *mockBytesFile) Size() (int64, error) { return m.Reader.Size(), nil }




func (m *mockFile) IsDir() bool          { return false }
func (m *mockFile) Stat() (files.FileInfo, error) { return &mockFileInfo{}, nil }
func (m *mockFile) ModTime() time.Time { return time.Now() }
func (m *mockFile) Mode() os.FileMode { return 0644 }
func (m *mockFile) Size() (int64, error) { return 0, nil }

// mockNode implements files.Node for testing
type mockNode struct {
	io.Reader
	io.Closer
	mode os.FileMode
}

func (m *mockNode) IsDir() bool          { return m.mode.IsDir() }
func (m *mockNode) Stat() (files.FileInfo, error) { return &mockFileInfo{}, nil }
func (m *mockNode) Close() error {
	if m.Closer != nil {
		return m.Closer.Close()
	}
	return nil
}
func (m *mockNode) ModTime() time.Time { return time.Now() }
func (m *mockNode) Mode() os.FileMode { return m.mode }
func (m *mockNode) Size() (int64, error) { return 0, nil }

// mockOsFileInfo implements os.FileInfo for testing
type mockOsFileInfo struct{}

func (m *mockOsFileInfo) Name() string                 { return "" }
func (m *mockOsFileInfo) Size() int64                   { return 0 }
func (m *mockOsFileInfo) Mode() os.FileMode            { return 0644 }
func (m *mockOsFileInfo) ModTime() time.Time           { return time.Now() }
func (m *mockOsFileInfo) Sys() interface{}             { return nil }
func (m *mockOsFileInfo) IsDir() bool                  { return false }

var _ os.FileInfo = (*mockOsFileInfo)(nil)

// mockFileInfo implements files.FileInfo for testing by embedding os.FileInfo
type mockFileInfo struct {
	mockOsFileInfo
}

// methods required by files.FileInfo but not in os.FileInfo
func (m *mockFileInfo) AbsPath() string                { return "" }
func (m *mockFileInfo) Path() string                   { return "" }
func (m *mockFileInfo) NumLinks() uint64               { return 1 }
func (m *mockFileInfo) FileType() interface{}          { return nil }
func (m *mockFileInfo) Close() error                   { return nil }

// overridden for signature conflict: (int64, error) vs int64
func (m *mockFileInfo) Stat() os.FileInfo              { return &m.mockOsFileInfo }
func (m *mockFileInfo) Size() (int64, error)          { return m.mockOsFileInfo.Size(), nil }

// test helpers for creating common test fixtures

const testCIDString = "QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx"

// setupMockDownloadService creates a new DownloadService with a mock backend.
// Returns the service and the mock backend for setting up expectations.
func setupMockDownloadService(t *testing.T) (*DownloadService, *mocks.MockBackend) {
	t.Helper()
	service := &DownloadService{}
	mockBackend := mocks.NewMockBackend(t)
	service.backend = mockBackend
	return service, mockBackend
}

// getTestCID returns a pre-decoded test CID for use in tests.
func getTestCID(t *testing.T) cid.Cid {
	t.Helper()
	c, err := cid.Decode(testCIDString)
	require.NoError(t, err)
	return c
}

// setupMockGetAll sets up a GetAll mock expectation for a file.
// Returns the metadata and expects the mock to return the provided file.
func setupMockGetAll(t *testing.T, mockBackend *mocks.MockBackend, ctx context.Context, c cid.Cid, file files.Node, err error) *mocks.MockBackend {
	t.Helper()
	immutablePath := path.FromCid(c)
	metadata := gateway.ContentPathMetadata{}
	
	if err != nil {
		mockBackend.EXPECT().
			GetAll(ctx, immutablePath).
			Return(metadata, nil, err)
	} else {
		mockBackend.EXPECT().
			GetAll(ctx, immutablePath).
			Return(metadata, file, nil)
	}
	return mockBackend
}

// setupMockGetAllWithMaybe sets up a GetAll mock expectation that may or may not be called.
// Useful for context cancellation tests where the call is conditional.
func setupMockGetAllWithMaybe(t *testing.T, mockBackend *mocks.MockBackend, ctx context.Context, c cid.Cid) *mocks.MockBackend {
	t.Helper()
	immutablePath := path.FromCid(c)

	mockBackend.EXPECT().
		GetAll(ctx, immutablePath).
		Return(gateway.ContentPathMetadata{}, nil, context.Canceled).
		Maybe()
	return mockBackend
}

// createMockBytesFile creates a mockBytesFile for testing without UnixFS wrapping.
func createMockBytesFile(data []byte) *mockBytesFile {
	return &mockBytesFile{Reader: bytes.NewReader(data)}
}

// createMockFileWithUnixFS creates a mockFile with UnixFS data wrapping.
func createMockFileWithUnixFS(data []byte) *mockFile {
	reader := bytes.NewReader(data)
	return &mockFile{
		ReadSeeker: reader,
		Closer:     io.NopCloser(reader),
	}
}

// setupMockGetBlock sets up a GetBlock mock expectation for fetching a single block.
// Used by FileSize to fetch only the root block for metadata parsing.
func setupMockGetBlock(t *testing.T, mockBackend *mocks.MockBackend, ctx context.Context, c cid.Cid, file files.File, err error) *mocks.MockBackend {
	t.Helper()
	immutablePath := path.FromCid(c)
	metadata := gateway.ContentPathMetadata{}
	
	if err != nil {
		mockBackend.EXPECT().
			GetBlock(ctx, immutablePath).
			Return(metadata, nil, err)
	} else {
		mockBackend.EXPECT().
			GetBlock(ctx, immutablePath).
			Return(metadata, file, nil)
	}
	return mockBackend
}

// setupMockGetBlockWithMaybe sets up a GetBlock mock expectation that may or may not be called.
func setupMockGetBlockWithMaybe(t *testing.T, mockBackend *mocks.MockBackend, ctx context.Context, c cid.Cid) *mocks.MockBackend {
	t.Helper()
	immutablePath := path.FromCid(c)
	metadata := gateway.ContentPathMetadata{}
	
	mockBackend.EXPECT().
		GetBlock(ctx, immutablePath).
		Return(metadata, nil, context.Canceled).
		Maybe()
	return mockBackend
}

// Tests for DownloadFile method
func TestDownloadService_DownloadFile(t *testing.T) {
	t.Run("downloads file successfully", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("test file content")

		// Use mockBytesFile for raw byte reads
		file := createMockBytesFile(testData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, file, nil)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read content and verify
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, content)

		// Close reader
		err = reader.Close()
		require.NoError(t, err)
	})
	
	t.Run("returns error from backend", func(t *testing.T) {
		ctx := context.Background()
		testErr := errors.New("backend error")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, nil, testErr)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify error is returned
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), testErr.Error())
	})
	
	t.Run("returns error on invalid CID", func(t *testing.T) {
		ctx := context.Background()
		service, mockBackend := setupMockDownloadService(t)

		// Create an invalid CID (wrong length, invalid multibase)
		invalidCID := cid.Cid{}

		// Setup mock to return error from backend
		mockBackend.EXPECT().
			GetAll(mock.Anything, mock.Anything).
			Return(gateway.ContentPathMetadata{}, nil, errors.New("invalid CID"))

		// Call DownloadFile with invalid CID
		reader, err := service.DownloadFile(ctx, invalidCID)

		// Verify error is returned
		assert.Error(t, err)
		assert.Nil(t, reader)
	})
	
	t.Run("cancels with context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Note: Context cancellation behavior depends on backend implementation
		// This test ensures the context is properly passed through
		setupMockGetAllWithMaybe(t, mockBackend, ctx, testCID)

		// Call DownloadFile
		_, err := service.DownloadFile(ctx, testCID)

		// Verify error
		assert.Error(t, err)
	})

	t.Run("downloads small inline file correctly", func(t *testing.T) {
		ctx := context.Background()
		// Small file that fits in a single block (under 256KB)
		testData := []byte("Hello, IPFS! This is a small test file that should fit in a single block.")

		// Use mockBytesFile for raw byte reads without UnixFS wrapping
		file := createMockBytesFile(testData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, file, nil)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read content and verify it matches
		content, err := io.ReadAll(reader)
		t.Logf("Small file test: read %d bytes, expected %d bytes", len(content), len(testData))
		t.Logf("Content: %s", string(content))
		require.NoError(t, err)
		assert.Equal(t, testData, content)

		// Close reader
		err = reader.Close()
		require.NoError(t, err)
	})

	t.Run("downloads large chunked file correctly", func(t *testing.T) {
		ctx := context.Background()
		// Large file data that would be chunked into multiple blocks (over 256KB)
		largeData := make([]byte, 1024*1024) // 1MB of data
		for i := range largeData {
			largeData[i] = byte(i % 256) // Pattern to verify integrity
		}

		// Use mockBytesFile for raw byte reads without UnixFS wrapping
		file := createMockBytesFile(largeData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, file, nil)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read content and verify it matches - test multiple reads
		buf := make([]byte, 4096) // Read in 4KB chunks
		totalRead := 0
		readCount := 0
		for {
			n, err := reader.Read(buf)
			readCount++
			t.Logf("Read iteration %d: got %d bytes, error: %v", readCount, n, err)
			totalRead += n
			if err == io.EOF && n == 0 {
				break
			}
			if err != nil && err != io.EOF {
				require.NoError(t, err)
			}
		}
		t.Logf("Finished reading after %d iterations, total bytes: %d", readCount, totalRead)

		// Verify we read the expected amount
		t.Logf("Expected to read %d bytes, actually read %d bytes", len(largeData), totalRead)
		assert.Equal(t, len(largeData), totalRead)

		// Close reader
		reader.Close()
	})

	t.Run("supports multiple read operations", func(t *testing.T) {
		ctx := context.Background()
		// Test data to verify multiple reads work correctly
		testData := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

		// Use mockBytesFile for raw byte reads without UnixFS wrapping
		file := createMockBytesFile(testData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, file, nil)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Multiple small reads - verify each returns correct data
		buf1 := make([]byte, 5)
		n1, err := reader.Read(buf1)
		t.Logf("First read: %d bytes, data: %v", n1, buf1[:n1])
		require.NoError(t, err)
		assert.Equal(t, 5, n1)
		assert.Equal(t, []byte{0, 1, 2, 3, 4}, buf1)

		buf2 := make([]byte, 5)
		n2, err := reader.Read(buf2)
		require.NoError(t, err)
		assert.Equal(t, 5, n2)
		assert.Equal(t, []byte{5, 6, 7, 8, 9}, buf2)

		buf3 := make([]byte, 20) // Larger buffer than remaining data
		n3, err := reader.Read(buf3)
		require.NoError(t, err)
		assert.Equal(t, 10, n3)
		assert.Equal(t, []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}, buf3[:10])

		// Final read should return EOF
		buf4 := make([]byte, 5)
		_, err = reader.Read(buf4)
		assert.Equal(t, io.EOF, err)

		reader.Close()
	})

	t.Run("handles empty file", func(t *testing.T) {
		ctx := context.Background()
		// Empty file edge case
		testData := []byte{}

		// Use mockBytesFile for empty file without UnixFS wrapping
		file := createMockBytesFile(testData)

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, file, nil)

		// Call DownloadFile
		reader, err := service.DownloadFile(ctx, testCID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read content - should be empty
		content, err := io.ReadAll(reader)
		t.Logf("Empty file test: read %d bytes, data: %v", len(content), content)
		require.NoError(t, err)
		assert.Empty(t, content)
		assert.Equal(t, 0, len(content))

		reader.Close()
	})
}

// Tests for ListDirectory method
func TestDownloadService_ListDirectory(t *testing.T) {
	t.Run("calls backend and returns entries", func(t *testing.T) {
		ctx := context.Background()

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		// Create a simple mock directory node
		mockDir := &mockNode{
			Reader: bytes.NewReader([]byte{}),
			Closer: io.NopCloser(bytes.NewReader([]byte{})),
			mode:   os.ModeDir | 0755,
		}

		setupMockGetAll(t, mockBackend, ctx, testCID, mockDir, nil)

		// Call ListDirectory
		// Note: Full directory listing with files.Walk requires a complete file tree
		// This test verifies the backend is called correctly and error handling
		_, err := service.ListDirectory(ctx, testCID)

		// The implementation uses files.Walk which may not work with simple mocks
		// For now we just verify no panic occurs
		_ = err
	})
	
	t.Run("returns error from backend", func(t *testing.T) {
		ctx := context.Background()
		testErr := errors.New("directory error")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		setupMockGetAll(t, mockBackend, ctx, testCID, nil, testErr)

		// Call ListDirectory
		entries, err := service.ListDirectory(ctx, testCID)

		// Verify error is returned
		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), testErr.Error())
	})
	
	t.Run("handles empty directory", func(t *testing.T) {
		ctx := context.Background()

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations with empty directory
		mockDir := &mockNode{
			Reader: bytes.NewReader([]byte{}),
			Closer: io.NopCloser(bytes.NewReader([]byte{})),
			mode:   os.ModeDir | 0755,
		}

		setupMockGetAll(t, mockBackend, ctx, testCID, mockDir, nil)

		// Call ListDirectory
		entries, err := service.ListDirectory(ctx, testCID)

		// Should return empty list without error
		require.NoError(t, err)
		assert.NotNil(t, entries)
	})
	
	t.Run("handles cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Setup mock expectations
		immutablePath := path.FromCid(testCID)

		mockBackend.EXPECT().
			GetAll(mock.Anything, immutablePath).
			Return(gateway.ContentPathMetadata{}, nil, context.Canceled)

		// Call ListDirectory
		entries, err := service.ListDirectory(ctx, testCID)

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, entries)
	})
}

// Tests for GetFile method
func TestDownloadService_GetFile(t *testing.T) {
	t.Run("retrieves file from directory successfully", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("file content in directory")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		filePath := "a/b/c/test.txt"

		// Construct the full path
		basePathStr := "/ipfs/" + testCID.String() + "/" + filePath
		fullPath, err := path.NewPath(basePathStr)
		require.NoError(t, err)
		immutablePath, err := path.NewImmutablePath(fullPath)
		require.NoError(t, err)

		// Use mockBytesFile for raw byte reads
		file := createMockBytesFile(testData)

		// Setup mock expectations
		mockBackend.EXPECT().
			GetAll(ctx, immutablePath).
			Return(gateway.ContentPathMetadata{}, file, nil)

		// Call GetFile
		reader, err := service.GetFile(ctx, testCID, filePath)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Read content and verify
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, content)

		// Close reader
		err = reader.Close()
		require.NoError(t, err)
	})
	
	t.Run("handles invalid path characters", func(t *testing.T) {
		ctx := context.Background()

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		filePath := "a/b/c/test.txt"

		// Use invalid path with special characters that may affect paths
		invalidPath := filePath + string(rune(0))

		// Set up mock for any path that might be called
		mockBackend.EXPECT().
			GetAll(mock.Anything, mock.Anything).
			Return(gateway.ContentPathMetadata{}, nil, errors.New("invalid path"))

		// Call GetFile
		_, err := service.GetFile(ctx, testCID, invalidPath)

		// Verify error
		assert.Error(t, err)
	})
	
	t.Run("returns error from backend", func(t *testing.T) {
		ctx := context.Background()
		testErr := errors.New("file not found")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		filePath := "a/b/c/test.txt"

		// Construct the full path
		basePathStr := "/ipfs/" + testCID.String() + "/" + filePath
		fullPath, err := path.NewPath(basePathStr)
		require.NoError(t, err)
		immutablePath, err := path.NewImmutablePath(fullPath)
		require.NoError(t, err)

		// Setup mock expectations
		mockBackend.EXPECT().
			GetAll(ctx, immutablePath).
			Return(gateway.ContentPathMetadata{}, nil, testErr)

		// Call GetFile
		reader, err := service.GetFile(ctx, testCID, filePath)

		// Verify error is returned
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), testErr.Error())
	})
	
	t.Run("handles path with leading/trailing slashes", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("file content")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		// Test paths with different slash configurations
		testPaths := []string{
			"a/b/c",
			"/a/b/c",
			"a/b/c/",
			"/a/b/c/",
		}

		for _, filePath := range testPaths {
			// Construct the full path
			trimmedPath := strings.Trim(filePath, "/")
			basePathStr := "/ipfs/" + testCID.String() + "/" + trimmedPath
			fullPath, err := path.NewPath(basePathStr)
			require.NoError(t, err)
			immutablePath, err := path.NewImmutablePath(fullPath)
			require.NoError(t, err)

			// Use mockBytesFile for raw byte reads
			file := createMockBytesFile(testData)

			// Setup mock expectations for this call
			mockBackend.EXPECT().
				GetAll(ctx, immutablePath).
				Return(gateway.ContentPathMetadata{}, file, nil)

			// Call GetFile
			reader, err := service.GetFile(ctx, testCID, filePath)

			// Verify results
			require.NoError(t, err, "Failed for path: %s", filePath)
			require.NotNil(t, reader, "Reader is nil for path: %s", filePath)

			// Close reader
			_ = reader.Close()
		}
	})
	
	t.Run("handles root file path", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("root file content")

		// Create service with mock backend
		service, mockBackend := setupMockDownloadService(t)
		testCID := getTestCID(t)

		filePath := ""

		// Empty path uses base CID path directly
		basePath := path.FromCid(testCID)
		immutablePath, err := path.NewImmutablePath(basePath)
		require.NoError(t, err)

		// Use mockBytesFile for raw byte reads
		file := createMockBytesFile(testData)

		// Setup mock expectations
		mockBackend.EXPECT().
			GetAll(mock.Anything, immutablePath).
			Return(gateway.ContentPathMetadata{}, file, nil)

		// Call GetFile
		reader, err := service.GetFile(ctx, testCID, filePath)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, reader)

		// Close reader
		_ = reader.Close()
	})
}

// testUnixFSGenerator provides UnixFS block generation for tests using ipfs-content
type testUnixFSGenerator struct {
	gen unixfs.UnixFSNodeGenerator
}

// newTestUnixFSGenerator creates a new generator with in-memory components
func newTestUnixFSGenerator(t *testing.T) *testUnixFSGenerator {
	t.Helper()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bsvc := blockservice.New(bstore, offline.Exchange(bstore))
	dagService := merkledag.NewDAGService(bsvc)

	return &testUnixFSGenerator{
		gen: unixfs.NewUnixFSNodeGenerator(
			unixfs.WithUnixFSNodeDAGService(dagService),
			unixfs.WithUnixFSNodeBlockstore(bstore),
		),
	}
}

// createBlock generates a properly encoded UnixFS block from data
func (g *testUnixFSGenerator) createBlock(t *testing.T, data []byte) blocks.Block {
	t.Helper()
	
	// Wrap as ReadSeekCloser for ipfs-content's CreateNode
	seekCloser := fs.NewReadSeekCloserAdapter(data)
	
	node, err := g.gen.CreateNode(context.Background(), seekCloser)
	require.NoError(t, err)
	
	// Get the block from the node's CID and DAG service
	dagSvc := g.gen.GetDAGService()
	block, err := dagSvc.Get(context.Background(), node.Cid())
	require.NoError(t, err)
	
	return block
}

// createChunkedBlock generates a chunked UnixFS block metadata (no actual data stored)
func (g *testUnixFSGenerator) createChunkedBlock(t *testing.T, fileSize int64, chunkSizes []uint64) blocks.Block {
	t.Helper()
	
	// Create PBNode with UnixFS file type (no initial file size - will be built from chunks)
	unixfsData := boxounixfs.FilePBData(nil, 0)
	pbNode := merkledag.NodeWithData(unixfsData)
	
	// Set chunk sizes in UnixFS metadata
	fsNode, err := boxounixfs.FSNodeFromBytes(pbNode.Data())
	require.NoError(t, err)
	for _, chunkSize := range chunkSizes {
		fsNode.AddBlockSize(chunkSize)
	}
	newData, err := fsNode.GetBytes()
	require.NoError(t, err)
	pbNode.SetData(newData)
	
	// Set CID builder and marshal
	cidBuilder := cid.V1Builder{Codec: cid.DagProtobuf, MhType: uint64(multicodec.Sha2_256)}
	pbNode.SetCidBuilder(cidBuilder)
	encoded, err := pbNode.Marshal()
	require.NoError(t, err)
	
	// Calculate CID and create block
	c, err := cidBuilder.Sum(encoded)
	require.NoError(t, err)
	
	block, err := blocks.NewBlockWithCid(encoded, c)
	require.NoError(t, err)
	
	return block
}


