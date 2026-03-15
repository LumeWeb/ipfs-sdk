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
func (b *BytesFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &bytesDir{filename: b.filename, size: int64(len(b.data))}, nil
	}
	if name == b.filename {
		return &bytesFile{name: b.filename, data: b.data}, nil
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
	return &bytesFileInfo{size: int64(len(f.data)), isDir: false}, nil
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

// bytesDir implements fs.File and fs.ReadDirFile for a directory containing a single file.
type bytesDir struct {
	filename string
	size     int64
	offset   int
}

// Stat implements fs.File.Stat.
func (d *bytesDir) Stat() (fs.FileInfo, error) {
	return &bytesFileInfo{size: 0, isDir: true}, nil
}

// Read implements io.Reader (directories can't be read as data).
func (d *bytesDir) Read([]byte) (int, error) {
	return 0, io.EOF
}

// Close implements fs.File.Close (no-op).
func (d *bytesDir) Close() error {
	return nil
}

// Seek implements io.Seeker for bytesDir (no-op, directories cannot be seeked).
func (d *bytesDir) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("seek not supported on directories")
}

// ReadDir implements fs.ReadDirFile.ReadDir.
func (d *bytesDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.offset >= 1 {
		return nil, io.EOF
	}
	d.offset++
	return []fs.DirEntry{&bytesDirEntry{name: d.filename, size: d.size}}, nil
}

// bytesFileInfo implements fs.FileInfo.
type bytesFileInfo struct {
	size  int64
	isDir bool
}

func (fi *bytesFileInfo) Name() string       { return "" }
func (fi *bytesFileInfo) Size() int64        { return fi.size }
func (fi *bytesFileInfo) Mode() fs.FileMode  { return 0644 }
func (fi *bytesFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *bytesFileInfo) IsDir() bool        { return fi.isDir }
func (fi *bytesFileInfo) Sys() any   { return nil }

// bytesDirEntry implements fs.DirEntry for BytesFS.
type bytesDirEntry struct {
	name string
	size int64
}

func (de *bytesDirEntry) Name() string           { return de.name }
func (de *bytesDirEntry) Type() fs.FileMode      { return 0 }
func (de *bytesDirEntry) Info() (fs.FileInfo, error) {
	return &bytesFileInfo{size: de.size, isDir: false}, nil
}
func (de *bytesDirEntry) IsDir() bool {
	return false
}
