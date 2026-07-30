package ipfs

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/mocks"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

func testDAGService(t *testing.T) (DAGService, *mocks.MockDAGClientWithResponsesInterface) {
	mockClient := mocks.NewMockDAGClientWithResponsesInterface(t)
	retries := 1
	service := NewDAGService(mockClient, WithDAGRetry(httputil.RetryConfig{Attempts: uint(retries)}))
	return service, mockClient
}

func TestNewDAGService(t *testing.T) {
	client := setupInternalClient()
	if client == nil {
		t.Skip("Internal client setup failed")
	}

	service := NewDAGService(ConvertClientToDAG(client))

	assert.NotNil(t, service)
	assert.Implements(t, (*DAGService)(nil), service)
}

func TestNewDAGServiceNilClient(t *testing.T) {
	service := NewDAGService(nil)

	assert.NotNil(t, service)
}

func TestDAGService_ResolveDAG(t *testing.T) {
	t.Run("returns DAG response on success", func(t *testing.T) {
		service, mockClient := testDAGService(t)

		expectedResp := &internalclient.DAGResponse{
			RootCid: "bafybeieffnocaq7t4w4daagvydl32igft5oziyyaebqr6vx6rb3fwh2ab4",
			Nodes: []internalclient.DAGBlockNodeResponse{
				{
					Cid:      "bafybeieffnocaq7t4w4daagvydl32igft5oziyyaebqr6vx6rb3fwh2ab4",
					Size:     1024,
					Children: []string{"bafybeihf7s3zfgfzhbv5k4wihbnbgqchild"},
					},
			},
		}

		mockClient.EXPECT().
			GetApiDagCidWithResponse(mock.Anything, "bafybeieffnocaq7t4w4daagvydl32igft5oziyyaebqr6vx6rb3fwh2ab4").
			Return(&internalclient.GetApiDagCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.ResolveDAG(context.Background(), "bafybeieffnocaq7t4w4daagvydl32igft5oziyyaebqr6vx6rb3fwh2ab4")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "bafybeieffnocaq7t4w4daagvydl32igft5oziyyaebqr6vx6rb3fwh2ab4", result.RootCid)
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, 1024, result.Nodes[0].Size)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDAGService(t)

		mockClient.EXPECT().
			GetApiDagCidWithResponse(mock.Anything, "bafybeiinvalid").
			Return(nil, assert.AnError).
			Once()

		result, err := service.ResolveDAG(context.Background(), "bafybeiinvalid")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testDAGService(t)

		mockClient.EXPECT().
			GetApiDagCidWithResponse(mock.Anything, "bafybeiempty").
			Return(&internalclient.GetApiDagCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ResolveDAG(context.Background(), "bafybeiempty")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on not found", func(t *testing.T) {
		service, mockClient := testDAGService(t)

		mockClient.EXPECT().
			GetApiDagCidWithResponse(mock.Anything, "bafyreinotfound").
			Return(&internalclient.GetApiDagCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
				JSON404:      &internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "CID not found"}},
			}, nil).
			Once()

		result, err := service.ResolveDAG(context.Background(), "bafyreinotfound")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on unauthorized", func(t *testing.T) {
		service, mockClient := testDAGService(t)

		mockClient.EXPECT().
			GetApiDagCidWithResponse(mock.Anything, "bafybeiunauth").
			Return(&internalclient.GetApiDagCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusUnauthorized},
			}, nil).
			Once()

		result, err := service.ResolveDAG(context.Background(), "bafybeiunauth")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
