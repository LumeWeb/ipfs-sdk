// Package examples demonstrates how to use the IPFS SDK's upload functionality.
//
// The SDK provides two main upload approaches:
//
// 1. UploadFromFS - Generates CAR files from filesystems automatically
//    - Best for files, directories, and local content
//    - Handles CAR generation, TUS/POST selection, and retry logic
//    - Uses ipfs-content's high-level APIs (StreamCAR, PrepareCAR)
//
// 2. Upload - Uploads raw streams without CAR generation
//    - Best for existing data streams (HTTP responses, network data, etc.)
//    - Simpler when you don't need CAR format validation
//    - Direct stream-to-upload with minimal overhead
package examples

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.lumeweb.com/ipfs-sdk"
)

// UploadSimpleFileExample demonstrates uploading a single file using UploadFromFS.
// This method automatically generates a CAR file from the file and uploads it.
func UploadSimpleFileExample() error {
	// Create the SDK client
	authToken := os.Getenv("IPFS_AUTH_TOKEN")
	if authToken == "" {
		return fmt.Errorf("IPFS_AUTH_TOKEN environment variable not set")
	}
	client, err := ipfs.NewClient("https://api.example.com", authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Create a filesystem for the file you want to upload
	filePath := "path/to/your/file.txt"
	dirPath := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	
	// Create filesystem from the directory containing the file
	filesystem := os.DirFS(dirPath)
	
	// Create upload options
	opts := &ipfs.UploadOptions{
		MemoryLimit:  100 * 1024 * 1024, // 100MB memory limit
		WrapInDir:    false,              // Don't wrap single file
	}
	
	// Upload the file - UploadFromFS handles CAR generation and upload protocol selection
	ctx := context.Background()
	result, err := uploadService.UploadFromFS(ctx, filesystem, fileName, opts)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	
	fmt.Printf("Successfully uploaded file!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}

// UploadDirectoryExample demonstrates uploading an entire directory.
// The directory will be converted to a CAR file and uploaded as a single unit.
func UploadDirectoryExample() error {
	// Create the SDK client
	authToken := os.Getenv("IPFS_AUTH_TOKEN")
	if authToken == "" {
		return fmt.Errorf("IPFS_AUTH_TOKEN environment variable not set")
	}
	client, err := ipfs.NewClient("https://api.example.com", authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Directory to upload
	dirPath := "path/to/your/directory"
	
	// Create filesystem for the directory
	filesystem := os.DirFS(dirPath)
	
	// Create upload options - WrapInDir=true wraps all files in a root directory
	opts := &ipfs.UploadOptions{
		MemoryLimit: 100 * 1024 * 1024, // 100MB memory limit
		WrapInDir:   true,               // Wrap in root directory
	}
	
	// Upload the directory - UploadFromFS handles everything
	ctx := context.Background()
	result, err := uploadService.UploadFromFS(ctx, filesystem, "my-directory", opts)
	if err != nil {
		return fmt.Errorf("failed to upload directory: %w", err)
	}
	
	fmt.Printf("Successfully uploaded directory!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}

// UploadLargeDirectoryExample demonstrates uploading a large directory
// that requires the TUS resumable upload protocol.
func UploadLargeDirectoryExample() error {
	// Create the SDK client
	authToken := os.Getenv("IPFS_AUTH_TOKEN")
	if authToken == "" {
		return fmt.Errorf("IPFS_AUTH_TOKEN environment variable not set")
	}
	client, err := ipfs.NewClient("https://api.example.com", authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Directory to upload (large directory)
	dirPath := "path/to/large/directory"
	
	// Create filesystem for the directory
	filesystem := os.DirFS(dirPath)
	
	// Create upload options for large content
	// UploadLimit=10MB forces TUS for larger files for resumable uploads
	opts := &ipfs.UploadOptions{
		MemoryLimit: 200 * 1024 * 1024, // 200MB memory limit
		WrapInDir:   true,               // Wrap in root directory
		UploadLimit: 10 * 1024 * 1024,  // 10MB threshold for TUS
	}
	
	// Upload the directory - UploadFromFS handles CAR generation and protocol selection
	ctx := context.Background()
	result, err := uploadService.UploadFromFS(ctx, filesystem, "large-directory", opts)
	if err != nil {
		return fmt.Errorf("failed to upload large directory: %w", err)
	}
	
	fmt.Printf("Successfully uploaded large directory!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}

// UploadRawStreamExample demonstrates uploading any stream of data.
// This is useful when you have data from network, memory, or other sources
// and want to upload it directly without CAR generation or filesystem operations.
func UploadRawStreamExample() error {
	// Create the SDK client
	authToken := os.Getenv("IPFS_AUTH_TOKEN")
	if authToken == "" {
		return fmt.Errorf("IPFS_AUTH_TOKEN environment variable not set")
	}
	client, err := ipfs.NewClient("https://api.example.com", authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Create a reader with your data (could be from HTTP response, file, bytes, etc.)
	data := "Hello, IPFS! This is raw stream data."
	reader := strings.NewReader(data)
	
	// Upload the stream directly
	ctx := context.Background()
	size := int64(len(data))
	
	result, err := uploadService.Upload(ctx, reader, "example-stream.txt", size)
	if err != nil {
		return fmt.Errorf("failed to upload stream: %w", err)
	}
	
	fmt.Printf("Successfully uploaded stream!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}

// StreamToPipeExample demonstrates using StreamToPipe helper directly
// to convert blocking operations into non-blocking streams.
// This is useful when you have blocking write operations that should
// run in the background while you stream the data.
func StreamToPipeExample() error {
	// Example: Generate CAR in background while starting upload
	// This pattern mirrors what UploadFromFS does internally
	
	// Simulate some data that would be generated by a blocking operation
	data := []byte("Hello, IPFS! This is a blocking example.")
	
	// StreamToPipe runs the blocking write in a goroutine
	// The reader is returned immediately for consumption
	reader := ipfs.StreamToPipe(func(w io.Writer) error {
		// This simulates a blocking write operation
		// In real usage, this could be CAR generation, JSON encoding, etc.
		time.Sleep(100 * time.Millisecond) // Simulate blocking work
		_, err := w.Write(data)
		return err
	})
	
	// reader is immediately available for use
	// The write happens in a goroutine in the background
	
	fmt.Printf("StreamToPipe created - reader size: %d bytes\n", int64(len(data)))
	fmt.Println("Write operation running in background goroutine")
	
	// In real usage, you would consume the reader immediately:
	// result, err := uploadService.Upload(ctx, reader, "example.txt", size)
	
	_ = reader // Reader is closed automatically when goroutine completes
	
	return nil
}

// UploadFileExample demonstrates uploading a single file using the UploadFile method.
// This method provides a convenient way to upload open file handles without manually
// creating filesystem wrappers.
func UploadFileExample() error {
	// Create the SDK client
	authToken := os.Getenv("IPFS_AUTH_TOKEN")
	if authToken == "" {
		return fmt.Errorf("IPFS_AUTH_TOKEN environment variable not set")
	}
	client, err := ipfs.NewClient("https://api.example.com", authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Open the file
	filePath := "path/to/your/file.txt"
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Create upload options
	opts := &ipfs.UploadOptions{
		MemoryLimit: 100 * 1024 * 1024, // 100MB memory limit
	}
	
	// Upload the file directly - UploadFile handles filesystem wrapping and CAR generation
	ctx := context.Background()
	fileName := filepath.Base(filePath)
	result, err := uploadService.UploadFile(ctx, file, fileName, opts)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	
	fmt.Printf("Successfully uploaded file!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	fmt.Printf("  Original file size: %d bytes\n", fileInfo.Size())
	
	return nil
}

// RunCarExamples runs all CAR upload examples
func RunCarExamples() {
	examples := []struct {
		name string
		fn   func() error
	}{
		{"Upload Simple File", UploadSimpleFileExample},
		{"Upload Directory", UploadDirectoryExample},
		{"Upload Large Directory (TUS)", UploadLargeDirectoryExample},
		{"Upload Raw Stream", UploadRawStreamExample},
		{"StreamToPipe Helper", StreamToPipeExample},
		{"Upload File", UploadFileExample},
	}
	
	for _, example := range examples {
		fmt.Printf("\n=== %s ===\n", example.name)
		if err := example.fn(); err != nil {
			log.Printf("Error: %v\n", err)
		}
	}
}
