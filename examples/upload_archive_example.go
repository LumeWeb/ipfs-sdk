// Package examples demonstrates archive upload functionality.
// This example shows how to upload ZIP and other archive files with different
// processing modes.
package examples

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.lumeweb.com/ipfs-sdk"
)

// UploadZipAutoProcessExample demonstrates uploading a ZIP file with automatic
// processing on the server. The server will unpack the archive and convert it
// to CAR format.
func UploadZipAutoProcessExample() error {
	// Create the SDK client
	client, err := ipfs.NewClient("https://api.example.com", "your-auth-token")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Open the ZIP file
	zipPath := "path/to/your/archive.zip"
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Upload with automatic archive processing
	ctx := context.Background()
	result, err := uploadService.Upload(ctx, file, "archive.zip", fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to upload ZIP: %w", err)
	}
	
	fmt.Printf("Successfully uploaded ZIP with auto processing!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	fmt.Printf("  Server will unpack and convert to CAR format\n")
	
	return nil
}

// UploadZipRawExample demonstrates uploading a ZIP file as a raw file
// without processing. The file will be stored as-is without unpacking.
func UploadZipRawExample() error {
	// Create the SDK client
	client, err := ipfs.NewClient("https://api.example.com", "your-auth-token")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service with custom configuration
	uploadService := client.Upload()
	
	// Open the ZIP file
	zipPath := "path/to/your/archive.zip"
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// For POST uploads, we need to handle archive config directly
	// This is handled automatically by the Upload method with the server
	// The server respects the archive query parameter
	
	ctx := context.Background()
	result, err := uploadService.Upload(ctx, file, "archive.zip", fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to upload ZIP: %w", err)
	}
	
	fmt.Printf("Successfully uploaded ZIP as raw file!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	fmt.Printf("  File stored as-is without processing\n")
	
	return nil
}

// UploadLargeZipExample demonstrates uploading a large ZIP file using TUS
// resumable upload protocol.
func UploadLargeZipExample() error {
	// Create the SDK client
	client, err := ipfs.NewClient("https://api.example.com", "your-auth-token")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Open the large ZIP file
	zipPath := "path/to/large/archive.zip"
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	fmt.Printf("Large ZIP file size: %d bytes (%.2f MB)\n", 
		fileInfo.Size(), 
		float64(fileInfo.Size())/(1024*1024))
	
	// Upload using TUS resumable protocol
	ctx := context.Background()
	result, err := uploadService.Upload(ctx, file, "large-archive.zip", fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to upload large ZIP: %w", err)
	}
	
	fmt.Printf("Successfully uploaded large ZIP!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}

// UploadMultipleArchivesExample demonstrates uploading multiple archive files
// in sequence.
func UploadMultipleArchivesExample() error {
	// Create the SDK client
	client, err := ipfs.NewClient("https://api.example.com", "your-auth-token")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// List of archives to upload
	archives := []string{
		"project1.zip",
		"project2.zip",
		"project3.zip",
	}
	
	ctx := context.Background()
	
	for _, archivePath := range archives {
		fmt.Printf("Uploading %s...\n", archivePath)
		
		file, err := os.Open(archivePath)
		if err != nil {
			log.Printf("Failed to open %s: %v\n", archivePath, err)
			continue
		}
		
		fileInfo, err := file.Stat()
		if err != nil {
			file.Close()
			log.Printf("Failed to get info for %s: %v\n", archivePath, err)
			continue
		}
		
		result, err := uploadService.Upload(ctx, file, archivePath, fileInfo.Size())
		file.Close()
		
		if err != nil {
			log.Printf("Failed to upload %s: %v\n", archivePath, err)
			continue
		}
		
		fmt.Printf("  Success! CID: %s, Size: %d bytes\n", result.CID, result.Size)
	}
	
	return nil
}

// UploadArchiveWithContextExample demonstrates uploading with context
// timeouts and cancellation.
func UploadArchiveWithContextExample() error {
	// Create the SDK client
	client, err := ipfs.NewClient("https://api.example.com", "your-auth-token")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get the upload service
	uploadService := client.Upload()
	
	// Open the archive file
	zipPath := "path/to/your/archive.zip"
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer file.Close()
	
	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*60) // 5 minute timeout
	defer cancel()
	
	// Upload with timeout
	result, err := uploadService.Upload(ctx, file, "archive.zip", fileInfo.Size())
	if err != nil {
		return fmt.Errorf("upload failed or timed out: %w", err)
	}
	
	fmt.Printf("Successfully uploaded with timeout!\n")
	fmt.Printf("  CID: %s\n", result.CID)
	fmt.Printf("  Size: %d bytes\n", result.Size)
	
	return nil
}



// RunExamples demonstrates different archive upload scenarios
func RunExamples() {
	examples := []struct {
		name string
		fn   func() error
	}{
			{"Upload ZIP with Auto Processing", UploadZipAutoProcessExample},
		{"Upload ZIP as Raw File", UploadZipRawExample},
		{"Upload Large ZIP (TUS)", UploadLargeZipExample},
		{"Upload Multiple Archives", UploadMultipleArchivesExample},
		{"Upload with Context Timeout", UploadArchiveWithContextExample},
	}
	
	for _, example := range examples {
		fmt.Printf("\n=== %s ===\n", example.name)
		if err := example.fn(); err != nil {
			log.Printf("Error: %v\n", err)
		}
	}
}
