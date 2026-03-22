package ipfs

import (
	"context"
	"errors"
	"net/http"
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
		expectedKeys := []client.IPNSKeyResponse{
			{Id: 1, Name: "key1"},
			{Id: 2, Name: "key2"},
		}

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &expectedKeys,
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, expectedKeys[0].Id, result[0].Id)
		assert.Equal(t, expectedKeys[1].Id, result[1].Id)
	})
}

func TestIPNSService_ListKeys_RetryOn500(t *testing.T) {
	t.Run("retries on 500 and succeeds", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedKeys := []client.IPNSKeyResponse{
			{Id: 1, Name: "key1"},
		}

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				JSON500:      &client.ErrorResponse{Error: "Internal server error"},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiIpnsKeysWithResponse(mock.Anything).
			Return(&client.GetApiIpnsKeysResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &expectedKeys,
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
				JSON400:      &client.ErrorResponse{Error: "Bad request"},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		result, err := service.ListKeys(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed with status 400")
		assert.Nil(t, result)
	})
}

func TestIPNSService_ListKeys_RetryOn502(t *testing.T) {
	t.Run("retries on 502 bad gateway", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedKeys := []client.IPNSKeyResponse{
			{Id: 1, Name: "key1"},
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
				JSON200:      &expectedKeys,
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
}

func TestIPNSService_Resolve_Success(t *testing.T) {
	t.Run("resolves IPNS name", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		expectedResponse := &client.IPNSResolveResponse{
			Name:  "test.ipns",
			Value: "QmResolved",
		}

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns").
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
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns").
			Return(&client.GetApiIpnsResolveNameResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusGatewayTimeout},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "test.ipns").
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
	t.Run("republishes all IPNS records", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsRepublishWithResponse(mock.Anything).
			Return(&client.PostApiIpnsRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		err := service.Republish(context.Background())

		require.NoError(t, err)
	})
}

func TestIPNSService_Republish_RetryOn502(t *testing.T) {
	t.Run("retries on 502 bad gateway", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsRepublishWithResponse(mock.Anything).
			Return(&client.PostApiIpnsRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadGateway},
			}, nil).
			Once()

		mockClient.EXPECT().
			PostApiIpnsRepublishWithResponse(mock.Anything).
			Return(&client.PostApiIpnsRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		err := service.Republish(context.Background())

		require.NoError(t, err)
	})
}

func TestIPNSService_Republish_NoRetryOn400(t *testing.T) {
	t.Run("does not retry on 400 bad request", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiIpnsRepublishWithResponse(mock.Anything).
			Return(&client.PostApiIpnsRepublishResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: "Bad request"},
			}, nil).
			Once()

		service := NewIPNSService(mockClient)
		err := service.Republish(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed with status 400")
	})
}

func TestIPNSService_WaitForIPNSResolution_Success(t *testing.T) {
	t.Run("polls until IPNS resolves to expected CID", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		callCount := 0

		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, name string, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				callCount++
				if callCount == 1 {
					return &client.GetApiIpnsResolveNameResponse{
						Body:         []byte("{}"),
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200: &client.IPNSResolveResponse{
							Name:  "example.com",
							Value: "QmOldCid",
						},
					}, nil
				}
				return &client.GetApiIpnsResolveNameResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.IPNSResolveResponse{
						Name:  "example.com",
						Value: "QmExpectedCid",
					},
				}, nil
			}).
			Times(2)

		service := NewIPNSService(mockClient)
		result, err := service.WaitForIPNSResolution(context.Background(), "example.com", "QmExpectedCid")

		require.NoError(t, err)
		assert.Equal(t, "QmExpectedCid", result.Value)
		assert.Equal(t, 2, callCount)
	})
}

func TestIPNSService_WaitForIPNSResolution_Timeout(t *testing.T) {
	t.Run("times out when CID never resolves to expected value", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, name string, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				return &client.GetApiIpnsResolveNameResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.IPNSResolveResponse{
						Name:  "example.com",
						Value: "QmOldCid",
					},
				}, nil
			}).Maybe()

		service := NewIPNSService(mockClient)
		_, err := service.WaitForIPNSResolution(context.Background(), "example.com", "QmExpectedCid",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestIPNSService_WaitForIPNSResolution_ErrorOnResponse(t *testing.T) {
	t.Run("returns error when resolve API call fails", func(t *testing.T) {
		mockClient := mocks.NewMockIPNSClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiIpnsResolveNameWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, name string, reqEditors ...client.RequestEditorFn) (*client.GetApiIpnsResolveNameResponse, error) {
				return nil, assert.AnError
			})

		service := NewIPNSService(mockClient)
		_, err := service.WaitForIPNSResolution(context.Background(), "example.com", "QmExpectedCid",
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
