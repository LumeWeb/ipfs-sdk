package ipfs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ippinning "go.lumeweb.com/ipfs-sdk/internal/pinning"

	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

const (
	testCID1 = "bafkreifzjut3te2nhyekklss27nh3k72ysco7y32koao5eei66wof36n5e"
	testCID2 = "bafkreibihfrdjdys4gip2ymnh5wcgulockswet3amzlwqy3ezbzsf6rhte"
)

func TestWithFilterCIDs(t *testing.T) {
	t.Run("creates filter option with single CID", func(t *testing.T) {
		opt := WithFilterCIDs(testCID1)

		assert.NotNil(t, opt)
	})

	t.Run("creates filter option with multiple CIDs", func(t *testing.T) {
		opt := WithFilterCIDs(testCID1, testCID2)

		assert.NotNil(t, opt)
	})
}

func TestWithFilterName(t *testing.T) {
	t.Run("creates name filter option", func(t *testing.T) {
		name := "test-pin-name"
		opt := WithFilterName(name)

		assert.NotNil(t, opt)
	})
}

func TestWithFilterMatch(t *testing.T) {
	t.Run("sets the text matching strategy", func(t *testing.T) {
		opt := WithFilterMatch(ippinning.Partial)
		require.NotNil(t, opt.Match)
		assert.Equal(t, ippinning.TextMatchingStrategy("partial"), *opt.Match)
	})

	t.Run("supports all spec strategies", func(t *testing.T) {
		want := []ippinning.TextMatchingStrategy{
			ippinning.Exact, ippinning.Iexact, ippinning.Partial, ippinning.Ipartial,
		}
		for _, s := range want {
			opt := WithFilterMatch(s)
			require.NotNil(t, opt.Match, "WithFilterMatch(%q)", s)
			assert.Equal(t, s, *opt.Match)
		}
	})
}

// TestWithFilterNamePartial guards the convenience helper that composes a name
// filter with match=partial in a single ListOption, so callers can express a
// server-side substring name search without naming the strategy type.
func TestWithFilterNamePartial(t *testing.T) {
	opt := WithFilterNamePartial("docs")
	require.NotNil(t, opt.Name)
	assert.Equal(t, ippinning.Name("docs"), *opt.Name)
	require.NotNil(t, opt.Match)
	assert.Equal(t, ippinning.TextMatchingStrategy("partial"), *opt.Match)
}

// TestReExportedMatchStrategies guards the publicly re-exported strategy consts.
func TestReExportedMatchStrategies(t *testing.T) {
	assert.Equal(t, ippinning.Exact, TextMatchingStrategy(MatchExact))
	assert.Equal(t, ippinning.Partial, TextMatchingStrategy(MatchPartial))
	assert.Equal(t, ippinning.Iexact, TextMatchingStrategy(MatchIExact))
	assert.Equal(t, ippinning.Ipartial, TextMatchingStrategy(MatchIPartial))
}

// TestListPinsSendsMatchParam is the end-to-end guard for server-side substring
// pin search: when WithFilterName is combined with WithFilterMatch(Partial),
// ListPins must emit the spec's name and match=partial query params (the IPFS
// Pinning Services API TextMatchingStrategy), not drop the match on the floor.
func TestListPinsSendsMatchParam(t *testing.T) {
	var (
		mu    sync.Mutex
		query = url.Values{}
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		query = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	s := NewPinningService(srv.URL, "token-a")
	require.NotNil(t, s)

	_, err := s.ListPins(context.Background(),
		WithFilterName("docs"),
		WithFilterMatch(ippinning.Partial),
		WithPinningLimit(10),
	)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "docs", query.Get("name"), "name filter must be sent server-side")
	assert.Equal(t, "partial", query.Get("match"), "match=partial must be sent server-side for substring search")
}

