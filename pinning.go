package ipfs

import (
	"context"
	"time"

	boxo "github.com/ipfs/boxo/pinning/remote/client"
	"github.com/ipfs/go-cid"
)

// PinningService wraps boxo's existing pinning client for consistency
type PinningService struct {
	client *boxo.Client
}

// NewPinningService creates a wrapper around boxo's existing client
func NewPinningService(baseURL, bearerToken string) *PinningService {
	return &PinningService{
		client: boxo.NewClient(baseURL, bearerToken),
	}
}

// ListPins lists all pins with optional filters
// Uses boxo's LsSync method for a synchronous list operation
func (s *PinningService) ListPins(ctx context.Context, opts ...LsOption) ([]boxo.PinStatusGetter, error) {
	var boxoOpts []boxo.LsOption
	for _, opt := range opts {
		boxoOpts = append(boxoOpts, boxo.LsOption(opt))
	}
	return s.client.LsSync(ctx, boxoOpts...)
}

// AddPin adds a pin with optional configuration
// Delegates to boxo's Add method
func (s *PinningService) AddPin(ctx context.Context, cid cid.Cid, opts ...AddOption) (boxo.PinStatusGetter, error) {
	var boxoOpts []boxo.AddOption
	for _, opt := range opts {
		boxoOpts = append(boxoOpts, boxo.AddOption(opt))
	}
	return s.client.Add(ctx, cid, boxoOpts...)
}

// GetPin gets a specific pin by request ID
// Delegates to boxo's GetStatusByID method
func (s *PinningService) GetPin(ctx context.Context, requestID string) (boxo.PinStatusGetter, error) {
	return s.client.GetStatusByID(ctx, requestID)
}

// RemovePin removes a pin by request ID
// Delegates to boxo's DeleteByID method
func (s *PinningService) RemovePin(ctx context.Context, requestID string) error {
	return s.client.DeleteByID(ctx, requestID)
}

// ReplacePin replaces an existing pin
// Delegates to boxo's Replace method
func (s *PinningService) ReplacePin(ctx context.Context, requestID string, cid cid.Cid, opts ...AddOption) (boxo.PinStatusGetter, error) {
	var boxoOpts []boxo.AddOption
	for _, opt := range opts {
		boxoOpts = append(boxoOpts, boxo.AddOption(opt))
	}
	return s.client.Replace(ctx, requestID, cid, boxoOpts...)
}

// LsOption configures list operations
// Type alias for boxo.LsOption to expose in this package
type LsOption boxo.LsOption

// AddOption configures add operations
// Type alias for boxo.AddOption to expose in this package
type AddOption boxo.AddOption

// PinStatusGetter represents a pin status
// Type alias for boxo.PinStatusGetter to expose in this package
type PinStatusGetter = boxo.PinStatusGetter

// Status represents pin status states
// Type alias to expose from boxo
type Status = boxo.Status

// PinStatusGetter exported helper methods for boxo types
// Methods delegated to boxo implementation

// Status constants exported from boxo
const (
	StatusUnknown = boxo.StatusUnknown
	StatusQueued  = boxo.StatusQueued
	StatusPinning = boxo.StatusPinning
	StatusPinned  = boxo.StatusPinned
	StatusFailed  = boxo.StatusFailed
)

// Ls exported functional options
// Pass-through to boxo's PinOpts methods

// FilterCIDs filters by CIDs
func FilterCIDs(cids ...cid.Cid) LsOption {
	return LsOption(boxo.PinOpts.FilterCIDs(cids...))
}

// FilterName filters by name
func FilterName(name string) LsOption {
	return LsOption(boxo.PinOpts.FilterName(name))
}

// FilterStatus filters by status
func FilterStatus(statuses ...Status) LsOption {
	return LsOption(boxo.PinOpts.FilterStatus(statuses...))
}

// FilterBefore filters by timestamp
func FilterBefore(t time.Time) LsOption {
	return LsOption(boxo.PinOpts.FilterBefore(t))
}

// FilterAfter filters by timestamp
func FilterAfter(t time.Time) LsOption {
	return LsOption(boxo.PinOpts.FilterAfter(t))
}

// Limit sets max results
func Limit(limit int) LsOption {
	return LsOption(boxo.PinOpts.Limit(limit))
}

// LsMeta sets metadata for list operations
func LsMeta(meta map[string]string) LsOption {
	return LsOption(boxo.PinOpts.LsMeta(meta))
}

// Add exported functional options for pin operations
// Pass-through to boxo's PinOpts methods

// WithName sets the name for a pin
func WithName(name string) AddOption {
	return AddOption(boxo.PinOpts.WithName(name))
}

// WithMeta sets metadata for a pin
func WithMeta(meta map[string]string) AddOption {
	return AddOption(boxo.PinOpts.AddMeta(meta))
}


