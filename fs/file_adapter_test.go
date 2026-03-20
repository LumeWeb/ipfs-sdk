package fs

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReaderCloser implements both io.Reader and io.Closer for testing
type mockReaderCloser struct {
	reader io.Reader
	closed bool
	closeErr error
}

func (m *mockReaderCloser) Read(p []byte) (n int, err error) {
	if m.reader != nil {
		return m.reader.Read(p)
	}
	return 0, io.EOF
}

func (m *mockReaderCloser) Close() error {
	m.closed = true
	return m.closeErr
}

// TestNewFileAdapter verifies the FileAdapter constructor
func TestNewFileAdapter(t *testing.T) {
	t.Run("creates adapter with reader and closer", func(t *testing.T) {
		data := []byte("test data")
		reader := bytes.NewReader(data)
		
		mockedCloser := &mockReaderCloser{reader: reader}
		
		adapter := NewFileAdapter(mockedCloser, mockedCloser)
		
		assert.NotNil(t, adapter)
		assert.Same(t, mockedCloser, adapter.Reader)
		assert.Same(t, mockedCloser, adapter.Closer)
	})
}

// TestFileAdapter_Read verifies reading functionality
func TestFileAdapter_Read(t *testing.T) {
	t.Run("reads data correctly", func(t *testing.T) {
		data := []byte("Hello, World!")
		reader := bytes.NewReader(data)
		mockedCloser := &mockReaderCloser{reader: reader}
		
		adapter := NewFileAdapter(mockedCloser, mockedCloser)
		
		buf := make([]byte, len(data))
		n, err := adapter.Read(buf)
		
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf[:n])
	})

	t.Run("reads in multiple chunks", func(t *testing.T) {
		data := []byte("This is a longer string for chunked reading")
		reader := bytes.NewReader(data)
		mockedCloser := &mockReaderCloser{reader: reader}
		
		adapter := NewFileAdapter(mockedCloser, mockedCloser)
		
		buf := make([]byte, 10)
		var totalRead int
		
		for {
			n, err := adapter.Read(buf)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			totalRead += n
		}
		
		assert.Equal(t, len(data), totalRead)
	})

	t.Run("propagates reader errors", func(t *testing.T) {
		expectedErr := errors.New("reader error")
		mockedCloser := &mockReaderCloser{
			reader: &errorReader{err: expectedErr},
		}
		
		adapter := NewFileAdapter(mockedCloser, mockedCloser)
		
		buf := make([]byte, 10)
		_, err := adapter.Read(buf)
		
		assert.Equal(t, expectedErr, err)
	})

	t.Run("returns EOF when depleted", func(t *testing.T) {
		data := []byte("short")
		reader := bytes.NewReader(data)
		mockedCloser := &mockReaderCloser{reader: reader}
		
		adapter := NewFileAdapter(mockedCloser, mockedCloser)
		
		buf := make([]byte, 100)
		// Read until EOF
		var err error
		for err == nil {
			_, err = adapter.Read(buf)
		}
		
		require.Equal(t, io.EOF, err)
	})
}

// TestFileAdapter_Close verifies closing functionality
func TestFileAdapter_Close(t *testing.T) {
	t.Run("closes the closer when present", func(t *testing.T) {
		mockedCloser := &mockReaderCloser{
			reader: bytes.NewReader([]byte("data")),
			closed: false,
		}
		
		adapter := NewFileAdapter(bytes.NewReader([]byte("data")), mockedCloser)
		
		err := adapter.Close()
		
		assert.NoError(t, err)
		assert.True(t, mockedCloser.closed, "Closer should be called")
	})

	t.Run("propagates close errors", func(t *testing.T) {
		expectedErr := errors.New("close error")
		mockedCloser := &mockReaderCloser{
			closeErr: expectedErr,
		}
		
		adapter := NewFileAdapter(bytes.NewReader([]byte("data")), mockedCloser)
		
		err := adapter.Close()
		
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handles nil closer gracefully", func(t *testing.T) {
		adapter := NewFileAdapter(bytes.NewReader([]byte("data")), nil)
		
		err := adapter.Close()
		
		assert.NoError(t, err)
	})
}

// TestFileAdapter_SeparateReaderAndCloser verifies behavior with separate reader and closer
func TestFileAdapter_SeparateReaderAndCloser(t *testing.T) {
	t.Run("uses separate reader and closer", func(t *testing.T) {
		data := []byte("separate instances")
		reader := bytes.NewReader(data)
		mockedCloser := &mockReaderCloser{closed: false}
		
		adapter := NewFileAdapter(reader, mockedCloser)
		
		// Verify reading works
		buf := make([]byte, len(data))
		n, err := adapter.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf[:n])
		
		// Verify closing works
		assert.False(t, mockedCloser.closed)
		err = adapter.Close()
		assert.NoError(t, err)
		assert.True(t, mockedCloser.closed)
	})
}

// TestFileAdapter_MultipleOperations verifies multiple read and close operations
func TestFileAdapter_MultipleOperations(t *testing.T) {
	data := []byte("multiple operations test")
	reader := bytes.NewReader(data)
	mockedCloser := &mockReaderCloser{reader: reader}
	
	adapter := NewFileAdapter(reader, mockedCloser)
	
	// First read
	buf1 := make([]byte, 8)
	n1, err := adapter.Read(buf1)
	assert.NoError(t, err)
	assert.Equal(t, 8, n1)
	assert.Equal(t, []byte("multiple"), buf1)
	
	// Second read - continue from where we left off
	buf2 := make([]byte, 100)
	n2, err := adapter.Read(buf2)
	assert.NoError(t, err)
	// Read the rest including space and "operations test"
	assert.Equal(t, len(data)-8, n2)
	
	// Third read should return EOF
	_, err = adapter.Read(buf2)
	assert.Equal(t, io.EOF, err)
	
	// Close should work
	err = adapter.Close()
	assert.NoError(t, err)
	assert.True(t, mockedCloser.closed)
}

// Helper types for testing

type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

type dataReader []byte

func (dr dataReader) Read(p []byte) (n int, err error) {
	n = copy(p, dr)
	dr = dr[n:]
	if len(dr) == 0 {
		return n, io.EOF
	}
	return n, nil
}
