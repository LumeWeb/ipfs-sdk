package sse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConnectionSendsHeadersAndLastEventID(t *testing.T) {
	var (
		mu        sync.Mutex
		gotSecret string
		gotLastID string
		gotAccept string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotSecret = r.Header.Get("X-Gateway-Secret")
		gotLastID = r.Header.Get("Last-Event-ID")
		gotAccept = r.Header.Get("Accept")
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := NewConnection(srv.URL)
	conn.SetHeader("X-Gateway-Secret", "s3cr3t")
	conn.SetLastEventID("42")

	_, err := conn.Connect()
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	conn.Close()

	mu.Lock()
	defer mu.Unlock()
	if gotSecret != "s3cr3t" {
		t.Errorf("X-Gateway-Secret header = %q, want %q", gotSecret, "s3cr3t")
	}
	if gotLastID != "42" {
		t.Errorf("Last-Event-ID header = %q, want %q", gotLastID, "42")
	}
	if gotAccept != "text/event-stream" {
		t.Errorf("Accept header = %q, want text/event-stream", gotAccept)
	}
}

func TestConnectionRejectsNonEventStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if _, err := NewConnection(srv.URL).Connect(); err == nil {
		t.Fatal("expected error for non-event-stream content type")
	}
}

func TestClientConnectsAndDispatchesEvents(t *testing.T) {
	var mu sync.Mutex
	var gotSecret string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotSecret = r.Header.Get("X-Gateway-Secret")
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "id: 1\nevent: site_published\ndata: {\"domain\":\"example.com\"}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := NewClient(srv.URL, Options{Reconnect: false})
	client.SetHeader("X-Gateway-Secret", "s3cr3t")
	received := make(chan Event, 1)
	client.OnEvent("site_published", func(ev Event) {
		received <- ev
	})

	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer client.Disconnect()

	select {
	case ev := <-received:
		if ev.ID != "1" {
			t.Errorf("event ID = %q, want 1", ev.ID)
		}
		if ev.Type != "site_published" {
			t.Errorf("event type = %q, want site_published", ev.Type)
		}
		if !strings.Contains(string(ev.Data), "example.com") {
			t.Errorf("event data = %q, want domain payload", ev.Data)
		}
		if client.LastEventID() != "1" {
			t.Errorf("LastEventID() = %q, want 1", client.LastEventID())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotSecret != "s3cr3t" {
		t.Errorf("X-Gateway-Secret header = %q, want %q", gotSecret, "s3cr3t")
	}
}

func TestClient_StatsHooks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "id: 3\nevent: site_published\ndata: {}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	received := make(chan struct{}, 1)
	connected := make(chan bool, 4)

	client := NewClient(srv.URL, Options{Reconnect: false})
	client.SetStats(Stats{
		EventReceived: func() { received <- struct{}{} },
		Connected:     func(v bool) { connected <- v },
	})
	client.OnEvent("site_published", func(Event) {})

	if err := client.Connect(); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer client.Disconnect()

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("EventReceived hook not fired")
	}

	select {
	case v := <-connected:
		if !v {
			t.Error("Connected hook reported false on an open stream")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Connected(true) hook not fired")
	}

	client.Disconnect()
	select {
	case v := <-connected:
		if v {
			t.Error("Connected hook reported true after disconnect")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Connected(false) hook not fired")
	}
}

func TestClientReconnectsWithLastEventID(t *testing.T) {
	var (
		mu          sync.Mutex
		connections int
		lastIDs     []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connections++
		conn := connections
		lastIDs = append(lastIDs, r.Header.Get("Last-Event-ID"))
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// First connection delivers one durable event then returns, closing the
		// stream to force a reconnect carrying the cursor.
		if conn == 1 {
			fmt.Fprint(w, "id: 7\nevent: site_published\ndata: {}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := NewClient(srv.URL, Options{Reconnect: true, Backoff: 10 * time.Millisecond, MaxBackoff: 10 * time.Millisecond})
	received := make(chan Event, 4)
	client.OnEvent("site_published", func(ev Event) { received <- ev })

	if err := client.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer client.Disconnect()

	select {
	case ev := <-received:
		if ev.ID != "7" {
			t.Errorf("first event ID = %q, want 7", ev.ID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for first SSE event")
	}

	// Wait for the reconnect to produce a second connection carrying the cursor.
	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		conns := connections
		mu.Unlock()
		if conns >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("client did not reconnect; connections=%d", conns)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	mu.Lock()
	conns := len(lastIDs)
	ids := append([]string(nil), lastIDs...)
	mu.Unlock()
	if conns < 2 {
		t.Fatalf("expected >= 2 connections, got %d", conns)
	}
	if ids[1] != "7" {
		t.Errorf("reconnect Last-Event-ID = %q, want 7 (received cursor)", ids[1])
	}
}
