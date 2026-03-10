package io_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	io_internal "go.lumeweb.com/ipfs-sdk/pkg/upload/internal/io"
)

func TestReadSeekCloser(t *testing.T) {
	t.Run("reads correctly", func(t *testing.T) {
		data := []byte("hello world")
		rsc := io_internal.NewReadSeekCloser(bytes.NewReader(data))
		defer rsc.Close()

		buf := make([]byte, 5)
		n, err := rsc.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf)
	})

	t.Run("seeks when underlying reader supports it", func(t *testing.T) {
		data := []byte("hello world")
		rsc := io_internal.NewReadSeekCloser(bytes.NewReader(data))

		// Read first 5 bytes
		buf1 := make([]byte, 5)
		rsc.Read(buf1)

		// Seek to start
		pos, err := rsc.Seek(0, io.SeekStart)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), pos)

		// Read again
		buf2 := make([]byte, 5)
		n, err := rsc.Read(buf2)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf2)
	})

	t.Run("seek not supported when no io.Seeker", func(t *testing.T) {
		data := []byte("hello")
		reader := &mockReaderSeeker{reader: strings.NewReader(string(data))}
		rsc := io_internal.NewReadSeekCloser(reader)

		pos, err := rsc.Seek(5, io.SeekStart)

		assert.Error(t, err)
		assert.Equal(t, int64(0), pos)
		assert.Contains(t, err.Error(), "seek not supported")
	})

	t.Run("close when underlying reader supports it", func(t *testing.T) {
		closed := false
		closer := &mockCloser{closeFn: func() error { closed = true; return nil }}
		rsc := io_internal.NewReadSeekCloser(closer)

		err := rsc.Close()

		assert.NoError(t, err)
		assert.True(t, closed)
	})

	t.Run("close when no close method", func(t *testing.T) {
		rsc := io_internal.NewReadSeekCloser(strings.NewReader("test"))

		err := rsc.Close()

		assert.NoError(t, err)
	})
}

func TestToReadSeeker(t *testing.T) {
	t.Run("converts reader to read seeker", func(t *testing.T) {
		data := []byte("hello world")
		rs := io_internal.ToReadSeeker(bytes.NewReader(data))

		buf := make([]byte, 5)
		n, err := rs.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf)
	})

	t.Run("supports seeking", func(t *testing.T) {
		data := []byte("hello world")
		rs := io_internal.ToReadSeeker(&mockReaderAt{data: data})

		// Read first 5 bytes
		buf1 := make([]byte, 5)
		rs.Read(buf1)
		assert.Equal(t, []byte("hello"), buf1)

		// Seek to start
		pos, err := rs.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), pos)

		// Read from start again
		buf2 := make([]byte, 5)
		n, err := rs.Read(buf2)
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf2)
	})
}

func TestToByteReader(t *testing.T) {
	t.Run("returns reader if already byte reader", func(t *testing.T) {
		original := bytes.NewReader([]byte("test"))
		br := io_internal.ToByteReader(original)

		assert.Same(t, original, br)
	})

	t.Run("wraps regular reader with byte reader", func(t *testing.T) {
		reader := strings.NewReader("test")
		br := io_internal.ToByteReader(reader)

		b, err := br.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
	})
}

func TestToByteReadSeeker(t *testing.T) {
	t.Run("wraps read seeker with byte seeker", func(t *testing.T) {
		rs := bytes.NewReader([]byte("test"))
		brs := io_internal.ToByteReadSeeker(rs)

		b, err := brs.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
	})

	t.Run("wraps regular reader with discarding byte seeker", func(t *testing.T) {
		reader := strings.NewReader("test")
		rs := io_internal.ToByteReadSeeker(reader)

		b, err := rs.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)

		pos, err := rs.Seek(2, io.SeekCurrent)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), pos)
	})
}

func TestToReaderAt(t *testing.T) {
	t.Run("converts read seeker to reader at", func(t *testing.T) {
		data := []byte("hello world")
		rs := io_internal.ToReaderAt(bytes.NewReader(data))

		buf := make([]byte, 5)
		n, err := rs.ReadAt(buf, 6)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("world"), buf)
	})
}

// mockCloser is a mock io.ReadCloser for testing
type mockCloser struct {
	closeFn func() error
	data    []byte
	pos     int
	closeCalled bool
}

type mockReaderSeeker struct {
	reader io.Reader
}

func (m *mockReaderSeeker) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockCloser) Read(p []byte) (n int, err error) {
	if m.closeCalled {
		return 0, errors.New("reader closed")
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockCloser) Close() error {
	m.closeCalled = true
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

// mockReaderAt is a mock io.ReaderAt for testing
type mockReaderAt struct {
	data []byte
}

func (m *mockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	return n, nil
}
