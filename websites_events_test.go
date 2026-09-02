package ipfs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestWebsiteEventsClient_StreamsLifecycleEvents verifies the SSE client
// consumes the portal's website lifecycle stream: it authenticates with the
// gateway secret and dispatches correctly-parsed published/removed events.
func TestWebsiteEventsClient_StreamsLifecycleEvents(t *testing.T) {
	var mu sync.Mutex
	var gotSecret string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotSecret = r.Header.Get("X-Gateway-Secret")
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprint(w,
			"event: site_published\n"+
				"id: 1\n"+
				"data: {\"type\":\"site_published\",\"data\":{\"domain\":\"example.com\",\"cid\":\"QmPublish\",\"published_at\":\"2026-09-01T00:00:00Z\"}}\n\n"+
				"event: site_removed\n"+
				"id: 2\n"+
				"data: {\"type\":\"site_removed\",\"data\":{\"domain\":\"example.org\",\"removed_at\":\"2026-09-02T00:00:00Z\"}}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	client, err := NewWebsiteEventsClient(srv.URL, "s3cr3t", WithWebsiteEventsReconnect(false))
	if err != nil {
		t.Fatalf("NewWebsiteEventsClient() error: %v", err)
	}

	events := make(chan WebsiteEvent, 2)
	client.OnEvent(func(ev WebsiteEvent) {
		events <- ev
	})

	client.Start()
	defer client.Stop()

	// Published event.
	select {
	case ev := <-events:
		if ev.Type != WebsiteEventPublished {
			t.Fatalf("event type = %q, want site_published", ev.Type)
		}
		if ev.Published == nil {
			t.Fatal("published payload missing")
		}
		if ev.Published.Domain != "example.com" || ev.Published.CID != "QmPublish" {
			t.Errorf("published event = %+v, want domain example.com / cid QmPublish", ev.Published)
		}
		if ev.Removed != nil {
			t.Error("removed payload should be nil for published event")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for published event")
	}

	// Removed event.
	select {
	case ev := <-events:
		if ev.Type != WebsiteEventRemoved {
			t.Fatalf("event type = %q, want site_removed", ev.Type)
		}
		if ev.Removed == nil {
			t.Fatal("removed payload missing")
		}
		if ev.Removed.Domain != "example.org" {
			t.Errorf("removed event domain = %q, want example.org", ev.Removed.Domain)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for removed event")
	}

	if client.LastEventID() != "2" {
		t.Errorf("LastEventID() = %q, want 2", client.LastEventID())
	}

	mu.Lock()
	defer mu.Unlock()
	if gotSecret != "s3cr3t" {
		t.Errorf("X-Gateway-Secret header = %q, want %q", gotSecret, "s3cr3t")
	}
}

// TestWebsiteEventsClient_OnEventAfterStartConcurrent replaces the handler while
// events are streaming to prove handler access is synchronized (run with -race).
func TestWebsiteEventsClient_OnEventAfterStartConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: site_published\nid: 1\ndata: {\"type\":\"site_published\",\"data\":{\"domain\":\"example.com\",\"cid\":\"QmX\",\"published_at\":\"2026-09-01T00:00:00Z\"}}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	client, err := NewWebsiteEventsClient(srv.URL, "test-gateway-secret", WithWebsiteEventsReconnect(false))
	if err != nil {
		t.Fatalf("NewWebsiteEventsClient() error: %v", err)
	}
	client.OnEvent(func(WebsiteEvent) {})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			client.OnEvent(func(WebsiteEvent) {})
		}
	}()

	client.Start()
	time.Sleep(50 * time.Millisecond)
	client.Stop()
	wg.Wait()
}
