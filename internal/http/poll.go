// Package http provides shared HTTP utilities for IPFS SDK clients.
package http

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultPollInterval is the default interval between polling attempts.
const DefaultPollInterval = 2 * time.Second

// DefaultPollTimeout is the default timeout for polling operations.
const DefaultPollTimeout = 5 * time.Minute

// PollConfig configures polling behavior for WaitForOperations.
type PollConfig struct {
	interval      time.Duration // Time between polls
	timeout       time.Duration // Maximum time to poll
	initialDelay  time.Duration // Optional delay before first poll
}

// PollOption is a functional option for poll configuration.
type PollOption func(*PollConfig)

// WithPollInterval sets the time between polling attempts.
func WithPollInterval(d time.Duration) PollOption {
	return func(c *PollConfig) {
		c.interval = d
	}
}

// WithPollTimeout sets the maximum time to poll before timing out.
func WithPollTimeout(d time.Duration) PollOption {
	return func(c *PollConfig) {
		c.timeout = d
	}
}

// WithInitialDelay sets a delay before the first poll.
func WithInitialDelay(d time.Duration) PollOption {
	return func(c *PollConfig) {
		c.initialDelay = d
	}
}

// DefaultPollConfig returns default polling configuration.
func DefaultPollConfig() *PollConfig {
	return &PollConfig{
		interval:     DefaultPollInterval,
		timeout:      DefaultPollTimeout,
		initialDelay: 0,
	}
}

// PollResult contains the result of a polling operation.
type PollResult struct {
	Value       interface{} // The value when isSettled returns true
	Attempts    int         // Number of polling attempts
	ElapsedTime time.Duration // Total time elapsed
}

// ExtractPollResult safely extracts the value from PollResult or returns an error.
// This helper eliminates boilerplate code for checking nil values after polling.
func ExtractPollResult(result *PollResult, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	if result == nil || result.Value == nil {
		return nil, fmt.Errorf("polling returned no value")
	}
	return result.Value, nil
}

// PollUntil polls a function until it returns true or the context times out.
//
// The isSettled callback receives a context and should return:
// - (true, value, nil) when the condition is met and polling should stop
// - (false, nil, nil) when polling should continue
// - (false, nil, error) when an unrecoverable error occurred
//
// Example:
//
//	result, err := PollUntil(ctx, cfg, func(ctx context.Context) (bool, interface{}, error) {
//	    item, err := getThing(ctx)
//	    if err != nil {
//	        return false, nil, err
//	    }
//	    settled := item.Status == "complete"
//	    return settled, item, nil
//	})
func PollUntil(ctx context.Context, cfg *PollConfig, isSettled func(context.Context) (bool, interface{}, error)) (*PollResult, error) {
	startTime := time.Now()
	attempts := 0

	// Create a context with timeout if not already present
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	// Apply initial delay if configured
	if cfg.initialDelay > 0 {
		select {
		case <-time.After(cfg.initialDelay):
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("poll timed out during initial delay")
		}
	}

	// Start ticker for polling attempts
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	// Execute first poll immediately, then poll on ticker ticks
	attempts++

	settled, value, err := isSettled(ctxWithTimeout)
	if err != nil {
		// Check if this is a context error (timeout or cancellation)
		// If so, let it propagate without wrapping to preserve error unwrapping
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("poll attempt %d failed: %w", attempts, err)
	}
	if settled {
		elapsed := time.Since(startTime)
		return &PollResult{
			Value:       value,
			Attempts:    attempts,
			ElapsedTime: elapsed,
		}, nil
	}

	for {
		select {
		case <-ctxWithTimeout.Done():
			// Return actual context error to preserve error unwrapping
			return nil, ctxWithTimeout.Err()

		case <-ticker.C:
			attempts++

			settled, value, err := isSettled(ctxWithTimeout)
			if err != nil {
				// Check if this is a context error (timeout or cancellation)
				// If so, let it propagate without wrapping to preserve error unwrapping
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					return nil, err
				}
				return nil, fmt.Errorf("poll attempt %d failed: %w", attempts, err)
			}

			if settled {
				elapsed := time.Since(startTime)
				return &PollResult{
					Value:       value,
					Attempts:    attempts,
					ElapsedTime: elapsed,
				}, nil
			}
		}
	}
}

// WaitForPolledState polls a function until it returns a value in the settled states.
// This is a convenience wrapper around PollUntil that checks if a value is in a set.
func WaitForPolledState(ctx context.Context, getCurrent func() (string, error), settledStates []string, opts ...PollOption) (*PollResult, error) {
	cfg := DefaultPollConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Convert states to a map for O(1) lookup
	statesMap := make(map[string]bool)
	for _, state := range settledStates {
		statesMap[state] = true
	}

	result, err := PollUntil(ctx, cfg, func(ctx context.Context) (bool, interface{}, error) {
		currentState, err := getCurrent()
		if err != nil {
			return false, nil, err
		}

		if statesMap[currentState] {
			return true, currentState, nil
		}
		return false, nil, nil
	})

	return result, err
}
