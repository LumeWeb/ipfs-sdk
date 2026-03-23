package ipfs

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/internal/testutil"
)

// getTestToken returns a test token from environment or falls back to test value
func getTestToken() string {
	if token := os.Getenv("TEST_API_TOKEN"); token != "" {
		return token
	}
	return "test-token" // fallback for local testing
}

func TestWebsitesClient_List_Success(t *testing.T) {
	expectedItem := internalclient.WebsiteItem{
		Id:         1,
		Domain:     "example.com",
		Status:     "active",
		TargetHash: "QmXxx",
		TargetType: "ipfs",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/websites")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(internalclient.WebsiteItemResponse{Data: []internalclient.WebsiteItem{expectedItem}}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().List(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, expectedItem.Id, result[0].Id)
	require.Equal(t, expectedItem.Domain, result[0].Domain)
}

func TestWebsitesClient_List_Unauthorized(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.NewJSONResponse().
				WithStatus(http.StatusUnauthorized).
				WithBody(internalclient.ErrorResponse{Error: "unauthorized"}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, "invalid-token")
	require.NoError(t, err, "NewClient should succeed with any URL")
	
	result, err := client.Websites().List(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
	require.Nil(t, result)
}

func TestWebsitesClient_Get_Success(t *testing.T) {
	expectedWebsite := internalclient.WebsiteResponse{
		Id:         1,
		Domain:     "example.com",
		TargetHash: "QmTest",
		TargetType: "ipfs",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/websites/1")

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWebsite).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().Get(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWebsite.Id, result.Id)
	require.Equal(t, expectedWebsite.Domain, result.Domain)
}

func TestWebsitesClient_Create_Success(t *testing.T) {
	expectedWebsite := internalclient.WebsiteResponse{
		Id:         1,
		Domain:     "example.com",
		TargetHash: "QmTest",
		TargetType: "ipfs",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/websites")

			testutil.NewJSONResponse().
				WithStatus(http.StatusCreated).
				WithBody(expectedWebsite).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().Create(context.Background(), "example.com", "QmTest", "ipfs")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWebsite.Id, result.Id)
}

func TestWebsitesClient_Update_Success(t *testing.T) {
	expectedWebsite := internalclient.WebsiteResponse{
		Id:         1,
		Domain:     "example.com",
		TargetHash: "QmUpdated",
		TargetType: "ipfs",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPut)
			testutil.VerifyPath(t, r, "/api/websites/1")

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWebsite).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().Update(context.Background(), "1", "example.com", "QmUpdated", "ipfs")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "QmUpdated", result.TargetHash)
}

func TestWebsitesClient_Delete_Success(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodDelete)
			testutil.VerifyPath(t, r, "/api/websites/1")

			w.WriteHeader(http.StatusOK)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	err = client.Websites().Delete(context.Background(), "1")

	require.NoError(t, err)
}

func TestWebsitesClient_GetSSLStatus_Success(t *testing.T) {
	expectedWebsite := internalclient.WebsiteResponse{
		Id:         1,
		Domain:     "example.com",
		TargetHash: "QmTest",
		TargetType: "ipfs",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/websites/example.com/ssl-status")

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWebsite).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().GetSSLStatus(context.Background(), "example.com")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWebsite.Id, result.Id)
}

func TestWebsitesClient_ValidateDNS_Success(t *testing.T) {
	expectedResponse := internalclient.WebsiteValidateResponse{
		Valid: true,
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/websites/1/validate")

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedResponse).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().ValidateDNS(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedResponse.Valid, result.Valid)
}

func TestWebsitesClient_ValidateDNS_NotFound(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/websites/999/validate")

			testutil.NewJSONResponse().
				WithStatus(http.StatusNotFound).
				WithBody(internalclient.ErrorResponse{Error: "website not found"}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)
	result, err := client.Websites().ValidateDNS(context.Background(), "999")

	require.Error(t, err)
	require.Contains(t, err.Error(), "website not found")
	require.Nil(t, result)
}

