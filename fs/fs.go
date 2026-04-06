// Package fs provides filesystem adapters and interfaces for the IPFS SDK.
//
// This package contains:
// - Filesystem adapters (BytesFS, SingleFileFS) for upload operations
// - Reusable mock interfaces for testing
//
package fs

import (
	"github.com/ipfs/boxo/files"
)

// Test interfaces for mocking boxo files types in tests.
// These embed the upstream boxo interfaces so mockery can generate mocks.

// File is a test interface that embeds boxo's files.File.
// Use this with mockery to generate mocks in tests.
type File interface {
	files.File
}

// Node is a test interface that embeds boxo's files.Node.
// Use this with mockery to generate mocks in tests.
type Node interface {
	files.Node
}

// FileInfo is a test interface that embeds boxo's files.FileInfo.
// Use this with mockery to generate mocks in tests.
type FileInfo interface {
	files.FileInfo
}

// DirEntry is a test interface that embeds boxo's files.DirEntry.
// Use this with mockery to generate mocks in tests.
type DirEntry interface {
	files.DirEntry
}

// DirIterator is a test interface that embeds boxo's files.DirIterator.
// Use this with mockery to generate mocks in tests.
type DirIterator interface {
	files.DirIterator
}

// Directory is a test interface that embeds boxo's files.Directory.
// Use this with mockery to generate mocks in tests.
type Directory interface {
	files.Directory
}
