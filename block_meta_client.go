package ipfs

import (
	"context"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
)

// BlockMetaClient is a minimal interface for block meta operations.
type BlockMetaClient interface {
	GetApiBlockMetaCidWithResponse(ctx context.Context, cid string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiBlockMetaCidResponse, error)
}
