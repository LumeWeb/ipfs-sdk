// Package sse provides a reconnecting Server-Sent Events (SSE) client used by
// the gateway-facing internal APIs. Wire parsing is delegated to
// github.com/apt304/sse-go so the stream handling matches the portal's own SSE
// server.
package sse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	aptsse "github.com/apt304/sse-go/server"
	"github.com/avast/retry-go/v4"
	"go.uber.org/zap"
)

// Event is a single parsed Server-Sent Event frame.
type Event = aptsse.Event

// EventHandler processes a parsed SSE event.
type EventHandler func(Event)

// ErrorHandler processes a connection-level error.
type ErrorHandler func(error)

// ConnectionState describes the lifecycle of an SSE connection.
type ConnectionState int

const (
	// StateDisconnected indicates no active SSE connection.
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates the initial connection attempt is in progress.
	StateConnecting
	// StateConnected indicates an SSE stream is open.
	StateConnected
	// StateReconnecting indicates the client is retrying after a dropped stream.
	StateReconnecting
)

// Stats exposes optional observability callbacks. Downstream consumers (e.g.
// the gateway) map these to a metrics backend such as Prometheus. All fields
// are optional; leaving them nil is a no-op. Implementations must be safe to
// call concurrently.
type Stats struct {
	// EventReceived is called once for every dispatched website lifecycle event.
	EventReceived func()
	// ParseError is called when an SSE frame or payload cannot be parsed.
	ParseError func()
	// ConnectionError is called when a connection or reconnection fails.
	ConnectionError func(err error)
	// Connected is called with the live connection state, either on state
	// transitions or on a fixed interval when ConnectedPollInterval is set.
	Connected func(connected bool)
}

// Options configures reconnection behaviour.
type Options struct {
	// Reconnect enables automatic reconnection after the stream drops.
	Reconnect bool
	// Backoff is the initial reconnection delay.
	Backoff time.Duration
	// MaxBackoff caps the exponential reconnection delay.
	MaxBackoff time.Duration
	// MaxRetries caps reconnection attempts (0 = unlimited).
	MaxRetries int
	// ConnectedPollInterval drives the optional connected-state telemetry
	// poller. When greater than zero and a Connected stats callback is
	// registered, Start reports the live connection state on this interval for
	// the client's lifetime (0 disables the poller).
	ConnectedPollInterval time.Duration
}

// ErrClientClosed is returned by Connect/Start once the client has been
// disconnected via Disconnect. A Client is single-use.
var ErrClientClosed = fmt.Errorf("sse client has been closed, cannot reconnect")

// Connection performs a single HTTP GET to an SSE endpoint and streams parsed
// frames until the server closes the stream or the connection is cancelled.
type Connection struct {
	client      *http.Client
	url         string
	lastEventID string
	headers     map[string]string
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewConnection creates an SSE connection. The underlying HTTP client streams
// an unbounded response body (no overall timeout) but enforces a response
// header timeout so a peer that never sends headers cannot stall reconnects.
func NewConnection(url string) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second

	return &Connection{
		client: &http.Client{
			Timeout:   0,
			Transport: transport,
		},
		url:     url,
		headers: make(map[string]string),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetHeader sets a header sent with the request. Must be called before Connect.
func (c *Connection) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetLastEventID sets the Last-Event-ID header for durable replay.
func (c *Connection) SetLastEventID(id string) {
	c.lastEventID = id
}

// Connect opens the stream and returns a channel of parsed events. The channel
// is closed when the server ends the stream or the connection is closed.
func (c *Connection) Connect() (<-chan Event, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create SSE request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open SSE connection: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected SSE status code: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		resp.Body.Close()
		return nil, fmt.Errorf("invalid SSE content type: %s", ct)
	}

	eventChan := make(chan Event, 100)
	go c.readEvents(resp.Body, eventChan)

	return eventChan, nil
}

func (c *Connection) readEvents(body io.ReadCloser, eventChan chan<- Event) {
	defer body.Close()
	defer close(eventChan)

	reader := bufio.NewReader(body)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		event, err := aptsse.ParseEvent(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			select {
			case eventChan <- Event{Type: "error", Data: []byte(fmt.Sprintf("parse error: %v", err))}:
			case <-c.ctx.Done():
				return
			}
			return
		}

		select {
		case eventChan <- event:
		case <-c.ctx.Done():
			return
		}
	}
}

// Close cancels the underlying request context.
func (c *Connection) Close() {
	c.cancel()
}

// IsClosed reports whether the connection has been cancelled.
func (c *Connection) IsClosed() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// Client is a reconnecting SSE client. It re-sends the Last-Event-ID header on
// each reconnection, applies custom headers (e.g. authentication) to every
// attempt, routes parsed frames to per-type handlers, and reports optional
// telemetry via Stats.
type Client struct {
	mu        sync.RWMutex
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
	logger    *zap.Logger
	state     ConnectionState
	stateMu   sync.RWMutex
	connMu    sync.Mutex
	conn      *Connection
	// spawnMu serializes the done-closure (Disconnect) with wg.Add in Connect so
	// no Add can race a concurrent Wait after shutdown begins.
	spawnMu       sync.Mutex
	url           string
	options       Options
	headers       map[string]string
	stats         Stats
	statsMu       sync.RWMutex
	lastEventID   string
	lastEventIDMu sync.RWMutex
	eventHandlers map[string][]EventHandler
	errorHandler  ErrorHandler
}

