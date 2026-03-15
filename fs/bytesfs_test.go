package fs

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesFSBasic(t *testing.T) {
	data := []byte("Hello, World!")
	fs := NewBytesFS(data, "test.txt")

	// Open directory
	dir, err := fs.Open(".")
	require.NoError(t, err)
	assert.Nil(t, dir.Close())

	// Open file
	file, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file.Close()

	buf := make([]byte, 13)
	n, err := file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 13, n)
	assert.Equal(t, data, buf[:n])
}

func TestBytesFileSeek(t *testing.T) {
	data := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	fs := NewBytesFS(data, "test.txt")

	file, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file.Close()

	// Assert that file implements io.Seeker
	seeker, ok := file.(io.Seeker)
	require.True(t, ok, "file should implement io.Seeker")

	// Test SeekStart
	offset, err := seeker.Seek(10, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(10), offset)

	buf := make([]byte, 5)
	n, err := file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("KLMNO"), buf)

	// Test SeekCurrent
	offset, err = seeker.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(15), offset)

	// Test SeekEnd
	offset, err = seeker.Seek(-5, io.SeekEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(21), offset)

	n, err = file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("VWXYZ"), buf)
}

func TestBytesFileSeekOutOfBounds(t *testing.T) {
	data := []byte("Hello, World!")
	fs := NewBytesFS(data, "test.txt")

	file, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file.Close()

	// Assert that file implements io.Seeker
	seeker, ok := file.(io.Seeker)
	require.True(t, ok, "file should implement io.Seeker")

	// Seek to position beyond file length
	offset, err := seeker.Seek(100, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(100), offset)

	// Reading should return EOF
	buf := make([]byte, 10)
	n, err := file.Read(buf)
	require.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)

	// Negative offset should be clamped to 0
	offset, err = seeker.Seek(-100, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)
}

func TestBytesFileSeekInvalidWhence(t *testing.T) {
	data := []byte("test")
	fs := NewBytesFS(data, "test.txt")

	file, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file.Close()

	// Assert that file implements io.Seeker
	seeker, ok := file.(io.Seeker)
	require.True(t, ok, "file should implement io.Seeker")

	_, err = seeker.Seek(0, 99)
	assert.Equal(t, "invalid whence parameter", err.Error())
}

func TestBytesDirNoSeek(t *testing.T) {
	data := []byte("test")
	fs := NewBytesFS(data, "test.txt")

	dir, err := fs.Open(".")
	require.NoError(t, err)
	defer dir.Close()

	seeker, ok := dir.(io.Seeker)
	require.True(t, ok, "dir should implement io.Seeker")

	_, err = seeker.Seek(0, io.SeekStart)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "seek not supported")
}

func TestBytesFileReOpen(t *testing.T) {
	data := []byte("Hello, World!")
	fs := NewBytesFS(data, "test.txt")

	// First open and read
	file1, err := fs.Open("test.txt")
	require.NoError(t, err)

	buf1 := make([]byte, 5)
	n1, err := file1.Read(buf1)
	require.NoError(t, err)
	assert.Equal(t, 5, n1)
	assert.Equal(t, []byte("Hello"), buf1)

	// Close and reopen - should start from beginning
	err = file1.Close()
	require.NoError(t, err)

	file2, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file2.Close()

	buf2 := make([]byte, 5)
	n2, err := file2.Read(buf2)
	require.NoError(t, err)
	assert.Equal(t, 5, n2)
	assert.Equal(t, []byte("Hello"), buf2)
}

func TestBytesFSReOpeningDuringCARGeneration(t *testing.T) {
	// Simulate the two-pass CAR generation pattern with BytesFS
	data := []byte("This is a larger file to test re-opening behavior pattern")
	fs := NewBytesFS(data, "test.txt")

	// Pass 1: Read during BuildSummary
	file1, err := fs.Open("test.txt")
	require.NoError(t, err)

	buf1 := make([]byte, len(data))
	n1, _ := file1.Read(buf1)
	require.NoError(t, file1.Close())

	// Simulate LRU eviction (conceptually)
	_ = n1 // Data was read and stored in blockstore

	// Pass 2: Reopen for regeneration
	file2, err := fs.Open("test.txt")
	require.NoError(t, err)
	defer file2.Close()

	buf2 := make([]byte, len(data))
	n2, err := file2.Read(buf2)
	require.NoError(t, err)

	// Verify we get the same data
	assert.Equal(t, len(data), n2)
	assert.Equal(t, data, buf2)
}
