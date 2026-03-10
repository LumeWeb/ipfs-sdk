// Package ipfs provides a Go SDK for interacting with IPFS HTTP gateway services.
//
// This SDK offers a comprehensive client for common IPFS operations including:
//   - Pinning: Pin and unpin content using IPFS Pinning Service API
//   - DNS: Manage DNSLink records for IPNS
//   - IPNS: Interact with InterPlanetary Naming System
//   - Websites: Deploy and manage website content
//   - Upload: Upload content via TUS resumable upload protocol
//
// Getting Started
//
// Create a new client with the gateway URL and authentication token:
//
//	client := NewClient("https://api.example.com", "your-token-here")
//
// Services
//
// The client provides access to specialized services:
//
//	client := NewClient(baseURL, token)
//	pinningClient := client.Pinning()
//	dnsClient := client.DNS()
//	ipnsClient := client.Ipns()
//	websitesClient := client.Websites()
//	uploadClient := client.Upload()
//
// Configuration
//
// The client supports configurable retry behavior:
//
//	cfg := DefaultClientConfig()
//	cfg.Retry.MaxAttempts = 5
//	cfg.Retry.BaseDelay = 100 * time.Millisecond
//	client := NewClient(baseURL, token, WithRetryConfig(cfg))
//
// Error Handling
//
// The SDK returns standard Go errors. Check for context cancellation when using
// operations with context:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	pins, err := client.Pinning().ListPins(ctx)
//	if err != nil {
//	    // handle error
//	}
package ipfs