// NewClient creates an SSE client with reconnection enabled by default.
func NewClient(url string, opts ...Options) *Client {
	options := Options{
		Reconnect:  true,
		Backoff:    1 * time.Second,
		MaxBackoff: 30 * time.Second,
		MaxRetries: 10,
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	return &Client{
		url:           url,
		options:       options,
		eventHandlers: make(map[string][]EventHandler),
		headers:       make(map[string]string),
		done:          make(chan struct{}),
	}
}

// SetHeader sets a header sent with every connection. Must be called before
// Connect/Start.
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// SetLastEventID sets the initial cursor sent as Last-Event-ID on the first
// connection (and on every reconnection, where it is updated automatically).
func (c *Client) SetLastEventID(id string) {
	c.lastEventIDMu.Lock()
	defer c.lastEventIDMu.Unlock()
	c.lastEventID = id
}

// SetLogger sets the diagnostics logger. When unset, logs are dropped.
func (c *Client) SetLogger(logger *zap.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

func (c *Client) log() *zap.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.logger == nil {
		return zap.NewNop()
	}
	return c.logger
}

// SetStats wires optional observability callbacks. Must be called before
// Connect/Start.
func (c *Client) SetStats(stats Stats) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats = stats
}

func (c *Client) fireEventReceived() {
	c.statsMu.RLock()
	s := c.stats
	c.statsMu.RUnlock()
	if s.EventReceived != nil {
		s.EventReceived()
	}
}
func (c *Client) fireParseError() {
	c.statsMu.RLock()
	s := c.stats
	c.statsMu.RUnlock()
	if s.ParseError != nil {
		s.ParseError()
	}
}
func (c *Client) fireConnectionError(err error) {
	c.statsMu.RLock()
	s := c.stats
	c.statsMu.RUnlock()
	if s.ConnectionError != nil {
		s.ConnectionError(err)
	}
}
func (c *Client) fireConnected(connected bool) {
	c.statsMu.RLock()
	s := c.stats
	c.statsMu.RUnlock()
	if s.Connected != nil {
		s.Connected(connected)
	}
}

// OnEvent registers a handler for a specific event type.
func (c *Client) OnEvent(eventType string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)
}

// OnError registers a handler called when reconnection is permanently given up.
func (c *Client) OnError(handler ErrorHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorHandler = handler
}

// Connect establishes the SSE stream. It returns nil once the stream is open;
// the client's internal handler then owns the stream and any reconnection. A
// Client that has been Disconnect()ed returns ErrClientClosed.
func (c *Client) Connect() error {
	select {
	case <-c.done:
		return ErrClientClosed
	default:
	}

	c.setState(StateConnecting)

	conn := NewConnection(c.url)
	conn.SetLastEventID(c.getLastEventID())
	for key, value := range c.headersSnapshot() {
		conn.SetHeader(key, value)
	}

	// Publish the connection up front so a concurrent Disconnect can cancel it
	// while the blocking request below is in flight. Re-check done in the same
	// critical section: a Disconnect that slipped in before publication left
	// c.conn nil and cannot cancel this new connection, so close it here.
	c.connMu.Lock()
	select {
	case <-c.done:
		c.connMu.Unlock()
		conn.Close()
		return ErrClientClosed
	default:
		c.conn = conn
	}
	c.connMu.Unlock()

	eventChan, err := conn.Connect()
	if err != nil {
		c.connMu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.connMu.Unlock()
		c.setState(StateDisconnected)
		c.fireConnectionError(err)
		return err
	}

	c.setState(StateConnected)
	c.fireConnected(true)

	// Add under spawnMu, re-checking done, so an Add can never race the Wait
	// that follows Disconnect after shutdown has begun.
	c.spawnMu.Lock()
	select {
	case <-c.done:
		c.spawnMu.Unlock()
		conn.Close()
		return ErrClientClosed
	default:
		c.wg.Add(1)
	}
	c.spawnMu.Unlock()

	go func() {
		defer c.wg.Done()
		c.handleEvents(eventChan)
	}()

	return nil
}