func TestWebsitesClient_UpdateSSLStatusInternal_Success(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/internal/websites/example.com/ssl-status")
			testutil.VerifySecret(t, r, "secret123")

			w.WriteHeader(http.StatusOK)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)

	sslStatus := internalclient.SSLStatusUpdateRequest{
		Status: "valid",
	}
	err = client.Websites().UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

	require.NoError(t, err)
}

func TestWebsitesClient_UpdateSSLStatusInternal_SuccessWithNoContent(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/internal/websites/example.com/ssl-status")
			testutil.VerifySecret(t, r, "secret456")

			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret456"))
	require.NoError(t, err)

	sslStatus := internalclient.SSLStatusUpdateRequest{
		Status: "valid",
		Error:  nil,
	}
	err = client.Websites().UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

	require.NoError(t, err)
}

func TestWebsitesClient_UpdateSSLStatusInternal_Unauthorized(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/internal/websites/example.com/ssl-status")

			testutil.NewJSONResponse().
				WithStatus(http.StatusUnauthorized).
				WithBody(internalclient.ErrorResponse{Error: "unauthorized"}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)

	sslStatus := internalclient.SSLStatusUpdateRequest{
		Status: "valid",
	}
	err = client.Websites().UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

func TestWebsitesClient_UpdateSSLStatusInternal_BadRequest(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/internal/websites/example.com/ssl-status")

			testutil.NewJSONResponse().
				WithStatus(http.StatusBadRequest).
				WithBody(internalclient.ErrorResponse{Error: "invalid SSL status data"}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)

	sslStatus := internalclient.SSLStatusUpdateRequest{
		Status: "",
	}
	err = client.Websites().UpdateSSLStatusInternal(context.Background(), "example.com", sslStatus)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid SSL status data")
}

func TestWebsitesClient_UpdateSSLStatusInternal_NotFound(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/internal/websites/nonexistent.example.com/ssl-status")

			testutil.NewJSONResponse().
				WithStatus(http.StatusNotFound).
				WithBody(internalclient.ErrorResponse{Error: "website not found"}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)

	sslStatus := internalclient.SSLStatusUpdateRequest{
		Status: "valid",
	}
	err = client.Websites().UpdateSSLStatusInternal(context.Background(), "nonexistent.example.com", sslStatus)

	require.Error(t, err)
	require.Contains(t, err.Error(), "website not found")
}

func TestWebsitesClient_GetGatewayWebsite_Success(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/internal/websites/example.com")
			testutil.VerifySecret(t, r, "secret123")

			testutil.NewJSONResponse().
				WithBody(internalclient.GatewayWebsiteResponse{
					Domain:     "example.com",
					TargetType: "ipfs",
					TargetHash: "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
					Status:     "active",
				}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret123"))
	require.NoError(t, err)

	website, err := client.Websites().GetGatewayWebsite(context.Background(), "example.com")

	require.NoError(t, err)
	require.Equal(t, "example.com", website.Domain)
	require.Equal(t, "ipfs", website.TargetType)
	require.Equal(t, "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", website.TargetHash)
	require.Equal(t, "active", website.Status)
}


func TestWebsitesClient_GetGatewayWebsiteStatus_Success(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/internal/websites/example.com/status")
			testutil.VerifySecret(t, r, "secret456")

			testutil.NewJSONResponse().
				WithBody(internalclient.GatewayWebsiteStatusResponse{
					Domain:   "example.com",
					Status:   "active",
					IsBroken: false,
				}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken(), WithGatewaySecret("secret456"))
	require.NoError(t, err)

	status, err := client.Websites().GetGatewayWebsiteStatus(context.Background(), "example.com")

	require.NoError(t, err)
	require.Equal(t, "example.com", status.Domain)
	require.Equal(t, "active", status.Status)
	require.Equal(t, false, status.IsBroken)
}

