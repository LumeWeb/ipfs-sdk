package ipfs

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/internal/testutil"
)

func TestWorkspacesClient_List_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:        1,
		Label:     "dev",
		Domain:    "dev.workspaces.example.com",
		Status:    "running",
		WebsiteId: intPtr(42),
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(internalclient.WorkspaceListResponseResponse{Data: []internalclient.WorkspaceResponse{expectedWorkspace}, Total: 1}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().List(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, expectedWorkspace.Id, result[0].Id)
	require.Equal(t, expectedWorkspace.Label, result[0].Label)
	require.Equal(t, expectedWorkspace.Domain, result[0].Domain)
}

func TestWorkspacesClient_List_Pagination(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces")
			testutil.VerifyAuthorization(t, r, getTestToken())
			require.Equal(t, "5", r.URL.Query().Get("_start"))
			require.Equal(t, "10", r.URL.Query().Get("_end"))

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(internalclient.WorkspaceListResponseResponse{Data: []internalclient.WorkspaceResponse{}, Total: 0}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().List(context.Background(),
		WithWorkspacesStart(5), WithWorkspacesLimit(5))

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestWorkspacesClient_ListWithTotal_Success(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(internalclient.WorkspaceListResponseResponse{
					Data:  []internalclient.WorkspaceResponse{{Id: 7, Label: "one"}},
					Total: 2,
				}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().ListWithTotal(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Data, 1)
	require.Equal(t, 7, result.Data[0].Id)
	require.Equal(t, 2, result.Total)
}

func TestWorkspacesClient_List_Unauthorized(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.NewJSONResponse().
				WithStatus(http.StatusUnauthorized).
				WithBody(internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "unauthorized"}}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, "invalid-value")
	require.NoError(t, err)

	result, err := client.Workspaces().List(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
	require.Nil(t, result)
}

func TestWorkspacesClient_Get_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "running",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces/1")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Get(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWorkspace.Id, result.Id)
	require.Equal(t, expectedWorkspace.Domain, result.Domain)
}

func TestWorkspacesClient_Create_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "creating",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/workspaces")
			testutil.VerifyAuthorization(t, r, getTestToken())

			var req internalclient.WorkspaceRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.NotNil(t, req.WebsiteId)
			require.Equal(t, 42, *req.WebsiteId)

			testutil.NewJSONResponse().
				WithStatus(http.StatusCreated).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Create(context.Background(), WorkspaceRequest{WebsiteId: intPtr(42)})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWorkspace.Id, result.Id)
	require.Equal(t, expectedWorkspace.Domain, result.Domain)
}

func TestWorkspacesClient_Delete_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "deleting",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodDelete)
			testutil.VerifyPath(t, r, "/api/workspaces/1")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Delete(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWorkspace.Status, result.Status)
}

func TestWorkspacesClient_Access_Success(t *testing.T) {
	expectedAccess := internalclient.WorkspaceAccessResponse{
		Username: "proxy-user",
		Password: "proxy-pass",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces/1/access")
			testutil.VerifyAuthorization(t, r, getTestToken())
			require.Equal(t, "true", r.URL.Query().Get("rotate"))

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedAccess).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Access(context.Background(), "1", true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedAccess.Username, result.Username)
	require.Equal(t, expectedAccess.Password, result.Password)
}

func TestWorkspacesClient_Access_RotateRotatesOnce(t *testing.T) {
	// Rotation is not idempotent: a retry after a 5xx must never rotate the
	// proxy credential a second time.
	var attempts int

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			attempts++
			testutil.NewJSONResponse().
				WithStatus(http.StatusInternalServerError).
				WithBody(internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "boom"}}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	_, err = client.Workspaces().Access(context.Background(), "1", true)

	require.Error(t, err)
	require.Equal(t, 1, attempts, "rotate must not be retried")
}

func TestWorkspacesClient_Attach_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:        1,
		Label:     "dev",
		Domain:    "dev.workspaces.example.com",
		Status:    "running",
		WebsiteId: intPtr(99),
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/workspaces/1/attach")
			testutil.VerifyAuthorization(t, r, getTestToken())

			var req internalclient.WorkspaceRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.NotNil(t, req.WebsiteId)
			require.Equal(t, 99, *req.WebsiteId)

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Attach(context.Background(), "1", 99)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, intPtr(99), result.WebsiteId)
}

func TestWorkspacesClient_Create_Unattached_OmitsWebsiteID(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/workspaces")
			testutil.VerifyAuthorization(t, r, getTestToken())

			var req internalclient.WorkspaceRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Nil(t, req.WebsiteId)

			testutil.NewJSONResponse().
				WithStatus(http.StatusCreated).
				WithBody(internalclient.WorkspaceResponse{
					Id:     1,
					Label:  "dev",
					Domain: "dev.workspaces.example.com",
					Status: "creating",
				}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Create(context.Background(), WorkspaceRequest{})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Nil(t, result.WebsiteId)
}

func TestWorkspacesClient_Suspend_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "suspended",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/workspaces/1/suspend")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Suspend(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWorkspace.Status, result.Status)
}

func TestWorkspacesClient_Resume_Success(t *testing.T) {
	expectedWorkspace := internalclient.WorkspaceResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "running",
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodPost)
			testutil.VerifyPath(t, r, "/api/workspaces/1/resume")
			testutil.VerifyAuthorization(t, r, getTestToken())

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedWorkspace).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Resume(context.Background(), "1")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedWorkspace.Status, result.Status)
}

func TestWorkspacesClient_Resolve_Success(t *testing.T) {
	expectedResolve := internalclient.WorkspaceResolveResponse{
		Id:     1,
		Label:  "dev",
		Domain: "dev.workspaces.example.com",
		Status: "running",
		Website: &internalclient.WorkspaceResolveWebsite{
			Id:         42,
			Status:     "active",
			TargetHash: "QmTest",
			TargetType: "ipfs",
		},
	}

	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.VerifyMethod(t, r, http.MethodGet)
			testutil.VerifyPath(t, r, "/api/workspaces/resolve")
			testutil.VerifyAuthorization(t, r, getTestToken())
			require.Equal(t, "uuid-1234", r.URL.Query().Get("resource_uuid"))

			testutil.NewJSONResponse().
				WithStatus(http.StatusOK).
				WithBody(expectedResolve).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Resolve(context.Background(), "uuid-1234")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, expectedResolve.Id, result.Id)
	require.NotNil(t, result.Website)
	require.Equal(t, expectedResolve.Website.TargetHash, result.Website.TargetHash)
}

func TestWorkspacesClient_Conflict(t *testing.T) {
	server := testutil.NewTestServer(t, testutil.HTTPTestServerConfig{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			testutil.NewJSONResponse().
				WithStatus(http.StatusConflict).
				WithBody(internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "already deleting"}}).
				Write(t, w)
		},
	})
	defer server.Close()

	client, err := NewClient(server.URL, getTestToken())
	require.NoError(t, err)
	result, err := client.Workspaces().Delete(context.Background(), "1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "already deleting")
	require.Nil(t, result)
}

func intPtr(i int) *int {
	return &i
}
