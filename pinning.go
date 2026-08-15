package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
	ippinning "go.lumeweb.com/ipfs-sdk/internal/pinning"
)

// Pin types from internal/pinning package to avoid import cycles
type Pin = ippinning.Pin
type PinStatus = ippinning.PinStatus
type PinMeta = ippinning.PinMeta
type PinResults = ippinning.PinResults
type StatusInfo = ippinning.StatusInfo
type Delegates = ippinning.Delegates
type Origins = ippinning.Origins

// PinStatusEnum represents the status of a pin
// Type alias to expose from generated client
type PinStatusEnum = ippinning.PinStatusEnum

const (
	// StatusQueued - pinning operation is waiting in the queue
	StatusQueued PinStatusEnum = ippinning.Queued
	// StatusPinning - pinning in progress
	StatusPinning PinStatusEnum = ippinning.Pinning
	// StatusPinned - pinned successfully
	StatusPinned PinStatusEnum = ippinning.Pinned
	// StatusFailed - pinning service was unable to finish pinning operation
	StatusFailed PinStatusEnum = ippinning.Failed
)

// PinningService provides pinning management functionality
type PinningService interface {
	// List all pins with optional filters
	ListPins(ctx context.Context, opts ...ListOption) ([]PinStatus, error)
	// Add a new pin
	AddPin(ctx context.Context, cid cid.Cid, opts ...AddOption) (*PinStatus, error)
	// Get a pin by request ID
	GetPin(ctx context.Context, requestID string) (*PinStatus, error)
	// Remove a pin by request ID
	RemovePin(ctx context.Context, requestID string) error
	// SetAuthToken hot-updates the bearer token used for authenticated
	// requests. The pinning service is long-lived, so the token must be
	// swappable at runtime (e.g. after a `pinner login` rewrites the config)
	// without recreating the client.
	SetAuthToken(token string)
}

// pinningService implements PinningService using the generated client
type pinningService struct {
	client  ippinning.ClientWithResponsesInterface
	retry   httputil.RetryConfig
	httpCli *http.Client

	// mu guards authToken so a concurrent SetAuthToken (from the live-reload
	// path) can never race a request editor read of the token.
	mu        sync.RWMutex
	authToken string
}

// NewPinningService creates a pinning service using the generated client
// This replaces the boxo wrapper and allows full control over the HTTP client
func NewPinningService(baseURL, bearerToken string, opts ...PinningServiceOption) PinningService {
	cfg := DefaultPinningServiceConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	svc := &pinningService{
		retry:     cfg.Retry,
		httpCli:   httpClient,
		authToken: bearerToken,
	}

	// Create request editor for authentication. The token is read from the
	// service's mutable field so SetAuthToken can hot-swap it for subsequent
	// requests without recreating the generated client.
	requestEditor := func(ctx context.Context, req *http.Request) error {
		svc.mu.RLock()
		token := svc.authToken
		svc.mu.RUnlock()
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	}

	// Create generated client - baseURL is passed as the second argument
	client, err := ippinning.NewClientWithResponses(baseURL,
		ippinning.WithRequestEditorFn(requestEditor),
		ippinning.WithHTTPClient(httpClient))
	if err != nil {
		panic(fmt.Sprintf("failed to create pinning client: %v", err))
	}
	svc.client = client

	return svc
}

// SetAuthToken hot-updates the bearer token used for pinning requests.
func (s *pinningService) SetAuthToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authToken = token
}

// PinningServiceOption applies configuration to PinningService
type PinningServiceOption func(*PinningServiceConfig)

// PinningServiceConfig holds configuration for pinning service
type PinningServiceConfig struct {
	Retry      httputil.RetryConfig
	HTTPClient *http.Client
}

// DefaultPinningServiceConfig returns default configuration
func DefaultPinningServiceConfig() PinningServiceConfig {
	return PinningServiceConfig{
		Retry: httputil.DefaultRetryConfig(),
	}
}

// WithPinningHTTPClient sets a custom HTTP client for pinning service
// This is used for host override support
func WithPinningHTTPClient(client *http.Client) PinningServiceOption {
	return func(cfg *PinningServiceConfig) {
		cfg.HTTPClient = client
	}
}

// WithPinningRetry sets the retry configuration for pinning operations
func WithPinningRetry(cfgRetry httputil.RetryConfig) PinningServiceOption {
	return func(cfg *PinningServiceConfig) {
		cfg.Retry = cfgRetry
	}
}

// ListOption configures list operations
// Type alias for query parameters
type ListOption ippinning.GetPinsParams

// List functional options
// WithFilterCIDs filters by CIDs
func WithFilterCIDs(cids ...string) ListOption {
	cidsCopy := ippinning.Cid(cids)
	return ListOption{Cid: &cidsCopy}
}

