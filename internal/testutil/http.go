package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// HTTPTestHandler is a type for HTTP request handlers in tests
type HTTPTestHandler func(w http.ResponseWriter, r *http.Request)

// ServeHTTP implements http.Handler interface
func (h HTTPTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(w, r)
}

// ServerResponse holds the configuration for building a test server response
type ServerResponse struct {
	StatusCode int
	Body       any
	Headers    map[string]string
}

// HTTPTestServerConfig configures a test HTTP server
type HTTPTestServerConfig struct {
	Handler      HTTPTestHandler
	StatusCode   int
	ResponseBody any
	Headers      map[string]string
}

// NewTestServer creates a new httptest.Server with the given configuration
func NewTestServer(t *testing.T, cfg HTTPTestServerConfig) *httptest.Server {
	handler := cfg.Handler
	if handler == nil {
		handler = defaultHandler(t, cfg)
	}
	return httptest.NewServer(handler)
}

// defaultHandler creates a default HTTP handler based on server config
func defaultHandler(t *testing.T, cfg HTTPTestServerConfig) HTTPTestHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		for k, v := range cfg.Headers {
			w.Header().Set(k, v)
		}

		// Set default Content-Type for JSON responses
		if cfg.ResponseBody != nil && w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}

		w.WriteHeader(cfg.StatusCode)

		// Write response body
		if cfg.ResponseBody != nil {
			if err := json.NewEncoder(w).Encode(cfg.ResponseBody); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		}
	}
}

// WriteJSON writes JSON response with status code
func WriteJSON(t *testing.T, w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}
}

// VerifyMethod checks if request method matches expected
func VerifyMethod(t *testing.T, r *http.Request, expectedMethod string) {
	if r.Method != expectedMethod {
		t.Errorf("expected %s request, got %s", expectedMethod, r.Method)
	}
}

// VerifyPath checks if request path matches expected
func VerifyPath(t *testing.T, r *http.Request, expectedPath string) {
	if r.URL.Path != expectedPath {
		t.Errorf("expected %s path, got %s", expectedPath, r.URL.Path)
	}
}

// VerifyAuthorization checks if Authorization header matches expected
func VerifyAuthorization(t *testing.T, r *http.Request, expectedToken string) {
	authHeader := r.Header.Get("Authorization")
	expectedAuth := "Bearer " + expectedToken
	if authHeader != expectedAuth {
		t.Errorf("expected Authorization header %q, got %q", expectedAuth, authHeader)
	}
}

// JSONResponseBuilder helps build JSON responses in tests
type JSONResponseBuilder struct {
	statusCode int
	headers    map[string]string
	body       any
}

// NewJSONResponse creates a new JSON response builder
func NewJSONResponse() *JSONResponseBuilder {
	return &JSONResponseBuilder{
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}

// WithStatus sets the status code
func (b *JSONResponseBuilder) WithStatus(code int) *JSONResponseBuilder {
	b.statusCode = code
	return b
}

// WithHeader adds a header
func (b *JSONResponseBuilder) WithHeader(key, value string) *JSONResponseBuilder {
	b.headers[key] = value
	return b
}

// WithBody sets the response body
func (b *JSONResponseBuilder) WithBody(body any) *JSONResponseBuilder {
	b.body = body
	return b
}

// Write writes the configured response to the http.ResponseWriter
func (b *JSONResponseBuilder) Write(t *testing.T, w http.ResponseWriter) {
	for k, v := range b.headers {
		w.Header().Set(k, v)
	}

	// Set default Content-Type
	if _, ok := b.headers["Content-Type"]; !ok && b.body != nil {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(b.statusCode)
	if b.body != nil {
		if err := json.NewEncoder(w).Encode(b.body); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}
}
