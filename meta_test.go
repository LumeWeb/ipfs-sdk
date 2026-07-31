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

func testMetaService(t *testing.T) (MetaService, *mocks.MockMetaClientWithResponsesInterface) {
	mockClient := mocks.NewMockMetaClientWithResponsesInterface(t)
	retries := 1
	service := NewMetaService(mockClient, WithMetaRetry(httputil.RetryConfig{Attempts: uint(retries)}))
	return service, mockClient
}

func TestNewMetaService(t *testing.T) {
	mockClient := mocks.NewMockMetaClientWithResponsesInterface(t)
	service := NewMetaService(mockClient)

	assert.NotNil(t, service)
	assert.Implements(t, (*MetaService)(nil), service)
}

func TestMetaService_ExportDAG(t *testing.T) {
	t.Run("returns DAG export on success", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		expectedResp := &internalclient.DAGExportResponse{
			RootCid:        "bafyroothash",
			TotalBlocks:    3,
			TotalSizeBytes: 4096,
			Blocks: []internalclient.DAGBlock{
				{
					Cid:  "bafyblock1",
					Size: 1024,
					Links: []internalclient.DAGLink{
						{Cid: "bafylink1", Index: 0},
					},
				},
			},
		}

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyroothash").
			Return(&internalclient.GetApiExportCidCidDagResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.ExportDAG(context.Background(), "bafyroothash")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "bafyroothash", result.RootCid)
		assert.Equal(t, uint64(3), result.TotalBlocks)
		assert.Equal(t, uint64(4096), result.TotalSizeBytes)
		assert.Len(t, result.Blocks, 1)
		assert.Equal(t, "bafyblock1", result.Blocks[0].Cid)
		assert.Equal(t, uint64(1024), result.Blocks[0].Size)
		assert.Len(t, result.Blocks[0].Links, 1)
		assert.Equal(t, "bafylink1", result.Blocks[0].Links[0].Cid)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyinvalid").
			Return(nil, assert.AnError).
			Once()

		result, err := service.ExportDAG(context.Background(), "bafyinvalid")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyempty").
			Return(&internalclient.GetApiExportCidCidDagResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ExportDAG(context.Background(), "bafyempty")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestMetaService_ExportSiaObject(t *testing.T) {
	t.Run("returns CID export on success", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		expectedResp := &internalclient.CIDExportResponse{
			Cid:       "bafycia",
			SizeBytes: 2048,
			CreatedAt: "2024-01-01T00:00:00Z",
			UpdatedAt: "2024-01-02T00:00:00Z",
		}

		mockClient.EXPECT().
			GetApiExportCidCidSiaObjectWithResponse(mock.Anything, "bafycia").
			Return(&internalclient.GetApiExportCidCidSiaObjectResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.ExportSiaObject(context.Background(), "bafycia")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "bafycia", result.Cid)
		assert.Equal(t, uint64(2048), result.SizeBytes)
		assert.Equal(t, "2024-01-01T00:00:00Z", result.CreatedAt)
		assert.Equal(t, "2024-01-02T00:00:00Z", result.UpdatedAt)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		mockClient.EXPECT().
			GetApiExportCidCidSiaObjectWithResponse(mock.Anything, "bafyinvalid").
			Return(nil, assert.AnError).
			Once()

		result, err := service.ExportSiaObject(context.Background(), "bafyinvalid")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testMetaService(t)

		mockClient.EXPECT().
			GetApiExportCidCidSiaObjectWithResponse(mock.Anything, "bafyempty").
			Return(&internalclient.GetApiExportCidCidSiaObjectResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.ExportSiaObject(context.Background(), "bafyempty")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}
