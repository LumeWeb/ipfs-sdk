package ipfs

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/mocks"
)

func TestNewWebsitesService(t *testing.T) {
	mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
	service := NewWebsitesService(mockClient)

	assert.NotNil(t, service)
}

func TestWebsitesService_List_RetryOn500(t *testing.T) {
	t.Run("retries on 500 and succeeds", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		callTime := make([]time.Time, 0, 2)
		expectedItem := client.WebsiteItem{
			Id:         1,
			Domain:     "example.com",
			Status:     "active",
			TargetHash: "QmXxx",
			TargetType: "ipfs",
		}

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			RunAndReturn(func(ctx context.Context, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesResponse, error) {
				callTime = append(callTime, time.Now())
				return &client.GetApiWebsitesResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
					JSON500:      &client.ErrorResponse{Error: "Internal server error"},
				}, nil
			}).
			Once()

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			RunAndReturn(func(ctx context.Context, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesResponse, error) {
				callTime = append(callTime, time.Now())
				return &client.GetApiWebsitesResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200:      &client.WebsiteItemResponse{Data: expectedItem},
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.List(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, expectedItem.Id, result[0].Id)
		assert.Equal(t, 2, len(callTime), "should have 2 calls due to retry")

		// Verify retry delay was applied
		if len(callTime) >= 2 {
			delay := callTime[1].Sub(callTime[0])
			assert.Greater(t, delay.Milliseconds(), int64(10), "retry should have delay")
		}
	})
}

func TestWebsitesService_List_NoRetryOn400(t *testing.T) {
	t.Run("does not retry on 400", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: "Bad request"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.List(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed with status 400")
		assert.Nil(t, result)
	})
}

func TestWebsitesService_List_RetryOn502(t *testing.T) {
	t.Run("retries on 502 bad gateway", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedItem := client.WebsiteItem{
			Id:         1,
			Domain:     "example.com",
			Status:     "active",
			TargetHash: "QmTest",
			TargetType: "ipfs",
		}

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadGateway},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &client.WebsiteItemResponse{Data: expectedItem},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.List(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 1)
	})
}

func TestWebsitesService_NoRetryOn401(t *testing.T) {
	t.Run("does not retry on 401 unauthorized", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
				JSON401:      &client.ErrorResponse{Error: "Unauthorized"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.List(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		assert.Nil(t, result)
	})
}

func TestWebsitesService_Create_Success(t *testing.T) {
	t.Run("creates website successfully", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:         1,
			Domain:     "example.com",
			TargetHash: "QmTest",
			TargetType: "ipfs",
		}

		mockClient.EXPECT().
			PostApiWebsitesWithResponse(mock.Anything, mock.Anything).
			Return(&client.PostApiWebsitesResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      expectedWebsite,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Create(context.Background(), "example.com", "QmTest", "ipfs")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
	})
}

func TestWebsitesService_Delete_Success(t *testing.T) {
	t.Run("deletes website by ID", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			DeleteApiWebsitesIdWithResponse(mock.Anything, "1").
			Return(&client.DeleteApiWebsitesIdResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.Delete(context.Background(), "1")

		require.NoError(t, err)
	})
}
