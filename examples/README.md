# IPFS SDK Upload Examples

This directory contains comprehensive examples demonstrating how to use the IPFS SDK's upload functionality with CAR files and archives.

## Overview

The IPFS SDK provides two main approaches for uploading content:

1. **CAR Upload** - Upload files/directories by generating Content Addressable Archive (CAR) files
2. **Archive Upload** - Upload ZIP/TAR archives with automatic or raw processing

## Examples

### CAR Upload Examples (`upload_car_example.go`)

- `UploadSimpleFileExample()` - Upload a single file using CAR generation
- `UploadDirectoryExample()` - Upload an entire directory as CAR
- `UploadLargeDirectoryExample()` - Upload large directories using TUS protocol
- `UploadRawStreamExample()` - Direct stream upload for raw data without CAR generation

### Archive Upload Examples (`upload_archive_example.go`)

- `UploadZipAutoProcessExample()` - Upload ZIP with automatic server processing
- `UploadZipRawExample()` - Upload ZIP as raw file without processing
- `UploadLargeZipExample()` - Upload large ZIP files using TUS
- `UploadMultipleArchivesExample()` - Batch archive upload
- `UploadArchiveWithContextExample()` - Upload with context timeout

## Key Concepts

### CAR Upload

**CAR (Content Addressable Archive)** is the IPFS native format for storing content. The SDK uses `go.lumeweb.com/ipfs-content` for efficient CAR generation:

**High-Level API Pattern (Simplified):**
```go
// UploadFromFS handles everything automatically
result, err := uploadService.UploadFromFS(ctx, filesystem, "name", opts)
```

**Blocking Nuance:**

CAR generation with `builder.WriteCAR()` can be **blocking/expensive**. To prevent this from blocking upload operations, `UploadFromFS` uses the `StreamToPipe` helper:

```go
// Uses StreamToPipe helper (upload.go)
pr := StreamToPipe(func(w io.Writer) error {
    return builder.WriteCAR(ctx, w)
})
result, err := uploadService.Upload(ctx, pr, name, carSize)
```

The `StreamToPipe` helper creates a pipe and runs CAR generation in a goroutine, automatically handling error propagation and pipe closing. This prevents blocking while efficiently streaming data.

**Why this matters:**
- CAR generation needs to read entire filesystem
- Upload should not wait for CAR generation to complete
- Pipe allows both operations to proceed in parallel
- Goroutine ensures CAR generation doesn't block upload

**Bottom line:** `UploadFromFS` handles this internally. Unless you need custom CAR handling, use the high-level API.

**How it works internally:**
1. Walks your filesystem and builds a tree summary
2. Calculates the CAR file size using `StreamCARWithSize`
3. Generates the CAR file using memory-efficient LRU blockstore
4. Uses `StreamToPipe` helper to avoid blocking CAR generation
5. Chooses POST or TUS protocol based on file size
6. Uploads and returns result with CID

**When to use:**
- Uploading directories or multiple files as a unit
- Ensuring CID integrity before upload
- Uploading from local filesystem or memory filesystem
- Large directory trees (memory-efficient with LRU caching)

### Using StreamToPipe Helper

The `StreamToPipe` helper converts blocking write operations into non-blocking streams. It allows you to run expensive operations (like CAR generation) in the background while their output is consumed immediately.

**When to use:**
- Your write function blocks until completion
- You want to start consuming data immediately
- Parallelizing generation and consumption improves performance
- Building custom upload pipelines

**Example usage:**

```go
// Convert blocking write into non-blocking stream
reader := ipfs.StreamToPipe(func(w io.Writer) error {
    // Blocking operation here (CAR generation, JSON encoding, etc.)
    time.Sleep(100 * time.Millisecond) // Simulate work
    _, err := w.Write(data)
    return err
})

// reader is immediately available
// Write operation runs in background goroutine

// Use the reader immediately
result, err := uploadService.Upload(ctx, reader, "file.txt", size)
```

**Key benefits:**
- **Non-blocking:** Returns `io.ReadCloser` immediately
- **Parallel execution:** Generation and consumption run simultaneously
- **Error handling:** Errors are propagated via `CloseWithError`
- **Automatic cleanup:** Pipe is closed when goroutine completes

**See:** `upload_car_example.go` for complete example.

### Archive Upload

**Archive processing** controls how the server handles ZIP/TAR files:

**Auto Mode (ArchiveModeAuto - Default)**
- Server unpacks the archive
- Converts contents to CAR format
- Returns CID of unpacked content
- Best for content distribution

**Raw Mode (ArchiveModeRaw)**  
- Server stores file as-is
- No unpacking or processing
- Returns CID of archive file itself
- Best for archive preservation

When uploading via `Upload()`, the server respects the `archive` query parameter:
- `archive=true` - Auto mode (unpack and convert)
- `archive=false` - Raw mode (store as-is)

**When to use:**
- Uploading ZIP/TAR archives for content distribution (Auto)
- Uploading archives for file storage/sharing (Raw)
- When you don't care about CAR format (use archives)

### Upload Protocols

The SDK automatically chooses the appropriate protocol:

**POST (HTTP)**
- Files ≤ upload limit (default: 100MB)
- Simpler, faster for small files
- CID returned is correct for CAR uploads
- For raw files with Auto mode, server processes in-memory

**TUS (Resumable)**
- Files > upload limit
- Supports resumable uploads
- Better for large files/slow connections
- Can pause and resume uploads
- Server may process archives in background

### Archive Processing Behavior

