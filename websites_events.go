package ipfs

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	sse "go.lumeweb.com/ipfs-sdk/internal/sse"
	"go.uber.org/zap"
)

// Gateway-facing SSE event streaming for the internal websites API. The client
// connects to GET /internal/websites/events, streams website lifecycle events
// (published, removed) in real time and durably (via Last-Event-ID replay), and
// can be paired with WebsitesService.ReconcileWebsiteChanges to close any gap
// after a reconnect.

// WebsiteEventType identifies a streamed website lifecycle event.
type WebsiteEventType string

const (
	// WebsiteEventPublished is streamed when a website is published or updated.
	WebsiteEventPublished WebsiteEventType = "site_published"
	// WebsiteEventRemoved is streamed when a website is deleted.
	WebsiteEventRemoved WebsiteEventType = "site_removed"
)

// WebsitePublishedEvent is the payload of a published event.
type WebsitePublishedEvent struct {
	Domain      string    `json:"domain"`
	CID         string    `json:"cid"`
	PublishedAt time.Time `json:"published_at"`
}

// WebsiteRemovedEvent is the payload of a removed event.
type WebsiteRemovedEvent struct {
	Domain    string    `json:"domain"`
	RemovedAt time.Time `json:"removed_at"`
}

// WebsiteEvent is a parsed lifecycle event ready for handler dispatch.
type WebsiteEvent struct {
	Type      WebsiteEventType
	Published *WebsitePublishedEvent
	Removed   *WebsiteRemovedEvent
}

// sseData is the envelope the portal serializes into each SSE frame's data
// field: the event type plus the typed payload.
type sseData struct {
	Type WebsiteEventType `json:"type"`
	Data json.RawMessage  `json:"data"`
}

// WebsiteEventHandler processes a parsed website lifecycle event.
type WebsiteEventHandler func(WebsiteEvent)

// WebsiteEventsStats receives telemetry from the website event SSE client.
// Downstream consumers (e.g. the gateway) wire these callbacks to an
// observability backend such as Prometheus. All fields are optional; a nil
// callback is a no-op.
type WebsiteEventsStats struct {
	// EventReceived is called for every dispatched website lifecycle event.
	EventReceived func()
	// ParseError is called when an SSE frame or payload cannot be parsed.
	ParseError func()
	// ConnectionError is called when a connection or reconnection fails.
	ConnectionError func(err error)
	// Connected is called with the live connection state on state transitions,
	// and on a fixed interval when a stats poll interval is configured.
	Connected func(connected bool)
}

// WebsiteEventsConfig configures the website SSE event client.
type WebsiteEventsConfig struct {
	Reconnect         bool
	Backoff           time.Duration
	MaxBackoff        time.Duration
	MaxRetries        int
	Logger            *zap.Logger
	Stats             WebsiteEventsStats
	StatsPollInterval time.Duration
}

// WebsiteEventsOption applies configuration to WebsiteEventsConfig.
type WebsiteEventsOption func(*WebsiteEventsConfig)

// DefaultWebsiteEventsConfig returns sensible defaults for the SSE client.
func DefaultWebsiteEventsConfig() WebsiteEventsConfig {
	return WebsiteEventsConfig{
		Reconnect:  true,
		Backoff:    1 * time.Second,
		MaxBackoff: 30 * time.Second,
		MaxRetries: 10,
	}
}

// WithWebsiteEventsReconnect enables or disables automatic reconnection.
func WithWebsiteEventsReconnect(reconnect bool) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) { c.Reconnect = reconnect }
}

// WithWebsiteEventsBackoff sets the initial and maximum reconnection delays.
func WithWebsiteEventsBackoff(backoff, maxBackoff time.Duration) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) {
		c.Backoff = backoff
		c.MaxBackoff = maxBackoff
	}
}

// WithWebsiteEventsMaxRetries caps reconnection attempts (0 = unlimited).
func WithWebsiteEventsMaxRetries(retries int) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) { c.MaxRetries = retries }
}

// WithWebsiteEventsLogger sets the diagnostics logger (zap).
func WithWebsiteEventsLogger(logger *zap.Logger) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) { c.Logger = logger }
}

// WithWebsiteEventsStats wires optional observability callbacks (e.g. to a
// Prometheus implementation in the consuming service).
func WithWebsiteEventsStats(stats WebsiteEventsStats) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) { c.Stats = stats }
}

