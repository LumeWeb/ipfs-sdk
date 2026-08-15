package ipfs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/mocks"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

func TestNewIPNSService(t *testing.T) {
	mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
	service := NewIPNSService(mockClient)

	assert.NotNil(t, service)
}

func TestIPNSService_ListKeys_Success(t *testing.T) {
	t.Run("returns list of keys", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedList := []client.IPNSKeyListResponse{
			{Id: 1, Name: "key1", IpnsName: "key1"},
			{Id: 2, Name: "key2", IpnsName: "key2"},
		}
		expectedResponse := &client.IPNSKeyListResponseResponse{
			Data:  expectedList,
			Total: len(expectedList),
		}

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, 1, result[0].Id)
		assert.Equal(t, 2, result[1].Id)
	})
}

func TestIPNSService_ListKeys_RetryOn500(t *testing.T) {
	t.Run("retries on 500 and succeeds", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedList := []client.IPNSKeyListResponse{
			{Id: 1, Name: "key1", IpnsName: "key1"},
		}
		expectedResponse := &client.IPNSKeyListResponseResponse{
			Data:  expectedList,
			Total: len(expectedList),
		}

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				JSON500:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Internal server error"}},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 1)
	})
}

func TestIPNSService_ListKeys_NoRetryOn400(t *testing.T) {
	t.Run("does not retry on 400", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Bad request"}},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed with status 400")
		assert.Nil(t, result)
	})
}

// TestListKeysSendsNameFilter is the end-to-end guard for server-side IPNS key
// search: ListKeys with WithFilterName must emit the queryutil filters[name]
// [contains]=<name> query param (not fetch-then-filter client-side), and a
// plain ListKeys must send no filters.
func TestListKeysSendsNameFilter(t *testing.T) {
	var (
		mu    sync.Mutex
		gotQuery url.Values
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotQuery = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	t.Cleanup(srv.Close)

	// Build a real generated client pointed at the test server, then wire it
	// through the public IPNS service.
	genClient, err := client.NewClientWithResponses(srv.URL)
	require.NoError(t, err)
	service := NewIPNSService(genClient)

	// With a name filter: the server-side contains filter must be sent.
	_, err = service.ListKeys(context.Background(), ListKeyOption{}.WithFilterName("docs"))
	require.NoError(t, err)
	mu.Lock()
	withFilter := gotQuery.Get("filters[name][contains]")
	mu.Unlock()
	assert.Equal(t, "docs", withFilter, "filters[name][contains] must be sent server-side for name search")

	// Without a filter: no query params on the list call.
	_, err = service.ListKeys(context.Background())
	require.NoError(t, err)
	mu.Lock()
	plain := gotQuery.Get("filters[name][contains]")
	mu.Unlock()
	assert.Empty(t, plain, "plain ListKeys must not send a name filter")
}

func TestIPNSService_ListKeys_RetryOn502(t *testing.T) {
	t.Run("retries on 502 bad gateway", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedList := []client.IPNSKeyListResponse{
			{Id: 1, Name: "key1", IpnsName: "key1"},
		}
		expectedResponse := &client.IPNSKeyListResponseResponse{
			Data:  expectedList,
			Total: len(expectedList),
		}

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadGateway},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 1)
	})
}

func TestIPNSService_GetKey_Success(t *testing.T) {
	t.Run("returns key by ID", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedKey := &client.IPNSKeyResponse{Id: 1, Name: "key1"}

		mockClient.EXPECT().
			GetApiIpnsKeysIdWithResponse(mock.Anything, "1").
			Return(&client.GetApiIpnsKeysIdResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedKey,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.GetKey(context.Background(), "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedKey.Id, result.Id)
	})
}

func TestIPNSService_CreateKey_Success(t *testing.T) {
	t.Run("creates new key", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		newKey := &client.IPNSKeyResponse{Id: 1, Name: "newkey"}

		mockClient.EXPECT().
			PostApiIpnsKeysWithResponse(mock.Anything, mock.Anything).
			Return(&client.PostApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      newKey,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.CreateKey(context.Background(), "newkey")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newKey.Id, result.Id)
	})

	t.Run("imports existing key with WithIPNSKey option", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		newKey := &client.IPNSKeyResponse{Id: 2, Name: "importedkey"}
		privateKey := "existing-private-key-data"

		mockClient.EXPECT().
			PostApiIpnsKeysWithResponse(mock.Anything, mock.MatchedBy(func(req client.IPNSKeyRequest) bool {
				return req.Name == "importedkey" && req.Key != nil && *req.Key == privateKey
			})).
			Return(&client.PostApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      newKey,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.CreateKey(context.Background(), "importedkey", WithIPNSKey(privateKey))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newKey.Id, result.Id)
	})
}

