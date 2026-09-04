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
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
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
					JSON500:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Internal server error"}},
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
					JSON200:      &client.WebsiteItemResponse{Data: []client.WebsiteItem{expectedItem}},
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
				JSON400:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Bad request"}},
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
				JSON200:      &client.WebsiteItemResponse{Data: []client.WebsiteItem{expectedItem}},
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
				JSON401:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "Unauthorized"}},
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

func TestWebsitesService_CreateWithOptions_DisableDNSHosting(t *testing.T) {
	t.Run("creates website with dns_hosting_enabled=false", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:                1,
			Domain:            "example.com",
			TargetHash:        "QmTest",
			TargetType:        "ipfs",
			DnsHostingEnabled: false,
		}

		var capturedReq client.WebsiteRequest
		mockClient.EXPECT().
			PostApiWebsitesWithResponse(mock.Anything, mock.AnythingOfType("client.WebsiteRequest")).
			RunAndReturn(func(ctx context.Context, req client.WebsiteRequest, reqEditors ...client.RequestEditorFn) (*client.PostApiWebsitesResponse, error) {
				capturedReq = req
				return &client.PostApiWebsitesResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
					JSON201:      expectedWebsite,
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.CreateWithOptions(context.Background(), client.WebsiteRequest{
			Domain:            new("example.com"),
			TargetHash:        "QmTest",
			TargetType:        "ipfs",
			DnsHostingEnabled: new(bool),
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.False(t, *capturedReq.DnsHostingEnabled, "dns_hosting_enabled should be false")
	})
}

func TestWebsitesService_CreateWithOptions_EnableDNSHosting(t *testing.T) {
	t.Run("creates website with dns_hosting_enabled=true", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:                1,
			Domain:            "example.com",
			TargetHash:        "QmTest",
			TargetType:        "ipns",
			DnsHostingEnabled: true,
		}

		var capturedReq client.WebsiteRequest
		mockClient.EXPECT().
			PostApiWebsitesWithResponse(mock.Anything, mock.AnythingOfType("client.WebsiteRequest")).
			RunAndReturn(func(ctx context.Context, req client.WebsiteRequest, reqEditors ...client.RequestEditorFn) (*client.PostApiWebsitesResponse, error) {
				capturedReq = req
				return &client.PostApiWebsitesResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
					JSON201:      expectedWebsite,
				}, nil
			}).
			Once()

		dnsHostingTrue := true
		service := NewWebsitesService(mockClient)
		result, err := service.CreateWithOptions(context.Background(), client.WebsiteRequest{
			Domain:            new("example.com"),
			TargetHash:        "QmTest",
			TargetType:        "ipfs",
			DnsHostingEnabled: &dnsHostingTrue,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.True(t, *capturedReq.DnsHostingEnabled, "dns_hosting_enabled should be true")
	})
}

func TestWebsitesService_Create_CallsCreateWithOptions(t *testing.T) {
	t.Run("Create() calls CreateWithOptions() with default dns_hosting_enabled", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:         1,
			Domain:     "example.com",
			TargetHash: "QmTest",
			TargetType: "ipfs",
		}

		var capturedReq client.WebsiteRequest
		mockClient.EXPECT().
			PostApiWebsitesWithResponse(mock.Anything, mock.AnythingOfType("client.WebsiteRequest")).
			RunAndReturn(func(ctx context.Context, req client.WebsiteRequest, reqEditors ...client.RequestEditorFn) (*client.PostApiWebsitesResponse, error) {
				capturedReq = req
				return &client.PostApiWebsitesResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
					JSON201:      expectedWebsite,
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Create(context.Background(), "example.com", "QmTest", "ipfs")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.Equal(t, "example.com", *capturedReq.Domain)
		assert.Equal(t, "QmTest", capturedReq.TargetHash)
		assert.Equal(t, "ipfs", capturedReq.TargetType)
		// DnsHostingEnabled should be nil when not specified
		assert.Nil(t, capturedReq.DnsHostingEnabled, "default dns_hosting_enabled should be nil")
	})
}

func TestWebsitesService_CreateWithOptions_HTTPError(t *testing.T) {
	t.Run("returns error on HTTP error", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PostApiWebsitesWithResponse(mock.Anything, mock.AnythingOfType("client.WebsiteRequest")).
			Return(nil, assert.AnError).
			Times(3) // Retry will attempt 3 times before failing

		service := NewWebsitesService(mockClient)
		result, err := service.CreateWithOptions(context.Background(), client.WebsiteRequest{
			Domain:     new("example.com"),
			TargetHash: "QmTest",
			TargetType: "ipfs",
		})

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_ListPlatformDomains(t *testing.T) {
	t.Run("returns platform domains when endpoint responds 200", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expected := &client.PlatformDomainListResponse{
			Data: []client.PlatformDomainResponse{
				{Domain: "acme.lume.io", Enabled: true, Id: 1, Namespace: "acme", ZoneId: 42},
			},
			Total: 1,
		}

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.ListPlatformDomains(context.Background())

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, "acme.lume.io", result.Data[0].Domain)
		assert.True(t, result.Data[0].Enabled)
		assert.Equal(t, 1, result.Total)
	})

	t.Run("returns error on unauthorized response", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsResponse{
				Body:         []byte(`{"error":{"reason":"authentication required"}}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
				JSON401:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "authentication required"}},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.ListPlatformDomains(context.Background())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "authentication required")
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ListPlatformDomains(context.Background())

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_UpdateWithOptions_DisableDNSHosting(t *testing.T) {
	t.Run("updates website with dns_hosting_enabled=false", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:                1,
			Domain:            "example.com",
			TargetHash:        "QmTest",
			TargetType:        "ipfs",
			DnsHostingEnabled: false,
		}

		var capturedReq client.WebsiteUpdateRequest
		mockClient.EXPECT().
			PutApiWebsitesIdWithResponse(mock.Anything, "1", mock.AnythingOfType("client.WebsiteUpdateRequest")).
			RunAndReturn(func(ctx context.Context, id string, req client.WebsiteUpdateRequest, reqEditors ...client.RequestEditorFn) (*client.PutApiWebsitesIdResponse, error) {
				capturedReq = req
				return &client.PutApiWebsitesIdResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200:      expectedWebsite,
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.UpdateWithOptions(context.Background(), "1", client.WebsiteUpdateRequest{
			DnsHostingEnabled: new(false),
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.False(t, *capturedReq.DnsHostingEnabled, "dns_hosting_enabled should be false")
	})
}

func TestWebsitesService_UpdateWithOptions_EnableDNSHosting(t *testing.T) {
	t.Run("updates website with dns_hosting_enabled=true", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:                1,
			Domain:            "example.com",
			TargetHash:        "QmTest",
			TargetType:        "ipns",
			DnsHostingEnabled: true,
		}

		var capturedReq client.WebsiteUpdateRequest
		mockClient.EXPECT().
			PutApiWebsitesIdWithResponse(mock.Anything, "1", mock.AnythingOfType("client.WebsiteUpdateRequest")).
			RunAndReturn(func(ctx context.Context, id string, req client.WebsiteUpdateRequest, reqEditors ...client.RequestEditorFn) (*client.PutApiWebsitesIdResponse, error) {
				capturedReq = req
				return &client.PutApiWebsitesIdResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200:      expectedWebsite,
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.UpdateWithOptions(context.Background(), "1", client.WebsiteUpdateRequest{
			DnsHostingEnabled: new(true),
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.True(t, *capturedReq.DnsHostingEnabled, "dns_hosting_enabled should be true")
	})
}

func TestWebsitesService_Update_CallsUpdateWithOptions(t *testing.T) {
	t.Run("Update() calls UpdateWithOptions() with pointer fields", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:         1,
			Domain:     "example.com",
			TargetHash: "QmTest",
			TargetType: "ipfs",
		}

		var capturedReq client.WebsiteUpdateRequest
		mockClient.EXPECT().
			PutApiWebsitesIdWithResponse(mock.Anything, "1", mock.AnythingOfType("client.WebsiteUpdateRequest")).
			RunAndReturn(func(ctx context.Context, id string, req client.WebsiteUpdateRequest, reqEditors ...client.RequestEditorFn) (*client.PutApiWebsitesIdResponse, error) {
				capturedReq = req
				return &client.PutApiWebsitesIdResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200:      expectedWebsite,
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Update(context.Background(), "1", "example.com", "QmTest", "ipfs")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.Equal(t, "example.com", *capturedReq.Domain)
		assert.Equal(t, "QmTest", *capturedReq.TargetHash)
		assert.Equal(t, "ipfs", *capturedReq.TargetType)
		assert.Nil(t, capturedReq.DnsHostingEnabled, "default dns_hosting_enabled should be nil")
	})
}

func TestWebsitesService_UpdateWithOptions_HTTPError(t *testing.T) {
	t.Run("returns error on HTTP error", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			PutApiWebsitesIdWithResponse(mock.Anything, "1", mock.AnythingOfType("client.WebsiteUpdateRequest")).
			Return(nil, assert.AnError).
			Times(3)

		service := NewWebsitesService(mockClient)
		result, err := service.UpdateWithOptions(context.Background(), "1", client.WebsiteUpdateRequest{
			Domain: new("example.com"),
		})

		require.Error(t, err)
		assert.Nil(t, result)
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
			Valid:  true,
			Reason: string(WebsiteValidationReasonValidated),
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
				JSON404:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "website not found"}},
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
				JSON401:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "unauthorized"}},
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
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

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
				JSON400:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "invalid SSL status data"}},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

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
				JSON404:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "website not found"}},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

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
				JSON401:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "unauthorized"}},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

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
							Error:  new("certificate validation failed"),
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
		assert.ErrorIs(t, err, context.DeadlineExceeded)
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
		assert.ErrorIs(t, err, context.DeadlineExceeded)
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

func TestWebsitesService_WaitForWebsiteStatus_BrokenVia410(t *testing.T) {
	t.Run("detects broken status from 410 Gone response", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"id":123,"status":"broken"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
				JSON410: &client.WebsiteResponse{
					Id:     123,
					Status: "broken",
				},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "123", "broken")

		require.NoError(t, err)
	})

	t.Run("detects deleted status from 410 Gone response", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "456").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"id":456,"status":"deleted"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
				JSON410: &client.WebsiteResponse{
					Id:     456,
					Status: "deleted",
				},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "456", "deleted")

		require.NoError(t, err)
	})

	t.Run("410 with nil result still propagates error", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "789").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"error":"gone"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
			}, nil)

		service := NewWebsitesService(mockClient)
		err := service.WaitForWebsiteStatus(context.Background(), "789", "active",
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
			Reason:  string(WebsiteValidationReasonDNSMissing),
		}
		expectedResponse2 := client.WebsiteValidateResponse{
			Valid:   false,
			Message: "Still waiting",
			Reason:  string(WebsiteValidationReasonDNSMissing),
		}
		expectedResponse3 := client.WebsiteValidateResponse{
			Valid:   true,
			Message: "DNS validated successfully",
			Reason:  string(WebsiteValidationReasonValidated),
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
			Reason:  string(WebsiteValidationReasonDNSMissing),
		}

		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "123").
			RunAndReturn(func(ctx context.Context, id string, reqEditors ...client.RequestEditorFn) (*client.PostApiWebsitesIdValidateResponse, error) {
				return &client.PostApiWebsitesIdValidateResponse{
					Body:         []byte("{}"),
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200:      &timeoutResponse,
				}, nil
			})

		err := service.WaitForDNSValidation(context.Background(), "123",
			httputil.WithPollInterval(10*time.Millisecond),
			httputil.WithPollTimeout(50*time.Millisecond))

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func boolPtr(b bool) *bool {
	return &b
}

func TestSentinels_ErrNotFound(t *testing.T) {
	t.Run("GetWebsite 404 wraps ErrNotFound", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "999").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"error":"website not found"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.Get(context.Background(), "999")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound), "error should wrap ErrNotFound")
	})

	t.Run("ValidateDNS 404 wraps ErrNotFound", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			PostApiWebsitesIdValidateWithResponse(mock.Anything, "999").
			Return(&client.PostApiWebsitesIdValidateResponse{
				Body:         []byte(`{"error":"website not found"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.ValidateDNS(context.Background(), "999")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound), "error should wrap ErrNotFound")
	})
}

func TestWebsitesService_Get_410_ReturnsBrokenWebsiteAndErrGone(t *testing.T) {
	t.Run("returns broken website data and ErrGone on 410 response", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		brokenWebsite := &client.WebsiteResponse{
			Id:         123,
			Domain:     "broken.example.com",
			Status:     "broken",
			TargetHash: "QmBroken",
			TargetType: "ipfs",
		}

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"id":123,"domain":"broken.example.com","status":"broken"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
				JSON410:      brokenWebsite,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Get(context.Background(), "123")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGone), "error should wrap ErrGone")
		assert.NotNil(t, result, "should return broken website data from JSON410")
		assert.Equal(t, 123, result.Id)
		assert.Equal(t, "broken", result.Status)
	})

	t.Run("410 does not retry", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		callCount := 0

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "456").
			RunAndReturn(func(ctx context.Context, id string, reqEditors ...client.RequestEditorFn) (*client.GetApiWebsitesIdResponse, error) {
				callCount++
				return &client.GetApiWebsitesIdResponse{
					Body:         []byte(`{"error":"website is broken"}`),
					HTTPResponse: &http.Response{StatusCode: http.StatusGone},
				}, nil
			}).
			Once()

		service := NewWebsitesService(mockClient)
		_, err := service.Get(context.Background(), "456")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGone), "error should wrap ErrGone")
		assert.Equal(t, 1, callCount, "410 should not trigger retry")
	})
}