| Mode | CAR Upload | POST - Auto | POST - Raw | TUS - Auto | TUS - Raw |
|------|-----------|-------------|------------|------------|-----------|
| Behavior | Ignored (bypassed) | In-memory proc. | No proc. | Background proc. | No proc. |
| CID | Known from CAR | Correct | Correct | May vary* | May vary* |

*For TUS with raw files, the CID returned may be incorrect during or immediately after upload due to asynchronous background processing.

## Usage Guidelines

### Choosing the Right Upload Method

```go
// Use UploadFromFS when:
// - Uploading directories or multiple files
// - Need CAR format with guaranteed CID
// - Want memory-efficient handling of large trees
result, err := uploadService.UploadFromFS(ctx, filesystem, "name", opts)

// Use Upload when:
// - Uploading a single stream or file
// - Don't want CAR generation
// - Simpler API is preferred
result, err := uploadService.Upload(ctx, reader, "name", size)
```

### Configuring Upload Behavior

```go
opts := &ipfs.UploadOptions{
    // Memory limit for CAR generation (bytes)
    MemoryLimit: 100 * 1024 * 1024, // 100MB
    
    // Wrap single file in directory
    WrapInDir: true, // for directories, false for single file
    
    // Threshold for TUS upload (bytes)
    UploadLimit: 10 * 1024 * 1024, // 10MB
    
    // Archive processing mode
    ArchiveConfig: ipfs.AutoArchive(), // or ipfs.RawArchive()
}
```

### Memory Limits

For large directories, choose memory limits based on your content:

```go
// Small directories (<1GB)
MemoryLimit: 100 * 1024 * 1024 // 100MB

// Medium directories (1-10GB)
MemoryLimit: 200 * 1024 * 1024 // 200MB

// Large directories (10-100GB)
MemoryLimit: 500 * 1024 * 1024 // 500MB
```

The SDK uses an LRU blockstore, so blocks exceeding the limit are regenerated on-demand.

## Running Examples

### Import as Package

Import the examples package and call the example functions directly:

```go
import "go.lumeweb.com/ipfs-sdk/examples"

// Run CAR upload examples
examples.RunCarExamples()

// Run archive upload examples
examples.RunExamples()
```

### Run Specific Example Function

You can create your own main to call specific examples:

```go
package main

import (
    "log"
    "go.lumeweb.com/ipfs-sdk/examples"
)

func main() {
    // Run single file upload example
    if err := examples.UploadSimpleFileExample(); err != nil {
        log.Fatal(err)
    }
}
```

### Build Package

```bash
go build ./...
```

## Integration Examples

### Uploading from memory filesystem

```go
import "github.com/psanford/memfs"

// Create in-memory filesystem
fsys := memfs.New()

// Add files
fsys.WriteFile("readme.md", []byte("hello"), 0644)
fsys.WriteFile("data.json", []byte(`{"val":1}`), 0644)

// Upload
result, err := uploadService.UploadFromFS(ctx, fsys, "my-data", opts)
```

### Updating existing uploads

```go
// Get upload status
upload, err := uploadService.GetUploadStatus(ctx, location)

// Resume interrupted upload
result, err := uploadService.ResumeUpload(ctx, location, reader)

// Cancel upload
err = uploadService.CancelUpload(ctx, location)
```

### Upload with retry

```go
import "github.com/avast/retry-go/v4"

err := retry.Do(
    func() error {
        _, err := uploadService.Upload(ctx, reader, "file", size)
        return err
    },
    retry.Attempts(3),
    retry.Context(ctx),
)
```

## Best Practices

1. **Use CAR format** for content distribution and CID guarantees
2. **Set appropriate memory limits** based on directory size
3. **Use Auto archive mode** when you want server processing
4. **Use Raw archive mode** to preserve archive files
5. **Use timeouts** to prevent hanging uploads
6. **Check file size** before upload to choose protocol
7. **Handle context cancellation** gracefully
8. **Close file handles** after upload
9. **Handle resumable uploads** for large files (>10MB)
10. **Validate content** before upload when possible

## Error Handling

```go
result, err := uploadService.UploadFromFS(ctx, filesystem, "name", opts)
if err != nil {
    // Check for context cancellation
    if errors.Is(err, context.Canceled) {
        // Upload was cancelled
    }
    
    // Handle specific errors
    var memErr interface{ Unwrap() error }
    if errors.As(err, &memErr) && strings.Contains(err.Error(), "memory limit") {
        // Reduce memory limit and retry
    }
    
    return err
}
```

## Performance Tips

1. **Adjust memory limit** based on content size (see Memory Limits section)
2. **Use POST** for small files (<100MB default)
3. **Use TUS** for large files with resume support
4. **Parallel uploads** for multiple small files
5. **Stream from disk** instead of memory for large files
6. **Validate file size** before upload to avoid surprise TUS usage

## Monitoring Uploads

```go
// Track upload progress (for TUS with custom client)
// See tusgo documentation for progress tracking

// Upload state management
type UploadState struct {
    CID       string
    Size      int64
    Location  string
    Status    string // "uploading", "completed", "error"
    Error     error
}

// Query upload status by location
upload, err := uploadService.GetUploadStatus(ctx, location)
fmt.Printf("Offset: %d, Size: %d\n", upload.RemoteOffset, upload.Size)
```

## Related Documentation

- [API Documentation](../doc.go) - Main SDK documentation
- [Upload API Documentation](#) - Detailed upload API docs
- [IPFS CAR Specification](https://github.com/ipfs/car) - CAR format details
- [TUS Protocol](https://tus.io/) - Resumable upload protocol

## Support

For issues or questions:
- Check existing [GitHub Issues](https://github.com/your-org/ipfs-sdk/issues)
- Review code comments and examples
- Consult API documentation
