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
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
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

func TestWebsitesService_ValidateDNS_Success(t *testing.T) {
	t.Run("validates DNS configuration successfully", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedResponse := &client.WebsiteValidateResponse{
			Valid: true,
		}

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "1").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.ValidateDNS(context.Background(), "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResponse.Valid, result.Valid)
	})
}

func TestWebsitesService_ValidateDNS_NotFound(t *testing.T) {
	t.Run("returns error when website not found during DNS validation", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "999").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
				JSON404:      &client.ErrorResponse{Error: "website not found"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.ValidateDNS(context.Background(), "999")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "website not found")
		assert.Nil(t, result)
	})
}

func TestWebsitesService_ValidateDNS_Unauthorized(t *testing.T) {
	t.Run("returns error when unauthorized during DNS validation", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "1").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
				JSON401:      &client.ErrorResponse{Error: "unauthorized"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.ValidateDNS(context.Background(), "1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
		assert.Nil(t, result)
	})
}

func TestWebsitesService_UpdateSSLStatusInternal_Success(t *testing.T) {
	t.Run("updates SSL status via internal API successfully", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		sslStatus := client.SSLStatusUpdateRequest{
			Status: "valid",
		}

		mockClient.EXPECT().
			PostInternalWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com", mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, domain string, body client.SSLStatusUpdateRequest, reqEditors ...client.RequestEditorFn) (*client.PostInternalWebsitesDomainSslStatusResponse, error) {
				return &client.PostInternalWebsitesDomainSslStatusResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", "secret123", sslStatus)

		require.NoError(t, err)
	})
}

func TestWebsitesService_UpdateSSLStatusInternal_BadRequest(t *testing.T) {
	t.Run("returns error when SSL status data is invalid", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		sslStatus := client.SSLStatusUpdateRequest{
			Status: "",
		}

		mockClient.EXPECT().
			PostInternalWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com", mock.Anything, mock.Anything).
			Return(&client.PostInternalWebsitesDomainSslStatusResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: "invalid SSL status data"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", "secret123", sslStatus)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SSL status data")
	})

	t.Run("returns error when website not found during SSL status update", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		sslStatus := client.SSLStatusUpdateRequest{
			Status: "valid",
		}

		mockClient.EXPECT().
			PostInternalWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com", mock.Anything, mock.Anything).
			Return(&client.PostInternalWebsitesDomainSslStatusResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
				JSON404:      &client.ErrorResponse{Error: "website not found"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", "secret123", sslStatus)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "website not found")
	})
}

func TestWebsitesService_UpdateSSLStatusInternal_Unauthorized(t *testing.T) {
	t.Run("returns error when unauthorized during SSL status update", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		sslStatus := client.SSLStatusUpdateRequest{
			Status: "valid",
		}

		mockClient.EXPECT().
			PostInternalWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com", mock.Anything, mock.Anything).
			Return(&client.PostInternalWebsitesDomainSslStatusResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
				JSON401:      &client.ErrorResponse{Error: "unauthorized"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", "wrong-secret", sslStatus)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

func TestWebsitesService_WaitForSSLStatusReady_Success(t *testing.T) {
	t.Run("polls until SSL status reaches ready", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		callCount := 0

		mockClient.EXPECT().
			GetApiWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, domain string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesDomainSslStatusResponse, error) {
				callCount++
				if callCount == 1 {
					// First call: pending
					return &client.GetApiWebsitesDomainSslStatusResponse{
						Body:         []byte("{}"),
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200: &client.WebsiteResponse{
							Domain: "example.com",
							Ssl: &client.SSLStatusInfo{
								Status: "pending",
							},
						},
					}, nil
				}
				// Second call: ready
				return &client.GetApiWebsitesDomainSslStatusResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.WebsiteResponse{
						Domain: "example.com",
						Ssl: &client.SSLStatusInfo{
							Status: "ready",
						},
					},
				}, nil
			}).
			Times(2)

		service := NewWebsitesService(mockClient)
		domain, err := service.WaitForSSLStatusReady(context.Background(), "example.com")

		require.NoError(t, err)
		assert.Equal(t, "ready", domain)
		assert.Equal(t, 2, callCount)
	})

	t.Run("polls until SSL status reaches failed", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		callCount := 0

		mockClient.EXPECT().
			GetApiWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, domain string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesDomainSslStatusResponse, error) {
				callCount++
				if callCount == 1 {
					return &client.GetApiWebsitesDomainSslStatusResponse{
						Body:         []byte("{}"),
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200: &client.WebsiteResponse{
							Domain: "example.com",
							Ssl: &client.SSLStatusInfo{
								Status: "pending",
							},
						},
					}, nil
				}
				return &client.GetApiWebsitesDomainSslStatusResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.WebsiteResponse{
						Domain: "example.com",
						Ssl: &client.SSLStatusInfo{
							Status: "failed",
							Error:  strPtr("certificate validation failed"),
						},
					},
				}, nil
			}).
			Times(2)

		service := NewWebsitesService(mockClient)
		domain, err := service.WaitForSSLStatusReady(context.Background(), "example.com")

		require.NoError(t, err)
		assert.Equal(t, "failed", domain)
		assert.Equal(t, 2, callCount)
	})
}