func TestWithFilterStatus(t *testing.T) {
	t.Run("creates status filter with single status", func(t *testing.T) {
		opt := WithFilterStatus(StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("creates status filter with multiple statuses", func(t *testing.T) {
		opt := WithFilterStatus(StatusQueued, StatusPinning, StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("supports all status constants", func(t *testing.T) {
		tests := []PinStatusEnum{
			StatusQueued,
			StatusPinning,
			StatusPinned,
			StatusFailed,
		}

		for _, status := range tests {
			opt := WithFilterStatus(status)
			assert.NotNil(t, opt, "WithFilterStatus should not return nil for status %v", status)
		}
	})
}

func TestWithListMeta(t *testing.T) {
	t.Run("creates metadata option", func(t *testing.T) {
		meta := PinMeta{
			"app":     "test",
			"version": "1.0.0",
			"author":  "user123",
		}
		opt := WithListMeta(meta)

		assert.NotNil(t, opt)
	})

	t.Run("creates empty metadata option", func(t *testing.T) {
		opt := WithListMeta(PinMeta{})

		assert.NotNil(t, opt)
	})
}

func TestNewPinningService(t *testing.T) {
	t.Run("creates service with URL and token", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", os.Getenv("TEST_PINNING_TOKEN"))

		assert.NotNil(t, s)
	})

	t.Run("creates service with custom HTTP client", func(t *testing.T) {
		httpClient := &http.Client{}
		s := NewPinningService("http://localhost:5001", os.Getenv("TEST_PINNING_TOKEN"), WithPinningHTTPClient(httpClient))

		assert.NotNil(t, s)
	})
}

// TestPinningService_SetAuthToken verifies a pinning service's bearer token can
// be hot-swapped at runtime without recreating the client. A long-lived server
// (e.g. an MCP instance) must be able to push a fresh JWT into the existing
// pinning service after a `pinner login` rewrites the config; otherwise the
// service keeps sending the stale startup token.
func TestPinningService_SetAuthToken(t *testing.T) {
	var mu sync.Mutex
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	s := NewPinningService(srv.URL, "token-a")
	require.NotNil(t, s)

	// Build a list-pins request and assert it used token-a.
	_, err := s.ListPins(context.Background(), WithPinningLimit(1))
	require.NoError(t, err)
	mu.Lock()
	first := gotAuth
	mu.Unlock()
	assert.Equal(t, "Bearer token-a", first, "initial request should use the startup token")

	// Hot-swap the token, then verify subsequent requests send the new one.
	s.SetAuthToken("token-b")
	_, err = s.ListPins(context.Background())
	require.NoError(t, err)
	mu.Lock()
	second := gotAuth
	mu.Unlock()
	assert.Equal(t, "Bearer token-b", second, "request after SetAuthToken must use the new token")
}

func TestWithPinningLimit(t *testing.T) {
	t.Run("creates limit option", func(t *testing.T) {
		limit := int32(10)
		opt := WithPinningLimit(limit)

		assert.NotNil(t, opt)
	})
}

func TestWithBefore(t *testing.T) {
	t.Run("creates before timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := WithBefore(testTime)

		assert.NotNil(t, opt)
	})
}

func TestWithAfter(t *testing.T) {
	t.Run("creates after timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := WithAfter(testTime)

		assert.NotNil(t, opt)
	})
}

func TestStatusConstants(t *testing.T) {
	t.Run("all status constants have correct values", func(t *testing.T) {
		assert.Equal(t, PinStatusEnum("queued"), StatusQueued)
		assert.Equal(t, PinStatusEnum("pinning"), StatusPinning)
		assert.Equal(t, PinStatusEnum("pinned"), StatusPinned)
		assert.Equal(t, PinStatusEnum("failed"), StatusFailed)
	})
}

func TestPinningServiceOptions(t *testing.T) {
	t.Run("with custom HTTP client", func(t *testing.T) {
		cfg := DefaultPinningServiceConfig()
		httpClient := &http.Client{}

		opt := WithPinningHTTPClient(httpClient)
		opt(&cfg)

		assert.Equal(t, httpClient, cfg.HTTPClient)
	})

	t.Run("with retry config", func(t *testing.T) {
		cfg := DefaultPinningServiceConfig()
		retryCfg := httputil.RetryConfig{Attempts: 5}

		opt := WithPinningRetry(retryCfg)
		opt(&cfg)

		assert.Equal(t, uint(5), cfg.Retry.Attempts)
	})
}