// WithFilterName filters by name
func WithFilterName(name string) ListOption {
	nameCopy := ippinning.Name(name)
	return ListOption{Name: &nameCopy}
}

// WithFilterMatch sets the text matching strategy applied together with
// WithFilterName (exact, iexact, partial, ipartial). Partial/ipartial perform
// substring matching anywhere in the name, per the IPFS Pinning Services API
// spec's TextMatchingStrategy. Empty zero-value disables match overrides.
func WithFilterMatch(strategy ippinning.TextMatchingStrategy) ListOption {
	matchCopy := ippinning.Match(strategy)
	return ListOption{Match: &matchCopy}
}

// WithFilterStatus filters by status
func WithFilterStatus(statuses ...PinStatusEnum) ListOption {
	statusesCopy := ippinning.Status(statuses)
	return ListOption{Status: &statusesCopy}
}

// WithListMeta sets metadata filters for list operations
func WithListMeta(meta ippinning.PinMeta) ListOption {
	return ListOption{Meta: &meta}
}

// WithLimit sets max results
func WithLimit(limit int32) ListOption {
	return ListOption{Limit: &limit}
}

// WithBefore filters by timestamp
func WithBefore(t time.Time) ListOption {
	return ListOption{Before: &t}
}

// WithAfter filters by timestamp
func WithAfter(t time.Time) ListOption {
	return ListOption{After: &t}
}

// AddOption configures add operations
type AddOption struct {
	Name    *string
	Meta    *PinMeta
	Origins *Origins
}

// WithAddName sets the name for a pin
func WithAddName(name string) AddOption {
	return AddOption{Name: &name}
}

// WithAddMeta sets metadata for a pin
func WithAddMeta(meta ippinning.PinMeta) AddOption {
	return AddOption{Meta: &meta}
}

// WithAddOrigins sets origins for a pin
func WithAddOrigins(origins []string) AddOption {
	return AddOption{Origins: &origins}
}

// ListPins retrieves all pins with optional filters
func (s *pinningService) ListPins(ctx context.Context, opts ...ListOption) ([]PinStatus, error) {
	var result []PinStatus

	err := httputil.RetryContext(ctx, s.retry, func() error {
		params := &ippinning.GetPinsParams{}
		for _, opt := range opts {
			if opt.Cid != nil {
				params.Cid = opt.Cid
			}
			if opt.Name != nil {
				params.Name = opt.Name
			}
			if opt.Match != nil {
				params.Match = opt.Match
			}
			if opt.Status != nil {
				params.Status = opt.Status
			}
			if opt.Meta != nil {
				params.Meta = opt.Meta
			}
			if opt.Limit != nil {
				params.Limit = opt.Limit
			}
			if opt.Before != nil {
				params.Before = opt.Before
			}
			if opt.After != nil {
				params.After = opt.After
			}
		}

		resp, err := s.client.GetPinsWithResponse(ctx, params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListPins, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil || resp.JSON200.Results == nil {
			result = []PinStatus{}
			return nil
		}

		result = resp.JSON200.Results
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AddPin adds a new pin
func (s *pinningService) AddPin(ctx context.Context, c cid.Cid, opts ...AddOption) (*PinStatus, error) {
	var result *PinStatus

	err := httputil.RetryContext(ctx, s.retry, func() error {
		pin := ippinning.Pin{
			Cid: c.String(),
		}

		for _, opt := range opts {
			if opt.Name != nil {
				pin.Name = opt.Name
			}
			if opt.Meta != nil {
				pin.Meta = opt.Meta
			}
			if opt.Origins != nil {
				pin.Origins = opt.Origins
			}
		}

		resp, err := s.client.AddPinWithResponse(ctx, pin)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpAddPin, []int{http.StatusOK, http.StatusAccepted}); err != nil {
			return err
		}

		if resp.JSON202 != nil {
			result = resp.JSON202
		} else {
			return ErrBadRequest("add pin: no response data")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetPin retrieves a pin by request ID
func (s *pinningService) GetPin(ctx context.Context, requestID string) (*PinStatus, error) {
	var result *PinStatus

	err := httputil.RetryContext(ctx, s.retry, func() error {
		resp, err := s.client.GetPinByRequestIdWithResponse(ctx, requestID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetPin, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			result = nil
			return ErrBadRequest("get pin: no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RemovePin removes a pin by request ID
func (s *pinningService) RemovePin(ctx context.Context, requestID string) error {
	return httputil.RetryContext(ctx, s.retry, func() error {
		resp, err := s.client.DeletePinByRequestIdWithResponse(ctx, requestID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpRemovePin, []int{http.StatusAccepted, http.StatusNotFound}); err != nil {
			return err
		}

		return nil
	})
}
