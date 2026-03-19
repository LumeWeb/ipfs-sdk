package fs

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

// BytesFS implements fs.FS to wrap a []byte as a single-file filesystem.
// This is useful for uploading byte slices as CAR files.
type BytesFS struct {
	data     []byte
	filename string
}

// NewBytesFS creates a new filesystem containing a single file with the given data.
func NewBytesFS(data []byte, filename string) *BytesFS {
	return &BytesFS{data: data, filename: filename}
}

// Open implements fs.FS.Open.
// Only the root "." and the single filename are valid paths.
// For "." we return the file itself since BytesFS represents a single file,
// not a directory containing the file.
func (b *BytesFS) Open(name string) (fs.File, error) {
	if name == "." || name == b.filename {
		return &bytesFile{name: b.filename, data: b.data}, nil
	}
	return nil, fs.ErrNotExist
}

// Stat implements fs.StatFS.Stat.
// Returns file info for "." or the single filename.
func (b *BytesFS) Stat(name string) (fs.FileInfo, error) {
	if name == "." || name == b.filename {
		return &bytesFileInfo{name: b.filename, size: int64(len(b.data)), isDir: false}, nil
	}
	return nil, fs.ErrNotExist
}

// bytesFile implements fs.File for a single byte slice.
type bytesFile struct {
	name string
	data []byte
	pos  int64
}

// Stat implements fs.File.Stat.
func (f *bytesFile) Stat() (fs.FileInfo, error) {
	return &bytesFileInfo{name: f.name, size: int64(len(f.data)), isDir: false}, nil
}

// Read implements io.Reader.
func (f *bytesFile) Read(p []byte) (int, error) {
	if f.pos >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += int64(n)
	return n, nil
}

// Close implements fs.File.Close (no-op).
func (f *bytesFile) Close() error {
	return nil
}

// Seek implements io.Seeker for repositioning within the file.
// Supports SeekStart, SeekCurrent, and SeekEnd.
func (f *bytesFile) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.pos + offset
	case io.SeekEnd:
		newOffset = int64(len(f.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence parameter")
	}

	if newOffset < 0 {
		newOffset = 0
	}

	f.pos = newOffset
	return newOffset, nil
}

// bytesFileInfo implements fs.FileInfo.
type bytesFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi *bytesFileInfo) Name() string       { return fi.name }
func (fi *bytesFileInfo) Size() int64        { return fi.size }
func (fi *bytesFileInfo) Mode() fs.FileMode  { return 0644 }
func (fi *bytesFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *bytesFileInfo) IsDir() bool        { return fi.isDir }
func (fi *bytesFileInfo) Sys() any   { return nil }