// Start connects with retry-go (the same strategy used across the SDK), backing
// off between attempts until a connection is established, the client is
// stopped, or MaxRetries is exhausted. It is blocking and intended to be run in
// a goroutine. When a Connected stats callback is registered, the polling
// interval in Options drives a periodic connected-state report for the client's
// lifetime.
func (c *Client) Start() error {
	// Opt-in connected-state poller: keeps a downstream connected gauge fresh
	// during backoff and while connected, driven by the caller's interval.
	if c.options.ConnectedPollInterval > 0 && c.hasConnectedStats() {
		c.wg.Add(1)
		go c.pollConnected()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()

	err := retry.Do(
		func() error {
			if cerr := c.Connect(); cerr != nil {
				if cerr == ErrClientClosed {
					return retry.Unrecoverable(cerr)
				}
				return cerr
			}
			return nil
		},
		retry.Attempts(uint(c.options.MaxRetries)),
		retry.Delay(c.options.Backoff),
		retry.MaxDelay(c.options.MaxBackoff),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			c.log().Warn("SSE initial connect failed, retrying",
				zap.Uint("attempt", n+1),
				zap.Error(err))
		}),
	)
	if err != nil && err != ErrClientClosed {
		c.setState(StateDisconnected)
		c.fireConnectionError(err)
	}
	return err
}

func (c *Client) hasConnectedStats() bool {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats.Connected != nil
}

func (c *Client) pollConnected() {
	defer c.wg.Done()
	for {
		c.fireConnected(c.IsConnected())
		select {
		case <-c.done:
			return
		case <-time.After(c.options.ConnectedPollInterval):
		}
	}
}

// handleEvents drains the event channel, dispatching frames to handlers.
func (c *Client) handleEvents(eventChan <-chan Event) {
	for {
		select {
		case <-c.done:
			return
		case event, ok := <-eventChan:
			if !ok {
				c.handleDisconnection()
				return
			}
			c.handleEvent(event)
		}
	}
}

func (c *Client) handleEvent(event Event) {
	if event.ID != "" {
		c.lastEventIDMu.Lock()
		c.lastEventID = event.ID
		c.lastEventIDMu.Unlock()
	}

	if event.IsHeartbeat() {
		return
	}

	if event.Type == "error" {
		c.fireParseError()
		return
	}

	if event.Type != "" && event.Type != "message" {
		c.mu.RLock()
		handlers := c.eventHandlers[event.Type]
		c.mu.RUnlock()
		for _, handler := range handlers {
			handler(event)
		}
		if len(handlers) > 0 {
			c.fireEventReceived()
		}
	}
}

// handleDisconnection reacts to a dropped stream, reconnecting with exponential
// backoff when reconnection is enabled.
func (c *Client) handleDisconnection() {
	c.setState(StateDisconnected)
	c.fireConnected(false)
	if !c.options.Reconnect {
		return
	}

	c.setState(StateReconnecting)
	backoff := c.options.Backoff
	attempt := 0

	for {
		select {
		case <-c.done:
			return
		case <-time.After(addJitter(backoff)):
			attempt++
			c.log().Info("attempting SSE reconnection",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff))

			if err := c.Connect(); err != nil {
				if err == ErrClientClosed {
					return
				}
				c.log().Warn("SSE reconnection failed",
					zap.Int("attempt", attempt),
					zap.Error(err))
				if c.options.MaxRetries > 0 && attempt >= c.options.MaxRetries {
					c.setState(StateDisconnected)
					err := fmt.Errorf("SSE reconnect failed after %d attempts", c.options.MaxRetries)
					if h := c.errorHandlerSnapshot(); h != nil {
						h(err)
					}
					return
				}
				backoff *= 2
				if c.options.MaxBackoff > 0 && backoff > c.options.MaxBackoff {
					backoff = c.options.MaxBackoff
				}
				continue
			}

			c.log().Info("SSE reconnected", zap.Int("attempts", attempt))
			return
		}
	}
}

// Disconnect closes the stream and stops all reconnection. The client is
// single-use afterwards.
func (c *Client) Disconnect() {
	// Close done under spawnMu so that once Disconnect returns, no Connect
	// can be mid-Add and race the subsequent Wait.
	c.closeOnce.Do(func() {
		c.spawnMu.Lock()
		close(c.done)
		c.spawnMu.Unlock()
	})

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	c.setState(StateDisconnected)
	c.fireConnected(false)
}

// Wait blocks until the event-handling and telemetry goroutines have drained.
// Call after Disconnect.
func (c *Client) Wait() {
	c.wg.Wait()
}

// IsConnected reports whether a stream is currently open.
func (c *Client) IsConnected() bool {
	return c.GetState() == StateConnected
}

// GetState returns the current connection state.
func (c *Client) GetState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Client) setState(state ConnectionState) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

// LastEventID returns the most recently received durable event ID.
func (c *Client) LastEventID() string {
	c.lastEventIDMu.RLock()
	defer c.lastEventIDMu.RUnlock()
	return c.lastEventID
}

func (c *Client) getLastEventID() string {
	c.lastEventIDMu.RLock()
	defer c.lastEventIDMu.RUnlock()
	return c.lastEventID
}

func (c *Client) headersSnapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.headers))
	for k, v := range c.headers {
		out[k] = v
	}
	return out
}

func (c *Client) errorHandlerSnapshot() ErrorHandler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errorHandler
}

// addJitter perturbs a backoff by +/- 2% to avoid self-synchronizing reconnects.
func addJitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Float64()*2-1)*time.Duration(float64(d)*0.02)
}
