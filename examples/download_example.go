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
