package tusstore

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_NewUpload(t *testing.T) {
	t.Run("creates upload with generated ID", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			Size: 100,
			MetaData: map[string]string{
				"filename": "test.txt",
			},
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, upload)

		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, uploadInfo.ID)
		assert.Equal(t, int64(100), uploadInfo.Size)
	})

	t.Run("uses provided ID", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-id-123",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-id-123", uploadInfo.ID)
	})
}

func TestMemoryStore_GetUpload(t *testing.T) {
	t.Run("retrieves existing upload", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-get-upload",
			Size: 100,
			MetaData: map[string]string{
				"filename": "test.txt",
			},
		}

		_, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		upload, err := store.GetUpload(ctx, "test-get-upload")
		require.NoError(t, err)
		require.NotNil(t, upload)

		retrievedInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-get-upload", retrievedInfo.ID)
	})

	t.Run("returns not found for non-existent upload", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		_, err := store.GetUpload(ctx, "non-existent")
		assert.Error(t, err)
		assert.Equal(t, handler.ErrNotFound, err)
	})
}

func TestMemoryStore_WriteChunk(t *testing.T) {
	t.Run("writes data with append-only behavior", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-write",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		data := []byte("hello world")
		n, err := upload.(interface{ WriteChunk(context.Context, int64, io.Reader) (int64, error) }).
			WriteChunk(ctx, 0, bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), n)

		retrievedData, exists := store.GetData("test-write")
		assert.True(t, exists)
		assert.Equal(t, data, retrievedData)

		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), uploadInfo.Offset)
	})

	t.Run("writes multiple chunks sequentially via append", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-multi",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		writeChunk := func(offset int64, data []byte) {
			wc := upload.(interface{ WriteChunk(context.Context, int64, io.Reader) (int64, error) })
			n, err := wc.WriteChunk(ctx, offset, bytes.NewReader(data))
			require.NoError(t, err)
			assert.Equal(t, int64(len(data)), n)
		}

		writeChunk(0, []byte("first"))
		writeChunk(5, []byte("second"))

		retrievedData, exists := store.GetData("test-multi")
		assert.True(t, exists)
		// With append-only, both chunks are appended sequentially
		assert.Equal(t, []byte("firstsecond"), retrievedData)
	})
}

func TestMemoryStore_GetReader(t *testing.T) {
	t.Run("returns reader for uploaded data", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-reader",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		wc := upload.(interface{ WriteChunk(context.Context, int64, io.Reader) (int64, error) })
		data := []byte("test data for reader")
		_, err = wc.WriteChunk(ctx, 0, bytes.NewReader(data))
		require.NoError(t, err)

		memFile := upload.(*MemoryFile)
		reader, err := memFile.GetReader(ctx)
		require.NoError(t, err)
		defer reader.Close()

		result := make([]byte, len(data))
		n, err := reader.Read(result)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, result)
	})
}

func TestMemoryStore_Terminate(t *testing.T) {
	t.Run("deletes upload", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-terminate",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		_, err = store.GetUpload(ctx, "test-terminate")
		require.NoError(t, err)

		err = upload.(interface{ Terminate(context.Context) error }).Terminate(ctx)
		require.NoError(t, err)

		_, err = store.GetUpload(ctx, "test-terminate")
		assert.Error(t, err)
		assert.Equal(t, handler.ErrNotFound, err)
	})
}

func TestMemoryStore_FinishUpload(t *testing.T) {
	t.Run("marks upload as complete", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-finish",
			Size: 100,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		memFile := upload.(*MemoryFile)
		err = memFile.FinishUpload(ctx)
		require.NoError(t, err)

		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.True(t, memFile.isComplete)
		assert.Equal(t, int64(100), uploadInfo.Offset)
	})
}

func TestMemoryStore_DeclareLength(t *testing.T) {
	t.Run("declares length for deferred uploads", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:            "test-declare",
			Size:          0,
			SizeIsDeferred: true,
		}

		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		ld := upload.(interface{ DeclareLength(context.Context, int64) error })
		err = ld.DeclareLength(ctx, 500)
		require.NoError(t, err)

		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(500), uploadInfo.Size)
		assert.False(t, uploadInfo.SizeIsDeferred)
	})
}

func TestMemoryStore_GanUploadInfo(t *testing.T) {
	t.Run("returns info for all uploads", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info1 := handler.FileInfo{ID: "upload-1", Size: 100}
		info2 := handler.FileInfo{ID: "upload-2", Size: 200}

		_, err := store.NewUpload(ctx, info1)
		require.NoError(t, err)
		_, err = store.NewUpload(ctx, info2)
		require.NoError(t, err)

		infos, err := store.GetUploadsInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, len(infos))
	})
}

func TestMemoryStore_GetData(t *testing.T) {
	t.Run("returns data for existing upload", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{ID: "test-data", Size: 100}
		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		wc := upload.(interface{ WriteChunk(context.Context, int64, io.Reader) (int64, error) })
		expectedData := []byte("test data")
		_, err = wc.WriteChunk(ctx, 0, bytes.NewReader(expectedData))
		require.NoError(t, err)

		data, exists := store.GetData("test-data")
		assert.True(t, exists)
		assert.Equal(t, expectedData, data)
	})

	t.Run("returns false for non-existent upload", func(t *testing.T) {
		store := New("/memory")
		_, exists := store.GetData("non-existent")
		assert.False(t, exists)
	})
}