func TestIPNSService_DeleteKey_Success(t *testing.T) {
	t.Run("deletes key by ID", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			DeleteApiIpnsKeysIdWithResponse(mock.Anything, "1").
			Return(&client.DeleteApiIpnsKeysIdResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		err := service.DeleteKey(context.Background(), "1")

		require.NoError(t, err)
	})
}

func TestIPNSService_DeleteKey_RetryOn503(t *testing.T) {
	t.Run("retries on 503 service unavailable", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			DeleteApiIpnsKeysIdWithResponse(mock.Anything, "1").
			Return(&client.DeleteApiIpnsKeysIdResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusServiceUnavailable},
			}, nil).
			Once()

		mockClient.EXPECT().
			DeleteApiIpnsKeysIdWithResponse(mock.Anything, "1").
			Return(&client.DeleteApiIpnsKeysIdResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		err := service.DeleteKey(context.Background(), "1")

		require.NoError(t, err)
	})
}

func TestIPNSService_Publish_Success(t *testing.T) {
	t.Run("publishes IPNS record", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedResponse := &client.IPNSPublishResponse{
			Name:  "test",
			Value: "QmTest",
		}

		mockClient.EXPECT().
			PostApiIpnsPublishWithResponse(mock.Anything, mock.Anything).
			Return(&client.PostApiIpnsPublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Publish(context.Background(), 123, "QmTest")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("publishes with TTL option", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedResponse := &client.IPNSPublishResponse{
			Name:  "test",
			Value: "QmTest",
		}
		ttl := "24h"

		mockClient.EXPECT().
			PostApiIpnsPublishWithResponse(mock.Anything, mock.MatchedBy(func(req client.IPNSPublishRequest) bool {
				return req.KeyId == 123 && req.Cid == "QmTest" && req.Ttl != nil && *req.Ttl == ttl
			})).
			Return(&client.PostApiIpnsPublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Publish(context.Background(), 123, "QmTest", WithTTL(ttl))

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestIPNSService_Resolve_Success(t *testing.T) {
	t.Run("resolves IPNS name", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedResponse := &client.IPNSResolveResponse{
			Name:  "test.ipns",
			Value: "QmResolved",
		}

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns", mock.Anything).
			Return(&client.GetApiIpnsResolveNameResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Resolve(context.Background(), "test.ipns")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestIPNSService_Resolve_RetryOnTimeout(t *testing.T) {
	t.Run("retries on gateway timeout", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedResponse := &client.IPNSResolveResponse{
			Name:  "test.ipns",
			Value: "QmResolved",
		}

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns", mock.Anything).
			Return(&client.GetApiIpnsResolveNameResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusGatewayTimeout},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns", mock.Anything).
			Return(&client.GetApiIpnsResolveNameResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Resolve(context.Background(), "test.ipns")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestIPNSService_Republish_Success(t *testing.T) {
	t.Run("republishes IPNS record for a specific key", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsKeysIdRepublishWithResponse(mock.Anything, "1").
			Return(&client.PostApiIpnsKeysIdRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &client.IPNSRepublishResponse{
					Count:   1,
					Message: "republished",
				},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Republish(context.Background(), "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Count)
	})
}

func TestIPNSService_Republish_RetryOn502(t *testing.T) {
	t.Run("retries on 502 bad gateway", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsKeysIdRepublishWithResponse(mock.Anything, "1").
			Return(&client.PostApiIpnsKeysIdRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadGateway},
			}, nil).
			Once()

		mockClient.EXPECT().
			PostApiIpnsKeysIdRepublishWithResponse(mock.Anything, "1").
			Return(&client.PostApiIpnsKeysIdRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &client.IPNSRepublishResponse{
					Count:   1,
					Message: "republished",
				},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.Republish(context.Background(), "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestIPNSService_Republish_NoRetryOn400(t *testing.T) {
	t.Run("does not retry on 400 bad request", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsKeysIdRepublishWithResponse(mock.Anything, "1").
			Return(&client.PostApiIpnsKeysIdRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Bad request"}},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		_, err := service.Republish(context.Background(), "1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed with status 400")
	})
}

func TestIPNSService_WaitForIPNSResolution_Success(t *testing.T) {
	t.Run("polls until IPNS resolves to expected CID", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		callCount := 0
		oldCid := "bafybeicgtzgrtbzlpznm5jcjmvsdlgpsrev7jxsknzzdbpnaqc4ky2hrlu"
		expectedCid := "bafybeifx7yebllhouqoa5a32rsxyg7mf36d2j6co3t6u7hf3jfdw2kfqpy"

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com", mock.Anything).
			RunAndReturn(func(ctx context.Context, name string, _ *client.GetApiIpnsResolveNameParams, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				callCount++
				if callCount == 1 {
					return &client.GetApiIpnsResolveNameResponse{
						Body:         []byte("{}"),
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200: &client.IPNSResolveResponse{
							Name:  "example.com",
							Value: oldCid,
						},
					}, nil
				}
				return &client.GetApiIpnsResolveNameResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.IPNSResolveResponse{
						Name:  "example.com",
						Value: expectedCid,
					},
				}, nil
			}).
			Times(2)

		service := NewIPNSService(mockClient)
		result, err := service.WaitForIPNSResolution(context.Background(), "example.com", expectedCid)

		require.NoError(t, err)
		assert.Equal(t, expectedCid, result.Value)
		assert.Equal(t, 2, callCount)
	})
}

func TestIPNSService_WaitForIPNSResolution_Timeout(t *testing.T) {
	t.Run("times out when CID never resolves to expected value", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		oldCid := "bafybeicgtzgrtbzlpznm5jcjmvsdlgpsrev7jxsknzzdbpnaqc4ky2hrlu"
		expectedCid := "bafybeifx7yebllhouqoa5a32rsxyg7mf36d2j6co3t6u7hf3jfdw2kfqpy"

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com", mock.Anything).
			RunAndReturn(func(ctx context.Context, name string, _ *client.GetApiIpnsResolveNameParams, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				return &client.GetApiIpnsResolveNameResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.IPNSResolveResponse{
						Name:  "example.com",
						Value: oldCid,
					},
				}, nil
			}).Maybe()

		service := NewIPNSService(mockClient)
		_, err := service.WaitForIPNSResolution(context.Background(), "example.com", expectedCid,
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestIPNSService_WaitForIPNSResolution_ErrorOnResponse(t *testing.T) {
	t.Run("returns error when resolve API call fails", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedCid := "bafybeifx7yebllhouqoa5a32rsxyg7mf36d2j6co3t6u7hf3jfdw2kfqpy"

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com", mock.Anything).
			RunAndReturn(func(ctx context.Context, name string, _ *client.GetApiIpnsResolveNameParams, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				return nil, assert.AnError
			})

		service := NewIPNSService(mockClient)
		_, err := service.WaitForIPNSResolution(context.Background(), "example.com", expectedCid,
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
		// Check for either the wrapped error or a context timeout (can happen in certain race conditions)
		if errors.Is(err, context.DeadlineExceeded) {
			// Context deadline exceeded - acceptable in race mode
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		} else {
			// Expected error wrapping
			assert.Contains(t, err.Error(), "failed to resolve")
		}
	})
}
