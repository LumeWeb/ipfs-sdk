// Package fs provides filesystem adapters for the IPFS SDK.
//
// This package contains adapters that convert Go data structures into the fs.FS
// interface, enabling them to be used with the IPFS SDK's CAR (Content Addressable
// Archive) generation facilities. The primary use case is uploading in-memory data
// (e.g., byte slices) to IPFS via the UploadService.
//
// # Primary Types
//
// The main type in this package is BytesFS, which wraps a []byte as a single-file
// filesystem. This allows byte slices to be uploaded via UploadBytes() with automatic
// CAR wrapping without needing to write temporary files.
//
// Usage Example
//
//	// Upload byte data to IPFS with CAR generation
//	service := ipfs.NewUploadService("https://api.example.com", "token")
//	ctx := context.Background()
//
//	data := []byte("Hello, World!")
//	result, err := service.UploadBytes(ctx, data, "hello.txt", nil)
//	// result.CID contains the IPFS content identifier
//
// The BytesFS internally implements the following standard Go interfaces:
//   - fs.FS for filesystem operations
//   - fs.File for file access
//   - fs.FileInfo for metadata
//   - fs.DirEntry for directory listings
//
// # Testing Considerations
//
// For testing purposes, Go's standard library provides excellent in-memory filesystem
// utilities that should be used instead of custom implementations:
//
//   - testing/fstest.MapFS - In-memory filesystem for testing
//     testFS := fstest.MapFS{"test.txt": {Data: []byte("test data")}}
//
// # When to Use BytesFS
//
// Use BytesFS when you need to upload byte slices via UploadBytes() or UploadFromFS()
// with automatic CAR generation. For other cases where an in-memory filesystem is needed
// (e.g., testing), use the standard library's testing/fstest.MapFS.
//
// # Thread Safety
//
// BytesFS implementations are not thread-safe. Create a new instance for each
// goroutine if concurrent access is required.
package fs
