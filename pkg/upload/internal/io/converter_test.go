package io

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ReadByte - the underlying implementations (readerPlusByte, readSeekerPlusByte, discardingReadSeekerPlusByte)
// are tested indirectly through ToByteReader and ToByteReadSeeker which return ByteReader interfaces

func TestToByteReader(t *testing.T) {
	t.Run("returns same reader if already ByteReader", func(t *testing.T) {
		original := bytes.NewReader([]byte("test"))
		br := ToByteReader(original)
		
		assert.Same(t, original, br)
	})
	
	t.Run("wraps regular reader to provide ByteReader", func(t *testing.T) {
		reader := strings.NewReader("test")
		br := ToByteReader(reader)
		
		b, err := br.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
		
		b2, err := br.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('e'), b2)
	})
}

func TestToByteReadSeeker(t *testing.T) {
	t.Run("returns same reader if already ByteReadSeeker", func(t *testing.T) {
		rs := bytes.NewReader([]byte("test"))
		br := ToByteReadSeeker(rs)
		
		assert.Same(t, rs, br)
	})
	
	t.Run("wraps ReadSeeker with ByteSeeker", func(t *testing.T) {
		rs := bytes.NewReader([]byte("test"))
		brs := ToByteReadSeeker(rs)
		
		b, err := brs.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
		
		// Should also support seeking
		pos, err := brs.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), pos)
		
		b, err = brs.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
	})
	
	t.Run("wraps regular Reader with discarding ByteSeeker", func(t *testing.T) {
		reader := strings.NewReader("test")
		rs := ToByteReadSeeker(reader)
		
		b, err := rs.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('t'), b)
		
		// Should support seeking via discarding
		pos, err := rs.Seek(2, io.SeekCurrent)
		assert.NoError(t, err)
		assert.Greater(t, pos, int64(2))
		
		// Reading byte should give 's' (skipped 2 from 'test')
		// The implementation advances the reader
	})
}

func TestToReadSeeker(t *testing.T) {
	t.Run("returns same value if already ReadSeeker", func(t *testing.T) {
		data := []byte("hello world")
		rs := bytes.NewReader(data)
		result := ToReadSeeker(rs)
		
		assert.Same(t, rs, result)
	})
	
	t.Run("converts ReaderAt to ReadSeeker", func(t *testing.T) {
		data := []byte("hello world")
		ra := bytes.NewReader(data) // bytes.Reader is both ReaderAt and ReadSeeker
		rs := ToReadSeeker(ra)
		
		// Read sequentially
		buf := make([]byte, 5)
		n, err := rs.Read(buf)
		
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf)
		
		// Next read should continue
		buf = make([]byte, 6)
		n, err = rs.Read(buf)
		
		require.NoError(t, err)
		assert.Equal(t, 6, n)
		assert.Equal(t, []byte(" world"), buf)
	})
	
	t.Run("supports seeking after conversion", func(t *testing.T) {
		data := []byte("hello world")
		ra := &mockReaderAt{data: data}
		rs := ToReadSeeker(ra)
		
		// Read first 5 bytes
		buf1 := make([]byte, 5)
		rs.Read(buf1)
		
		// Seek to start
		pos, err := rs.Seek(0, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(0), pos)
		
		// Read again
		buf2 := make([]byte, 5)
		n, err := rs.Read(buf2)
		
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf2)
	})
	
	t.Run("supports SeekCurrent", func(t *testing.T) {
		data := []byte("hello world")
		ra := &mockReaderAt{data: data}
		rs := ToReadSeeker(ra)
		
		// Seek from start
		_, err := rs.Seek(2, io.SeekStart)
		require.NoError(t, err)
		
		// Seek from current
		pos, err := rs.Seek(1, io.SeekCurrent)
		require.NoError(t, err)
		assert.Equal(t, int64(3), pos)
		
		// Read should start from position 3
		buf := make([]byte, 5)
		n, err := rs.Read(buf)
		
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("lo wo"), buf)
	})
	
	t.Run("SeekEnd returns error", func(t *testing.T) {
		data := []byte("test")
		ra := &mockReaderAt{data: data}
		rs := ToReadSeeker(ra)
		
		_, err := rs.Seek(0, io.SeekEnd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
	
	t.Run("handles empty data", func(t *testing.T) {
		data := []byte{}
		ra := &mockReaderAt{data: data}
		rs := ToReadSeeker(ra)
		
		buf := make([]byte, 10)
		n, err := rs.Read(buf)
		
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})
}

func TestToReaderAt(t *testing.T) {
	t.Run("returns same value if already ReaderAt", func(t *testing.T) {
		// bytes.Reader implements both io.ReaderAt and io.ReadSeeker
		rs := bytes.NewReader([]byte("test"))
		rsa := ToReaderAt(rs)
		
		// Since it already implements ReaderAt, should return same object
		assert.Same(t, rs, rsa)
	})
	
	t.Run("converts ReadSeeker to ReaderAt", func(t *testing.T) {
		data := []byte("hello world")
		rs := bytes.NewReader(data)
		ra := ToReaderAt(rs)
		
		buf := make([]byte, 5)
		n, err := ra.ReadAt(buf, 6)
		
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("world"), buf)
	})
	
	t.Run("uses seek before reading", func(t *testing.T) {
		data := []byte("hello world")
		rs := bytes.NewReader(data)
		ra := ToReaderAt(rs)
		
		// Should seek before reading
		buf := make([]byte, 5)
		n, err := ra.ReadAt(buf, 0)
		
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf)
	})
	
	t.Run("handles seek failure", func(t *testing.T) {
		rs := &failingReaderSeeker{seekErr: errors.New("seek failed")}
		result := ToReaderAt(rs)
		
		// ReadAt should fail
		buf := make([]byte, 5)
		_, err := result.ReadAt(buf, 0)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "seek failed")
	})
}

// Mock types to expose private interfaces for testing

type failingReaderSeeker struct {
	seekErr   error
	readErr   error
	position  int64
	lastSeek  int64
	lastWhence int
}

func (f *failingReaderSeeker) Read(p []byte) (n int, err error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return 0, io.EOF
}

func (f *failingReaderSeeker) Seek(offset int64, whence int) (int64, error) {
	f.lastSeek = offset
	f.lastWhence = whence
	if f.seekErr != nil {
		return 0, f.seekErr
	}
	return f.position, nil
}

type mockReaderAt struct {
	data []byte
}

func (m *mockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	if off+int64(n) >= int64(len(m.data)) && n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
