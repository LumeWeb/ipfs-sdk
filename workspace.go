package ipfs

import (
	"context"
	"net/http"
	"strconv"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Type aliases for Workspaces types from generated client
type WorkspaceResponse = internalclient.WorkspaceResponse
type WorkspaceRequest = internalclient.WorkspaceRequest
type WorkspaceListResponseResponse = internalclient.WorkspaceListResponseResponse
type WorkspaceListResponse = internalclient.WorkspaceListResponseResponse
type WorkspaceAccessResponse = internalclient.WorkspaceAccessResponse
type WorkspaceResolveResponse = internalclient.WorkspaceResolveResponse
type WorkspaceResolveWebsite = internalclient.WorkspaceResolveWebsite

// WorkspacesConfig holds configuration for Workspaces service operations
type WorkspacesConfig struct {
	Retry  RetryConfig
	Client WorkspacesClientWithResponsesInterface
}

// DefaultWorkspacesConfig returns default configuration for Workspaces service
func DefaultWorkspacesConfig() WorkspacesConfig {
	return WorkspacesConfig{
		Retry: DefaultRetryConfig(),
	}
}

// WorkspacesOption applies configuration to WorkspacesConfig
type WorkspacesOption func(*WorkspacesConfig)

// WithWorkspacesRetry sets the retry configuration for Workspaces operations
func WithWorkspacesRetry(cfg RetryConfig) WorkspacesOption {
	return func(c *WorkspacesConfig) {
		c.Retry = cfg
	}
}

// WithWorkspacesClient sets the client interface for Workspaces operations
// This allows overriding the internal generated client for testing or mocking
func WithWorkspacesClient(client WorkspacesClientWithResponsesInterface) WorkspacesOption {
	return func(c *WorkspacesConfig) {
		c.Client = client
	}
}

// WorkspacesService provides workspace management functionality
type WorkspacesService interface {
	// List retrieves workspaces owned by the authenticated user, including
	// unattached workspaces, narrowed server-side by the provided list options.
	// Without options the server applies its default list window, so pagination
	// options are needed to page beyond it.
	List(ctx context.Context, opts ...ListWorkspacesOption) ([]WorkspaceResponse, error)
	// ListWithTotal behaves like List but returns the server window's total
	// count alongside the data, letting callers detect further pages.
	ListWithTotal(ctx context.Context, opts ...ListWorkspacesOption) (*WorkspaceListResponse, error)
	// Get retrieves a specific workspace by ID.
	Get(ctx context.Context, id string) (*WorkspaceResponse, error)
	// Create creates a workspace. Pass a non-nil website_id in req to attach it
	// to a website the user owns (the publish link); a nil website_id is omitted,
	// creating an unattached workspace that needs no Website record or domain.
	Create(ctx context.Context, req WorkspaceRequest) (*WorkspaceResponse, error)
	// Delete deletes a workspace and returns its final state. The workspace is
	// marked deleting, its portal API key revoked, and the backing application
	// removed; provider 404 is treated as already deleted.
	Delete(ctx context.Context, id string) (*WorkspaceResponse, error)
	// Access returns the owner's proxy Basic Auth credentials for a workspace.
	// Set rotate to rotate the proxy credential before returning. Only the proxy
	// credential is returned; the portal API key and database password are never
	// exposed. With rotation enabled the request is made exactly once so a failed
	// response can never rotate the credential a second time.
	Access(ctx context.Context, id string, rotate bool) (*WorkspaceAccessResponse, error)
	// Attach attaches a workspace to a website the user owns (the publish link).
	Attach(ctx context.Context, id string, websiteID int) (*WorkspaceResponse, error)
	// Resume resumes a suspended workspace.
	Resume(ctx context.Context, id string) (*WorkspaceResponse, error)
	// Suspend suspends a workspace.
	Suspend(ctx context.Context, id string) (*WorkspaceResponse, error)
	// Resolve lets a runtime container resolve its own workspace identity from
	// the Coolify-injected resource UUID plus its workspace PORTAL_API_KEY. The
	// resource UUID may be empty, in which case the query parameter is omitted.
	Resolve(ctx context.Context, resourceUUID string) (*WorkspaceResolveResponse, error)
}

type workspacesService struct {
	client WorkspacesClientWithResponsesInterface
	config WorkspacesConfig
}

// NewWorkspacesService creates a Workspaces service from a client interface with options
func NewWorkspacesService(genClient WorkspacesClientWithResponsesInterface, opts ...WorkspacesOption) WorkspacesService {
	cfg := DefaultWorkspacesConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &workspacesService{client: genClient, config: cfg}
}

// ListWorkspacesOption configures the server-side pagination of the workspaces
// list. The portal /api/workspaces endpoint is a queryutil list endpoint, so
// the window is sent as query params (_start/_end) via a request editor.
type ListWorkspacesOption func(*workspaceListFilters)

type workspaceListFilters struct {
	start int
	limit int
}

// WithWorkspacesStart sets the 0-based starting index of the workspace list
// window (_start). Use together with WithWorkspacesLimit to page beyond the
// server's default window.
func WithWorkspacesStart(start int) ListWorkspacesOption {
	return func(f *workspaceListFilters) {
		f.start = start
	}
}

// WithWorkspacesLimit sets the number of workspaces to return per page; the
// underlying exclusive _end index is derived as start+limit.
func WithWorkspacesLimit(limit int) ListWorkspacesOption {
	return func(f *workspaceListFilters) {
		f.limit = limit
	}
}

// buildListWorkspacesEditor returns a request editor that appends the queryutil
// pagination params for a workspace list call. Calls with no options yield a
// nil editor (no query mutations).
func buildListWorkspacesEditor(f workspaceListFilters) internalclient.RequestEditorFn {
	if f.start == 0 && f.limit == 0 {
		return nil
	}
	return func(_ context.Context, req *http.Request) error {
		q := req.URL.Query()
		if f.limit != 0 {
			q.Set("_end", strconv.Itoa(f.start+f.limit))
		}
		if f.start != 0 {
			q.Set("_start", strconv.Itoa(f.start))
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

// List retrieves workspaces owned by the authenticated user, including
// unattached workspaces, narrowed server-side by the provided list options.
func (s *workspacesService) List(ctx context.Context, opts ...ListWorkspacesOption) ([]WorkspaceResponse, error) {
	resp, err := s.listWorkspaces(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if resp == nil || resp.JSON200 == nil {
		return []WorkspaceResponse{}, nil
	}

	return resp.JSON200.Data, nil
}

// ListWithTotal behaves like List but returns the server window's total count
// alongside the data, letting callers detect further pages.
func (s *workspacesService) ListWithTotal(ctx context.Context, opts ...ListWorkspacesOption) (*WorkspaceListResponse, error) {
	resp, err := s.listWorkspaces(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if resp == nil || resp.JSON200 == nil {
		return &WorkspaceListResponse{Data: []WorkspaceResponse{}}, nil
	}

	return resp.JSON200, nil
}

// listWorkspaces fetches one workspace list page with the supplied options.
func (s *workspacesService) listWorkspaces(ctx context.Context, opts ...ListWorkspacesOption) (*internalclient.GetApiWorkspacesResponse, error) {
	var filters workspaceListFilters
	for _, opt := range opts {
		opt(&filters)
	}

	var result *internalclient.GetApiWorkspacesResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.listPage(ctx, buildListWorkspacesEditor(filters))
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListWorkspaces, []int{http.StatusOK}); err != nil {
			return err
		}

		result = resp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// listPage fetches one page of the workspace list. Calls with no editor keep
// the no-editor call so existing callers and test mocks are unaffected.
func (s *workspacesService) listPage(ctx context.Context, reqEditor internalclient.RequestEditorFn) (*internalclient.GetApiWorkspacesResponse, error) {
	if reqEditor == nil {
		return s.client.GetApiWorkspacesWithResponse(ctx)
	}
	return s.client.GetApiWorkspacesWithResponse(ctx, reqEditor)
}

// Get retrieves a specific workspace by ID.
func (s *workspacesService) Get(ctx context.Context, id string) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWorkspacesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetWorkspace) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Create creates a workspace. A non-nil website_id attaches the workspace to
// a website the user owns (the publish link); a nil website_id is omitted,
// creating an unattached workspace that needs no Website record or domain.
func (s *workspacesService) Create(ctx context.Context, req WorkspaceRequest) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWorkspacesWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateWorkspace, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			return ErrBadRequest(opsString(OpCreateWorkspace) + " no response data")
		}

		result = resp.JSON201
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete deletes a workspace and returns its final state. The workspace is
// marked deleting, its portal API key revoked, and the backing application
// removed; provider 404 is treated as already deleted.
func (s *workspacesService) Delete(ctx context.Context, id string) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiWorkspacesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDeleteWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpDeleteWorkspace) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Access returns the owner's proxy Basic Auth credentials for a workspace.
// Set rotate to rotate the proxy credential before returning.
func (s *workspacesService) Access(ctx context.Context, id string, rotate bool) (*WorkspaceAccessResponse, error) {
	var result *WorkspaceAccessResponse

	// Rotation is not idempotent: a retry after a successful rotation could
	// invalidate the credential already handed back. Restrict rotate requests
	// to a single attempt.
	retryCfg := s.config.Retry
	if rotate {
		retryCfg.Attempts = 1
	}

	err := httputil.RetryContext(ctx, retryCfg, func() error {
		params := &internalclient.GetApiWorkspacesIdAccessParams{}
		if rotate {
			params.Rotate = &rotate
		}

		resp, err := s.client.GetApiWorkspacesIdAccessWithResponse(ctx, id, params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetWorkspaceAccess, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetWorkspaceAccess) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Attach attaches a workspace to a website the user owns (the publish link).
func (s *workspacesService) Attach(ctx context.Context, id string, websiteID int) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWorkspacesIdAttachWithResponse(ctx, id, WorkspaceRequest{WebsiteId: &websiteID})
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpAttachWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpAttachWorkspace) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Resume resumes a suspended workspace.
func (s *workspacesService) Resume(ctx context.Context, id string) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWorkspacesIdResumeWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpResumeWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpResumeWorkspace) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Suspend suspends a workspace.
func (s *workspacesService) Suspend(ctx context.Context, id string) (*WorkspaceResponse, error) {
	var result *WorkspaceResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWorkspacesIdSuspendWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpSuspendWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpSuspendWorkspace) + " no response data for workspace " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Resolve lets a runtime container resolve its own workspace identity from
// the Coolify-injected resource UUID plus its workspace PORTAL_API_KEY. The
// resource UUID may be empty, in which case the query parameter is omitted.
// Authentication via the workspace API key is handled by the client's token.
func (s *workspacesService) Resolve(ctx context.Context, resourceUUID string) (*WorkspaceResolveResponse, error) {
	var result *WorkspaceResolveResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		params := &internalclient.GetApiWorkspacesResolveParams{}
		if resourceUUID != "" {
			params.ResourceUuid = &resourceUUID
		}

		resp, err := s.client.GetApiWorkspacesResolveWithResponse(ctx, params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpResolveWorkspace, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpResolveWorkspace) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
