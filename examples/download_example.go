// Package examples demonstrates download functionality for the IPFS SDK.
package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/ipfs/go-cid"
	"go.lumeweb.com/ipfs-sdk"
)

// DownloadBlockExample demonstrates how to download IPFS blocks using the DownloadService.
// This example shows checking block existence, downloading blocks, and getting block metadata.
func DownloadBlockExample(baseURL, token string) {
	// Create a new IPFS SDK client
	client, err := ipfs.NewClient(baseURL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example: Download a block by CID
	ctx := context.Background()
	cidStr := "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6orfzdoy-addon2yykg3q"

	c, err := cid.Decode(cidStr)
	if err != nil {
		log.Fatalf("Failed to decode CID: %v", err)
	}

	// Get the download service
	dl := client.Download()

	// Check if block exists
	exists, err := dl.Has(ctx, c)
	if err != nil {
		log.Fatalf("Failed to check block existence: %v", err)
	}
	fmt.Printf("Block exists: %v\n", exists)

	// Get the block
	block, err := dl.Block(ctx, c)
	if err != nil {
		log.Fatalf("Failed to download block: %v", err)
	}

	fmt.Printf("Downloaded block with CID: %s\n", block.Cid().String())
	fmt.Printf("Block size: %d bytes\n", len(block.RawData()))

	// Get block size directly
	size, err := dl.BlockSize(ctx, c)
	if err != nil {
		log.Fatalf("Failed to get block size: %v", err)
	}
	fmt.Printf("Block size from BlockSize(): %d bytes\n", size)

	// Get raw data
	raw, err := dl.Raw(ctx, c)
	if err != nil {
		log.Fatalf("Failed to get raw data: %v", err)
	}
	fmt.Printf("Raw data length: %d bytes\n", len(raw))
}

// DownloadFileFromDirectoryExample demonstrates how to download a full file from a directory.
func DownloadFileFromDirectoryExample(baseURL, token string) {
	// Create a new IPFS SDK client
	client, err := ipfs.NewClient(baseURL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Example: Download a complete file from a directory CID
	// The service automatically detects UnixFS format and handles directories
	dl := client.Download()

	// CID of a file in IPFS
	c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
	if err != nil {
		log.Fatalf("Failed to decode CID: %v", err)
	}

	// Download the full file
	reader, err := dl.DownloadFile(ctx, c)
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	defer reader.Close()

	// Read the file contents
	data := make([]byte, 1024)
	n, err := reader.Read(data)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	fmt.Printf("Downloaded %d bytes from file\n", n)
}

// ListDirectoryExample demonstrates how to list directory entries.
func ListDirectoryExample(baseURL, token string) {
	// Create a new IPFS SDK client
	client, err := ipfs.NewClient(baseURL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	dl := client.Download()

	// CID of a directory in IPFS
	c, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
	if err != nil {
		log.Fatalf("Failed to decode CID: %v", err)
	}

	// List directory entries
	entries, err := dl.ListDirectory(ctx, c)
	if err != nil {
		log.Fatalf("Failed to list directory: %v", err)
	}

	// Iterate through directory entries
	for _, entry := range entries {
		name := entry.Name()
		node := entry.Node()
		
		fmt.Printf("Entry: %s\n", name)
		fmt.Printf("  IsDirectory: %v\n", node.Mode().IsDir())
		
		size, err := node.Size()
		if err == nil {
			fmt.Printf("  Size: %d bytes\n", size)
		}
	}
}

// GetFileFromPathExample demonstrates how to access a file by path within a directory.
func GetFileFromPathExample(baseURL, token string) {
	// Create a new IPFS SDK client
	client, err := ipfs.NewClient(baseURL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	dl := client.Download()

	// CID of a directory in IPFS
	dirCID, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
	if err != nil {
		log.Fatalf("Failed to decode CID: %v", err)
	}

	// Get a specific file by path (e.g., "examples/myfile.txt")
	reader, err := dl.GetFile(ctx, dirCID, "examples/myfile.txt")
	if err != nil {
		log.Fatalf("Failed to get file: %v", err)
	}
	defer reader.Close()

	// Read the file contents
	data := make([]byte, 1024)
	n, err := reader.Read(data)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	fmt.Printf("Read %d bytes from path 'examples/myfile.txt'\n", n)
}
