// Package testing provides internal interfaces for mocking upstream dependencies
// in unit tests. These interfaces embed upstream interfaces to allow mockery to generate
// mocks without dealing with import cycles.
package testing

import (
	blockstore "github.com/ipfs/boxo/blockstore"
)

// Blockstore embeds the github.com/ipfs/boxo/blockstore.Blockstore interface
// to enable mockery to generate mock implementations without import cycles.
type Blockstore interface {
	blockstore.Blockstore
}