func TestWebsitesService_Get_Success(t *testing.T) {
	t.Run("returns website data on 200 response", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expectedWebsite := &client.WebsiteResponse{
			Id:         123,
			Domain:     "example.com",
			Status:     "active",
			TargetHash: "QmActive",
			TargetType: "ipfs",
		}

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"id":123,"domain":"example.com","status":"active"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedWebsite,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Get(context.Background(), "123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedWebsite.Id, result.Id)
		assert.Equal(t, "active", result.Status)
	})
}

func TestWebsitesService_Get_NotFound(t *testing.T) {
	t.Run("returns ErrNotFound on 404 and nil result", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "999").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"error":"website not found"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
				JSON404:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "website not found"}},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.Get(context.Background(), "999")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound), "error should wrap ErrNotFound")
		assert.Nil(t, result)
	})
}

func TestSentinels_ErrGone(t *testing.T) {
	t.Run("Get 410 wraps ErrGone", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesIdWithResponse(mock.Anything, "123").
			Return(&client.GetApiWebsitesIdResponse{
				Body:         []byte(`{"error":"website is broken"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.Get(context.Background(), "123")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGone), "error should wrap ErrGone")
	})

	t.Run("GetGatewayWebsite 410 wraps ErrGone", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetInternalWebsitesDomainWithResponse(mock.Anything, "broken.example.com").
			Return(&client.GetInternalWebsitesDomainResponse{
				Body:         []byte(`{"error":"website is broken or deleted"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.GetGatewayWebsite(context.Background(), "broken.example.com")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGone), "error should wrap ErrGone")
	})

	t.Run("GetGatewayWebsiteStatus 410 wraps ErrGone", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetInternalWebsitesDomainStatusWithResponse(mock.Anything, "broken.example.com").
			Return(&client.GetInternalWebsitesDomainStatusResponse{
				Body:         []byte(`{"error":"website is broken or deleted"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusGone},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.GetGatewayWebsiteStatus(context.Background(), "broken.example.com")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrGone), "error should wrap ErrGone")
	})

	t.Run("GetGatewayWebsite 404 wraps ErrNotFound not ErrGone", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetInternalWebsitesDomainWithResponse(mock.Anything, "missing.example.com").
			Return(&client.GetInternalWebsitesDomainResponse{
				Body:         []byte(`{"error":"website not found"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.GetGatewayWebsite(context.Background(), "missing.example.com")

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound), "404 should wrap ErrNotFound")
		assert.False(t, errors.Is(err, ErrGone), "404 should not wrap ErrGone")
	})
}

func TestSentinels_ErrUnauthorized(t *testing.T) {
	t.Run("List 401 wraps ErrUnauthorized", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		mockClient.EXPECT().
			GetApiWebsitesWithResponse(mock.Anything).
			Return(&client.GetApiWebsitesResponse{
				Body:         []byte(`{"error":"unauthorized"}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
			}, nil).Once()

		service := NewWebsitesService(mockClient)
		_, err := service.List(context.Background())

		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrUnauthorized), "error should wrap ErrUnauthorized")
	})
}

func testWebsitesDomainService(t *testing.T) (WebsitesService, *mocks.MockWebsitesClientWithResponsesInterface) {
	mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
	retries := 1
	service := NewWebsitesService(mockClient, WithWebsitesRetry(RetryConfig{Attempts: uint(retries)}))
	return service, mockClient
}

func TestWebsitesService_ListDomains(t *testing.T) {
	t.Run("returns domain list on success", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedDomains := []client.DomainResponse{
			{
				Id:        1,
				Domain:    "example.com",
				Namespace: "icann",
				ZoneName:  new("example.com."),
			},
			{
				Id:        2,
				Domain:    "example.hns",
				Namespace: "hns",
			},
		}

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsWithResponse(mock.Anything, "1").
			Return(&client.GetApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &client.DomainListResponse{
					Data:  expectedDomains,
					Total: 2,
				},
			}, nil).
			Once()

		result, err := service.ListDomains(context.Background(), "1")

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, 1, result[0].Id)
		assert.Equal(t, "example.com", result[0].Domain)
		assert.Equal(t, "icann", string(result[0].Namespace))
		assert.NotNil(t, result[0].ZoneName)
		assert.Equal(t, "example.com.", *result[0].ZoneName)
		assert.Equal(t, 2, result[1].Id)
		assert.Equal(t, "example.hns", result[1].Domain)
		assert.Equal(t, "hns", string(result[1].Namespace))
	})

	t.Run("returns empty list when no domains bound", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsWithResponse(mock.Anything, "1").
			Return(&client.GetApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &client.DomainListResponse{
					Data:  []client.DomainResponse{},
					Total: 0,
				},
			}, nil).
			Once()

		result, err := service.ListDomains(context.Background(), "1")

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns empty list when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsWithResponse(mock.Anything, "1").
			Return(&client.GetApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ListDomains(context.Background(), "1")

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsWithResponse(mock.Anything, "1").
			Return(nil, assert.AnError).
			Once()

		result, err := service.ListDomains(context.Background(), "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_BindDomain(t *testing.T) {
	t.Run("binds domain successfully", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.DomainResponse{
			Id:        1,
			Domain:    "example.com",
			Namespace: "icann",
		}

		domainReq := client.DomainRequest{
			Domain:    "example.com",
			Namespace: "icann",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsWithResponse(mock.Anything, "1", mock.AnythingOfType("client.DomainRequest")).
			Return(&client.PostApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      expectedResponse,
			}, nil).
			Once()

		result, err := service.BindDomain(context.Background(), "1", DomainRequest(domainReq))

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResponse.Id, result.Id)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, "icann", string(result.Namespace))
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		domainReq := client.DomainRequest{
			Domain:    "example.com",
			Namespace: "icann",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsWithResponse(mock.Anything, "1", mock.AnythingOfType("client.DomainRequest")).
			Return(nil, assert.AnError).
			Once()

		result, err := service.BindDomain(context.Background(), "1", DomainRequest(domainReq))

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON201 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		domainReq := client.DomainRequest{
			Domain:    "example.com",
			Namespace: "icann",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsWithResponse(mock.Anything, "1", mock.AnythingOfType("client.DomainRequest")).
			Return(&client.PostApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      nil,
			}, nil).
			Once()

		result, err := service.BindDomain(context.Background(), "1", DomainRequest(domainReq))

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_UpdateDomain(t *testing.T) {
	t.Run("updates domain successfully", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.DomainResponse{
			Id:                1,
			Domain:            "example.com",
			Namespace:         "icann",
			DnsHostingEnabled: true,
		}

		updateReq := client.DomainUpdateRequest{
			DnsHostingEnabled: boolPtr(true),
		}

		mockClient.EXPECT().
			PatchApiWebsitesIdDomainsDomainIdWithResponse(mock.Anything, "1", "1", mock.AnythingOfType("client.DomainUpdateRequest")).
			Return(&client.PatchApiWebsitesIdDomainsDomainIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.UpdateDomain(context.Background(), "1", "1", DomainUpdateRequest(updateReq))

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResponse.Id, result.Id)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, true, result.DnsHostingEnabled)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		updateReq := client.DomainUpdateRequest{
			DnsHostingEnabled: boolPtr(true),
		}

		mockClient.EXPECT().
			PatchApiWebsitesIdDomainsDomainIdWithResponse(mock.Anything, "1", "1", mock.AnythingOfType("client.DomainUpdateRequest")).
			Return(nil, assert.AnError).
			Once()

		result, err := service.UpdateDomain(context.Background(), "1", "1", DomainUpdateRequest(updateReq))

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		updateReq := client.DomainUpdateRequest{
			DnsHostingEnabled: boolPtr(true),
		}

		mockClient.EXPECT().
			PatchApiWebsitesIdDomainsDomainIdWithResponse(mock.Anything, "1", "1", mock.AnythingOfType("client.DomainUpdateRequest")).
			Return(&client.PatchApiWebsitesIdDomainsDomainIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.UpdateDomain(context.Background(), "1", "1", DomainUpdateRequest(updateReq))

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_UnbindDomain(t *testing.T) {
	t.Run("unbinds domain successfully", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			DeleteApiWebsitesIdDomainsDomainIdWithResponse(mock.Anything, "1", "1").
			Return(&client.DeleteApiWebsitesIdDomainsDomainIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNoContent},
			}, nil).
			Once()

		err := service.UnbindDomain(context.Background(), "1", "1")

		require.NoError(t, err)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			DeleteApiWebsitesIdDomainsDomainIdWithResponse(mock.Anything, "1", "1").
			Return(nil, assert.AnError).
			Once()

		err := service.UnbindDomain(context.Background(), "1", "1")

		require.Error(t, err)
	})
}

func TestWebsitesService_VerifyDomain(t *testing.T) {
	t.Run("verifies domain successfully", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.DomainResponse{
			Id:        1,
			Domain:    "example.com",
			Namespace: "icann",
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdVerifyResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.VerifyDomain(context.Background(), "1", "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResponse.Id, result.Id)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, "icann", string(result.Namespace))
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(mock.Anything, "1", "1").
			Return(nil, assert.AnError).
			Once()

		result, err := service.VerifyDomain(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdVerifyResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.VerifyDomain(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_GetDomainDNSRequirements(t *testing.T) {
	t.Run("returns delegation records", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		ds := "lumeweb. 3600 IN DS 12345 13 2 <digest>"
		expectedResponse := &client.DomainResponse{
			Id:          1,
			Domain:      "lumeweb",
			Namespace:   "hns",
			Status:      new(client.DomainResponseStatusRecordsGenerated),
			ZoneName:    new("lumeweb."),
			GatewayHost: new("gateway.lumeweb.com"),
			Delegation: &client.DNSDelegation{
				Mode:        new("delegated"),
				Dnssec:      new("enabled"),
				DnssecError: nil,
				ParentRecords: &[]client.DNSDelegationRecord{
					{Type: "NS", Value: new("ns1.lumeweb,ns2.lumeweb")},
					{Type: "DS", Value: new(ds)},
				},
				AuthoritativeRecords: &[]client.DNSDelegationRecord{
					{Type: "NS", Value: new("ns1.lumeweb\nns2.lumeweb")},
					{Type: "TLSA", Value: new("_443._tcp.lumeweb. 3 1 1 <sha256>")},
				},
			},
		}

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(mock.Anything, "1", "1").
			Return(&client.GetApiWebsitesIdDomainsDomainIdDnsRequirementsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.GetDomainDNSRequirements(context.Background(), "1", "1")

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Delegation)
		assert.Equal(t, "hns", string(result.Namespace))
		assert.Equal(t, new("gateway.lumeweb.com"), result.GatewayHost)
		assert.Equal(t, new("delegated"), result.Delegation.Mode)
		assert.Equal(t, new("enabled"), result.Delegation.Dnssec)
		require.NotNil(t, result.Delegation.ParentRecords)
		assert.Len(t, *result.Delegation.ParentRecords, 2)
		assert.Equal(t, "TLSA", (*result.Delegation.AuthoritativeRecords)[1].Type)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(mock.Anything, "1", "1").
			Return(nil, assert.AnError).
			Once()

		result, err := service.GetDomainDNSRequirements(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(mock.Anything, "1", "1").
			Return(&client.GetApiWebsitesIdDomainsDomainIdDnsRequirementsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.GetDomainDNSRequirements(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDomainNamespaceOf(t *testing.T) {
	t.Run("maps known namespaces", func(t *testing.T) {
		assert.Equal(t, DomainNamespaceICANN, DomainNamespaceOf(&DomainResponse{Namespace: "icann"}))
		assert.Equal(t, DomainNamespaceHNS, DomainNamespaceOf(&DomainResponse{Namespace: "hns"}))
	})

	t.Run("casts unknown namespaces and empty for nil", func(t *testing.T) {
		assert.Equal(t, DomainNamespace("ens"), DomainNamespaceOf(&DomainResponse{Namespace: "ens"}))
		assert.Equal(t, DomainNamespace(""), DomainNamespaceOf(nil))
	})
}

func TestWebsiteChangeEventTypeEnum(t *testing.T) {
	assert.Equal(t, WebsiteChangeEventType("published"), WebsiteChangeEventPublished)
	assert.Equal(t, WebsiteChangeEventType("removed"), WebsiteChangeEventRemoved)
}

func TestGatewayNamespaceEnum(t *testing.T) {
	assert.Equal(t, GatewayNamespace("icann"), GatewayNamespaceICANN)
	assert.Equal(t, GatewayNamespace("hns"), GatewayNamespaceHNS)
}

func TestWebsiteChangeEventUsesExportedEnum(t *testing.T) {
	ev := WebsiteChangeEvent{
		EventType: WebsiteChangeEventPublished,
	}
	assert.Equal(t, WebsiteChangeEventPublished, ev.EventType)
	assert.Equal(t, "published", string(ev.EventType))
}

func TestWebsitesService_RepublishDANE(t *testing.T) {
	t.Run("returns republish result", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.DomainDANERepublishResponse{
			Id:         1,
			Domain:     "lumeweb",
			Namespace:  "hns",
			TlsaRdata:  new("3 1 1 <sha256>"),
			OwnerName:  new("_443._tcp.lumeweb"),
			TlsaRecord: new("_443._tcp.lumeweb. 3600 IN TLSA 3 1 1 <sha256>"),
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdDaneRepublishResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.RepublishDANE(context.Background(), "1", "1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "hns", string(result.Namespace))
		assert.Equal(t, new("_443._tcp.lumeweb. 3600 IN TLSA 3 1 1 <sha256>"), result.TlsaRecord)
		assert.Equal(t, new("_443._tcp.lumeweb"), result.OwnerName)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(mock.Anything, "1", "1").
			Return(nil, assert.AnError).
			Once()

		result, err := service.RepublishDANE(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdDaneRepublishResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.RepublishDANE(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_ConvertDomainToOnChain(t *testing.T) {
	t.Run("converts domain to on-chain managed", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.DomainResponse{
			Id:        1,
			Domain:    "lumeweb",
			Namespace: "hns",
			ZoneName:  new("lumeweb."),
		}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdOnchainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.ConvertDomainToOnChain(context.Background(), "1", "1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedResponse.Id, result.Id)
		assert.Equal(t, "lumeweb", result.Domain)
		assert.Equal(t, "hns", string(result.Namespace))
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(mock.Anything, "1", "1").
			Return(nil, assert.AnError).
			Once()

		result, err := service.ConvertDomainToOnChain(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdOnchainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ConvertDomainToOnChain(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on 422 eligibility rejection", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdOnchainResponse{
				Body:         []byte(`{"error":{"reason":"domain is not yet on-chain managed"}}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnprocessableEntity},
				JSON422:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "InvalidRequest", Details: new("Invalid request parameter: domain is not yet on-chain managed")}},
			}, nil).
			Once()

		result, err := service.ConvertDomainToOnChain(context.Background(), "1", "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("treats 422 already-on-chain as success after retry", func(t *testing.T) {
		// A lost 5xx response after a committed conversion makes the retry come
		// back as 422 "already on-chain". That is the desired end state, so it
		// must not surface as an error, and the caller must receive the current
		// domain state rather than a nil result.
		service, mockClient := testWebsitesDomainService(t)

		currentState := &client.DomainResponse{Id: 1, Domain: "lumeweb", Namespace: "hns"}

		mockClient.EXPECT().
			PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(mock.Anything, "1", "1").
			Return(&client.PostApiWebsitesIdDomainsDomainIdOnchainResponse{
				Body:         []byte(`{"error":{"reason":"InvalidRequest","details":"Invalid request parameter: domain is already on-chain managed"}}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnprocessableEntity},
				JSON422:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "InvalidRequest", Details: new("Invalid request parameter: domain is already on-chain managed")}},
			}, nil).
			Once()

		mockClient.EXPECT().
			GetApiWebsitesIdDomainsWithResponse(mock.Anything, "1").
			Return(&client.GetApiWebsitesIdDomainsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &client.DomainListResponse{Data: []client.DomainResponse{*currentState}},
			}, nil).
			Once()

		result, err := service.ConvertDomainToOnChain(context.Background(), "1", "1")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, currentState.Id, result.Id)
		assert.Equal(t, currentState.Domain, result.Domain)
		assert.Equal(t, currentState.Namespace, result.Namespace)
	})
}

func TestWebsitesService_ReconcileWebsiteChanges(t *testing.T) {
	t.Run("returns changes after cursor", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		expectedResponse := &client.WebsiteChangesResponse{
			Events: []client.WebsiteChangeEvent{
				{Id: 1, EventType: "published", Domain: "lumeweb", Cid: new("QmTest")},
				{Id: 2, EventType: "removed", Domain: "example.com"},
			},
			HighWaterMark: 2,
			Truncated:     false,
		}

		mockClient.EXPECT().
			GetInternalWebsitesChangesWithResponse(mock.Anything, mock.AnythingOfType("*client.GetInternalWebsitesChangesParams")).
			Run(func(_ context.Context, params *client.GetInternalWebsitesChangesParams, _ ...client.RequestEditorFn) {
				require.NotNil(t, params.After)
				assert.Equal(t, "5", *params.After)
			}).
			Return(&client.GetInternalWebsitesChangesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResponse,
			}, nil).
			Once()

		result, err := service.ReconcileWebsiteChanges(context.Background(), "5")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Events, 2)
		assert.Equal(t, 2, result.HighWaterMark)
		assert.Equal(t, "lumeweb", result.Events[0].Domain)
	})

	t.Run("omits after param when cursor is empty", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetInternalWebsitesChangesWithResponse(mock.Anything, mock.AnythingOfType("*client.GetInternalWebsitesChangesParams")).
			Run(func(_ context.Context, params *client.GetInternalWebsitesChangesParams, _ ...client.RequestEditorFn) {
				assert.Nil(t, params.After)
			}).
			Return(&client.GetInternalWebsitesChangesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &client.WebsiteChangesResponse{},
			}, nil).
			Once()

		result, err := service.ReconcileWebsiteChanges(context.Background(), "")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetInternalWebsitesChangesWithResponse(mock.Anything, mock.AnythingOfType("*client.GetInternalWebsitesChangesParams")).
			Return(nil, assert.AnError).
			Once()

		result, err := service.ReconcileWebsiteChanges(context.Background(), "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetInternalWebsitesChangesWithResponse(mock.Anything, mock.AnythingOfType("*client.GetInternalWebsitesChangesParams")).
			Return(&client.GetInternalWebsitesChangesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ReconcileWebsiteChanges(context.Background(), "1")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on invalid after cursor", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetInternalWebsitesChangesWithResponse(mock.Anything, mock.AnythingOfType("*client.GetInternalWebsitesChangesParams")).
			Return(&client.GetInternalWebsitesChangesResponse{
				Body:         []byte(`{"error":{"reason":"invalid after cursor"}}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "invalid after cursor"}},
			}, nil).
			Once()

		result, err := service.ReconcileWebsiteChanges(context.Background(), "not-a-number")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWebsitesService_CheckPlatformDomainAvailability_Success(t *testing.T) {
	t.Run("returns availability results when endpoint responds 200", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)
		expected := &client.PlatformAvailabilityResponse{
			Label: "mysite",
			Results: []client.PlatformAvailabilityResult{
				{Available: true, Namespace: "acme", PlatformDomain: "acme.lume.io"},
			},
		}

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsAvailabilityWithResponse(mock.Anything, mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsAvailabilityResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		result, err := service.CheckPlatformDomainAvailability(context.Background(), "mysite")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "mysite", result.Label)
		assert.Len(t, result.Results, 1)
		assert.True(t, result.Results[0].Available)
	})

	t.Run("sends label only when non-empty", func(t *testing.T) {
		mockClient := mocks.NewMockWebsitesClientWithResponsesInterface(t)

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsAvailabilityWithResponse(mock.Anything, mock.Anything).
			Run(func(_ context.Context, params *client.GetApiWebsitesPlatformDomainsAvailabilityParams, _ ...client.RequestEditorFn) {
				require.NotNil(t, params.Label)
				assert.Equal(t, "mysite", *params.Label)
			}).
			Return(&client.GetApiWebsitesPlatformDomainsAvailabilityResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &client.PlatformAvailabilityResponse{Label: "mysite"},
			}, nil).
			Once()

		service := NewWebsitesService(mockClient)
		_, err := service.CheckPlatformDomainAvailability(context.Background(), "mysite")
		require.NoError(t, err)
	})
}

