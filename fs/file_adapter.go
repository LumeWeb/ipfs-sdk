package fs

import (
	"bytes"
	"io"
)

// FileAdapter wraps any reader/closer combination to implement both io.Reader and io.Closer.
// This is useful for adapting different file types to a common interface, particularly
// when you have a reader that needs to be released when done.
//
// Example usage with a files.File from boxo:
//
//	fileNode, err := backend.GetBlock(ctx, path)
//	if err != nil {
//	    return nil, err
//	}
//
//	// Wrap the files.File as an io.ReadCloser
//	return &FileAdapter{Closer: fileNode, Reader: fileNode}, nil
type FileAdapter struct {
	Closer io.Closer
	Reader io.Reader
}

// Read implements io.Reader.
func (fa *FileAdapter) Read(p []byte) (n int, err error) {
	return fa.Reader.Read(p)
}

// Close implements io.Closer.
func (fa *FileAdapter) Close() error {
	if fa.Closer != nil {
		return fa.Closer.Close()
	}
	return nil
}

// NewFileAdapter creates a new FileAdapter from any object that implements both
// io.Reader and io.Closer, or separate reader and closer instances.
func NewFileAdapter(reader io.Reader, closer io.Closer) *FileAdapter {
	return &FileAdapter{
		Reader: reader,
		Closer: closer,
	}
}

// ReadSeekCloserAdapter wraps a *bytes.Reader to implement io.ReadSeekCloser.
// Useful when you need to combine Seek functionality with a Closer (even a no-op).
//
// Example usage with ipfs-content:
//
//	data := []byte("hello")
//	seekCloser := NewReadSeekCloserAdapter(data)
//	_, err := seekCloser.Seek(0, io.SeekStart) // Reset to beginning
type ReadSeekCloserAdapter struct {
	*bytes.Reader
}

// Close implements io.Closer as a no-op for bytes.Reader.
func (r *ReadSeekCloserAdapter) Close() error {
	return nil
}

// NewReadSeekCloserAdapter creates a new ReadSeekCloserAdapter from byte data.
// The returned type implements io.Reader, io.Seeker, and io.Closer.
func NewReadSeekCloserAdapter(data []byte) *ReadSeekCloserAdapter {
	return &ReadSeekCloserAdapter{
		Reader: bytes.NewReader(data),
	}
}
