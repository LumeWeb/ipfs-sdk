// Package testutil provides testing utilities for HTTP handlers and responses.
//
// This package contains helpers for writing tests that interact with HTTP
// servers and handlers. It provides utilities for creating test servers,
// building test requests, and validating responses.
//
// Test Server
//
// Create a test HTTP server with custom configuration:
//
//   server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
//       StatusCode: 200,
//       ResponseBody: expectedData,
//       Headers: map[string]string{"Content-Type": "application/json"},
//   })
//   defer server.Close()
//
// Response Helpers
//
// Build complex JSON responses:
//
//   response := testutil.NewJSONResponse().
//       WithStatus(http.StatusCreated).
//       WithHeader("X-Custom-Header", "value").
//       WithBody(expectedStruct).
//       Write(t, w)
//
// Verification Helpers
//
// Verify HTTP request properties:
//
//   testutil.VerifyMethod(t, r, "POST")
//   testutil.VerifyPath(t, r, "/api/endpoint")
//   testutil.VerifyAuthorization(t, r, "token123")
package testutil