func TestWebsitesService_WaitForSSLStatusReady_Timeout(t *testing.T) {
	t.Run("times out when status never settles", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesDomainSslStatusWithResponse(mock.Anything, "example.com").
			RunAndReturn(func(ctx context.Context, domain string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesDomainSslStatusResponse, error) {
				// Always return pending
				return &client.GetApiWebsitesDomainSslStatusResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.WebsiteResponse{
						Domain: "example.com",
						Ssl: &client.SSLStatusInfo{
							Status: "pending",
						},
					},
				}, nil
			}).Maybe()

		service := NewWebsitesService(mockClient)
		_, err := service.WaitForSSLStatusReady(context.Background(), "example.com",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "poll timed out")
	})
}

func TestWebsitesService_WaitForWebsiteStatus_Success(t *testing.T) {
	t.Run("polls until website status reaches expected state", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		callCount := 0

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			RunAndReturn(func(ctx context.Context, id string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesIdResponse, error) {
				callCount++
				if callCount == 1 {
					return &client.GetApiWebsitesIdResponse{
						Body:         []byte("{}"),
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200: &client.WebsiteResponse{
							Id:     123,
							Status: "deploying",
						},
					}, nil
				}
				return &client.GetApiWebsitesIdResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.WebsiteResponse{
						Id:     123,
						Status: "active",
					},
				}, nil
			}).
			Times(2)

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "123", "active")

		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})
}

func TestWebsitesService_WaitForWebsiteStatus_Timeout(t *testing.T) {
	t.Run("times out when status never reaches expected state", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			RunAndReturn(func(ctx context.Context, id string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesIdResponse, error) {
				return &client.GetApiWebsitesIdResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &client.WebsiteResponse{
						Id:     123,
						Status: "deploying",
					},
				}, nil
			})

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "123", "active",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "poll timed out")
	})
}

func TestWebsitesService_WaitForWebsiteStatus_ErrorOnResponse(t *testing.T) {
	t.Run("returns error when API response fails", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			RunAndReturn(func(ctx context.Context, id string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesIdResponse, error) {
				return nil, assert.AnError
			})

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "123", "active",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(100*time.Millisecond))

		require.Error(t, err)
	})
}

func TestWebsitesService_WaitForDNSValidation(t *testing.T) {
	t.Run("succeeds when DNS becomes valid", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		service := NewWebsitesService(mockClient)

		expectedResponse1 := client.WebsiteValidateResponse{
			Valid:   false,
			Message: "DNS not yet propagated",
		}
		expectedResponse2 := client.WebsiteValidateResponse{
			Valid:   false,
			Message: "Still waiting",
		}
		expectedResponse3 := client.WebsiteValidateResponse{
			Valid:   true,
			Message: "DNS validated successfully",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "123").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &expectedResponse1,
			}, nil).
			Once()

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "123").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &expectedResponse2,
			}, nil).
			Once()

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "123").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &expectedResponse3,
			}, nil).
			Once()

		err := service.WaitForDNSValidation(context.Background(), "123",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(200*time.Millisecond))

		require.NoError(t, err)
	})

	t.Run("times out when DNS never becomes valid", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		service := NewWebsitesService(mockClient)

		timeoutResponse := client.WebsiteValidateResponse{
			Valid:   false,
			Message: "Timeout",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "123").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte("{}"),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &timeoutResponse,
			}, nil).
			Maybe().
			Times(0)

		err := service.WaitForDNSValidation(context.Background(), "123",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(50*time.Millisecond))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

func strPtr(s string) *string {
	return &s
}