func TestWebsitesService_CheckPlatformDomainAvailability_HTTPError(t *testing.T) {
	t.Run("returns error on unauthorized response", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsAvailabilityWithResponse(mock.Anything, mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsAvailabilityResponse{
				Body:         []byte(`{"error":{"reason":"authentication required"}}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
				JSON401:      &client.ErrorResponse{Error: client.ErrorDetail{Reason: "authentication required"}},
			}, nil).
			Once()

		result, err := service.CheckPlatformDomainAvailability(context.Background(), "mysite")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "authentication required")
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testWebsitesDomainService(t)

		mockClient.EXPECT().
			GetApiWebsitesPlatformDomainsAvailabilityWithResponse(mock.Anything, mock.Anything).
			Return(&client.GetApiWebsitesPlatformDomainsAvailabilityResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.CheckPlatformDomainAvailability(context.Background(), "mysite")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestWebsitesService_ListSendsFilters is the end-to-end guard for server-side
// website list filtering. List with a filter option must emit the corresponding
// queryutil filters[field][op]=value query param (not fetch-then-filter
// client-side), and a plain List must send no filters.
func TestWebsitesService_ListSendsFilters(t *testing.T) {
	var (
		mu       sync.Mutex
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

	genClient, err := client.NewClientWithResponses(srv.URL)
	require.NoError(t, err)
	service := NewWebsitesService(convertWebsitesClient(genClient))

	// With filters: each configured filter must be sent as its queryutil param.
	_, err = service.List(context.Background(),
		WithDomainFilter("example.com"),
		WithStatusFilter("active"),
		WithTargetTypeFilter("ipfs"),
	)
	require.NoError(t, err)
	mu.Lock()
	assert.Equal(t, "example.com", gotQuery.Get("filters[domain][contains]"))
	assert.Equal(t, "active", gotQuery.Get("filters[status][eq]"))
	assert.Equal(t, "ipfs", gotQuery.Get("filters[target_type][eq]"))
	mu.Unlock()

	// With pagination: _start and _end window params must be sent so filtered
	// results are not truncated to the server's default 10-item window.
	_, err = service.List(context.Background(), WithStart(20), WithWebsitesLimit(10))
	require.NoError(t, err)
	mu.Lock()
	assert.Equal(t, "20", gotQuery.Get("_start"))
	assert.Equal(t, "30", gotQuery.Get("_end"))
	mu.Unlock()

	// Without filters or pagination: no query params on the list call.
	_, err = service.List(context.Background())
	require.NoError(t, err)
	mu.Lock()
	assert.Empty(t, gotQuery.Get("filters[domain][contains]"))
	assert.Empty(t, gotQuery.Get("filters[status][eq]"))
	assert.Empty(t, gotQuery.Get("filters[target_type][eq]"))
	assert.Empty(t, gotQuery.Get("_start"))
	assert.Empty(t, gotQuery.Get("_end"))
	mu.Unlock()
}