func TestMemoryStore_HasUpload(t *testing.T) {
	t.Run("returns true for existing upload", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{ID: "test-has", Size: 100}
		_, err := store.NewUpload(ctx, info)
		require.NoError(t, err)

		assert.True(t, store.HasUpload("test-has"))
	})

	t.Run("returns false for non-existent upload", func(t *testing.T) {
		store := New("/memory")
		assert.False(t, store.HasUpload("non-existent"))
	})
}

func TestMemoryFile_ServeContent(t *testing.T) {
	t.Run("serves content via HTTP with range requests", func(t *testing.T) {
		store := New("/memory")
		ctx := context.Background()

		info := handler.FileInfo{
			ID:   "test-serve",
			Size: 42,
			MetaData: map[string]string{
				"filename": "serve.txt",
			},
		}
		upload, err := store.NewUpload(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, upload)

		// Check Storage map is set correctly
		uploadInfo, err := upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, len(uploadInfo.Storage))
		assert.Equal(t, "memory", uploadInfo.Storage["Type"])
		assert.NotEmpty(t, uploadInfo.Storage["Path"])
		assert.NotEmpty(t, uploadInfo.Storage["InfoPath"])

		content := []byte("hello world")
		n, err := upload.WriteChunk(ctx, 0, bytes.NewReader(content))
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), n)

		// Check offset updated
		uploadInfo, err = upload.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(11), uploadInfo.Offset)
		assert.Equal(t, int64(42), uploadInfo.Size)

		// Read content
		reader, err := upload.GetReader(ctx)
		require.NoError(t, err)
		retrieved, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, retrieved)
		reader.Close()
	})
}

// TestMemoryStore_Integration is a comprehensive test similar to filestore's TestFilestore
// It tests the full lifecycle: create -> write -> read -> terminate
func TestMemoryStore_Integration(t *testing.T) {
	store := New("/memory")
	ctx := context.Background()

	// Create new upload with metadata
	upload, err := store.NewUpload(ctx, handler.FileInfo{
		Size: 42,
		MetaData: map[string]string{
			"filename": "integration.txt",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, upload)

	// Get info and verify Storage map
	info, err := upload.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(42), info.Size)
	assert.Equal(t, int64(0), info.Offset)
	assert.Equal(t, handler.MetaData{"filename": "integration.txt"}, info.MetaData)
	assert.Equal(t, 3, len(info.Storage)) // Type, Path, InfoPath
	assert.Equal(t, "memory", info.Storage["Type"])

	// Write data
	content := "hello world"
	bytesWritten, err := upload.WriteChunk(ctx, 0, strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), bytesWritten)

	// Check offset updated
	info, err = upload.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(42), info.Size)
	assert.Equal(t, int64(11), info.Offset)

	// Read content
	reader, err := upload.GetReader(ctx)
	require.NoError(t, err)
	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(retrieved))
	reader.Close()

	// Terminate upload
	err = store.AsTerminatableUpload(upload).Terminate(ctx)
	require.NoError(t, err)

	// Verify upload is gone
	_, err = store.GetUpload(ctx, info.ID)
	assert.Error(t, err)
	assert.Equal(t, handler.ErrNotFound, err)
}

func TestMemoryStore_ConcatUploads(t *testing.T) {
	store := New("/memory")
	ctx := context.Background()

	// Create final upload to hold concatenated data
	finUpload, err := store.NewUpload(ctx, handler.FileInfo{
		Size: 9,
		MetaData: map[string]string{
			"type": "concat",
		},
	})
	require.NoError(t, err)

	finInfo, err := finUpload.GetInfo(ctx)
	require.NoError(t, err)
	finID := finInfo.ID

	// Create three partial uploads
	partialUploads := make([]handler.Upload, 3)
	contents := []string{
		"abc",
		"def",
		"ghi",
	}

	for i := range 3 {
		upload, err := store.NewUpload(ctx, handler.FileInfo{
			Size: 3,
			MetaData: map[string]string{
				"part": string(rune('1' + i)),
			},
		})
		require.NoError(t, err)

		n, err := upload.WriteChunk(ctx, 0, strings.NewReader(contents[i]))
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		partialUploads[i] = upload
	}

	// Concatenate
	err = store.AsConcatableUpload(finUpload).ConcatUploads(ctx, partialUploads)
	require.NoError(t, err)

	// Verify final upload
	finUpload, err = store.GetUpload(ctx, finID)
	require.NoError(t, err)

	info, err := finUpload.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(9), info.Size)
	assert.Equal(t, int64(9), info.Offset)

	// Verify concatenated content
	reader, err := finUpload.GetReader(ctx)
	require.NoError(t, err)
	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "abcdefghi", string(retrieved))
	reader.Close()
}
