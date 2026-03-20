package download

import "github.com/ipfs/boxo/gateway"

// Backend extends the upstream gateway.IPFSBackend interface for testing.
// This interface can be used with mockery to generate mocks that implement
// all the gateway backend methods.
type Backend interface {
	gateway.IPFSBackend
}