// WithWebsiteEventsStatsPollInterval enables a periodic connected-state poller
// (e.g. every 5s) feeding the Connected stats callback. Values > 0 enable the
// poller; 0 disables it.
func WithWebsiteEventsStatsPollInterval(interval time.Duration) WebsiteEventsOption {
	return func(c *WebsiteEventsConfig) { c.StatsPollInterval = interval }
}

// WebsiteEventsClient streams website lifecycle events from the portal's
// internal SSE endpoint. The X-Gateway-Secret header (required by the portal's
// gateway auth middleware) is set from the gateway secret; the client tracks
// the last durable event ID so a reconnect is replayed seamlessly via
// Last-Event-ID.
type WebsiteEventsClient struct {
	client    *sse.Client
	handler   WebsiteEventHandler
	handlerMu sync.RWMutex
	startOnce sync.Once
}

// NewWebsiteEventsClient creates an SSE event client for the given portal base
// URL and gateway secret. Call OnEvent to register a handler and Start to
// connect.
func NewWebsiteEventsClient(baseURL, gatewaySecret string, opts ...WebsiteEventsOption) (*WebsiteEventsClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("website events client requires a base URL")
	}

	cfg := DefaultWebsiteEventsConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	url := strings.TrimSuffix(baseURL, "/") + "/internal/websites/events"
	client := sse.NewClient(url, sse.Options{
		Reconnect:             cfg.Reconnect,
		Backoff:               cfg.Backoff,
		MaxBackoff:            cfg.MaxBackoff,
		MaxRetries:            cfg.MaxRetries,
		ConnectedPollInterval: cfg.StatsPollInterval,
	})
	if gatewaySecret != "" {
		client.SetHeader("X-Gateway-Secret", gatewaySecret)
	}
	if cfg.Logger != nil {
		client.SetLogger(cfg.Logger)
	}
	client.SetStats(sse.Stats{
		EventReceived:   cfg.Stats.EventReceived,
		ParseError:      cfg.Stats.ParseError,
		ConnectionError: cfg.Stats.ConnectionError,
		Connected:       cfg.Stats.Connected,
	})

	ec := &WebsiteEventsClient{client: client}
	client.OnEvent(string(WebsiteEventPublished), ec.route)
	client.OnEvent(string(WebsiteEventRemoved), ec.route)

	return ec, nil
}

// OnEvent registers the handler invoked for every parsed lifecycle event. Only
// one handler is supported; later calls replace the previous one. It is safe to
// call while the client is streaming.
func (c *WebsiteEventsClient) OnEvent(handler WebsiteEventHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	c.handler = handler
}

// OnError registers a handler invoked when reconnection is permanently given up.
func (c *WebsiteEventsClient) OnError(handler func(error)) {
	c.client.OnError(handler)
}

// Start connects to the SSE endpoint in a background goroutine, reconnecting
// with the configured backoff. Must be called only once; subsequent calls are
// no-ops.
func (c *WebsiteEventsClient) Start() {
	c.startOnce.Do(func() {
		go c.client.Start()
	})
}

// Stop disconnects from the SSE endpoint and waits for in-flight handlers.
func (c *WebsiteEventsClient) Stop() {
	c.client.Disconnect()
	c.client.Wait()
}

// SetLastEventID restores the durable cursor to resume replay from. Call before
// Start to pick up events missed while disconnected.
func (c *WebsiteEventsClient) SetLastEventID(id string) {
	c.client.SetLastEventID(id)
}

// LastEventID returns the most recently received durable event ID.
func (c *WebsiteEventsClient) LastEventID() string {
	return c.client.LastEventID()
}

// IsConnected reports whether an SSE stream is currently open.
func (c *WebsiteEventsClient) IsConnected() bool {
	return c.client.IsConnected()
}

// route parses a raw SSE frame into a WebsiteEvent and dispatches it.
func (c *WebsiteEventsClient) route(ev sse.Event) {
	var envelope sseData
	if err := json.Unmarshal(ev.Data, &envelope); err != nil {
		return
	}

	event := WebsiteEvent{Type: envelope.Type}
	switch envelope.Type {
	case WebsiteEventPublished:
		var pub WebsitePublishedEvent
		if err := json.Unmarshal(envelope.Data, &pub); err != nil {
			return
		}
		event.Published = &pub
	case WebsiteEventRemoved:
		var rem WebsiteRemovedEvent
		if err := json.Unmarshal(envelope.Data, &rem); err != nil {
			return
		}
		event.Removed = &rem
	default:
		return
	}

	c.handlerMu.RLock()
	h := c.handler
	c.handlerMu.RUnlock()
	if h != nil {
		h(event)
	}
}
