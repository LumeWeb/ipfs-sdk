package ipfs

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/internal/testutil"
)

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
			testutil.VerifyAuthorization(t, r, "test-token")

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(internalclient.WebsiteItemResponse{Data: expectedItem}).
				Write(t, w)
		},
	})
	defer server.Close()

	client := NewClient(server.URL, "test-token")
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

	client := NewClient(server.URL, "invalid-token")
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

	client := NewClient(server.URL, "test-token")
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

	client := NewClient(server.URL, "test-token")
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

	client := NewClient(server.URL, "test-token")
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

	client := NewClient(server.URL, "test-token")
	err := client.Websites().Delete(context.Background(), "1")

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

	client := NewClient(server.URL, "test-token")
	result, err := client.Websites().GetSSLStatus(context.Background(), "example.com")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWebsite.Id, result.Id)
}
