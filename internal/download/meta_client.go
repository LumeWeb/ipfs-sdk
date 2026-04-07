package download

import (
	"context"

	"github.com/ipfs/go-cid"
)

// BlockMetaClient queries block metadata without rate limiting.
type BlockMetaClient interface {
	GetBlockSize(context.Context, cid.Cid) (int, error)
}
