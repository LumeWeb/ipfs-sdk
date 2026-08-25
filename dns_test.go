package ipfs

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	"go.lumeweb.com/ipfs-sdk/mocks"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

func setupInternalClient() *internalclient.ClientWithResponses {
	// Create a minimal internal client for testing
	genClient, err := internalclient.NewClientWithResponses("http://example.com")
	if err != nil {
		return nil
	}
	return genClient
}

// testDNSService creates a mock DNS service configured for testing
// Returns service configured with no retries for predictable test behavior
func testDNSService(t *testing.T) (DNSService, *mocks.MockDNSClientWithResponsesInterface) {
	mockClient := mocks.NewMockDNSClientWithResponsesInterface(t)
	// Configure service with 1 attempt (no retries) for predictable tests
	retries := 1
	service := NewDNSService(mockClient, WithDNSRetry(httputil.RetryConfig{Attempts: uint(retries)}))
	return service, mockClient
}

func TestNewDNSService(t *testing.T) {
	client := setupInternalClient()
	if client == nil {
		t.Skip("Internal client setup failed")
	}

	service := NewDNSService(client)

	assert.NotNil(t, service)
	assert.Implements(t, (*DNSService)(nil), service)
}

func TestNewDNSServiceNilClient(t *testing.T) {
	service := NewDNSService(nil)

	assert.NotNil(t, service) // Service should be created even with nil client
}

func TestDNSServiceListZonesContext(t *testing.T) {
	client := setupInternalClient()
	if client == nil {
		t.Skip("Internal client setup failed")
	}

	service := NewDNSService(client)

	_, err := service.ListZones(context.Background())
	assert.Error(t, err) // Expected to fail without actual API
}

func TestDNSService_GetZone(t *testing.T) {
	t.Run("returns zone data on successful API call", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedZone := &internalclient.ZoneResponse{
			Id:         123,
			Domain:     "example.com",
			Status:     "active",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		mockClient.EXPECT().
			GetApiDnsZonesIdWithResponse(mock.Anything, "123").
			Return(&internalclient.GetApiDnsZonesIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedZone,
			}, nil).
			Once()

		zone, err := service.GetZone(context.Background(), "123")

		require.NoError(t, err)
		assert.NotNil(t, zone)
		assert.Equal(t, 123, zone.Id)
		assert.Equal(t, "example.com", zone.Domain)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetApiDnsZonesIdWithResponse(mock.Anything, "123").
			Return(nil, assert.AnError).
			Once()

		zone, err := service.GetZone(context.Background(), "123")

		assert.Error(t, err)
		assert.Nil(t, zone)
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		respBody := []byte(`{"error":"Not found"}`)

		mockClient.EXPECT().
			GetApiDnsZonesIdWithResponse(mock.Anything, "999").
			Return(&internalclient.GetApiDnsZonesIdResponse{
				HTTPResponse: &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       mockReadCloser(respBody),
				},
			}, nil).
			Once()

		zone, err := service.GetZone(context.Background(), "999")

		assert.Error(t, err)
		assert.Nil(t, zone)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetApiDnsZonesIdWithResponse(mock.Anything, "123").
			Return(&internalclient.GetApiDnsZonesIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		zone, err := service.GetZone(context.Background(), "123")

		assert.Error(t, err)
		assert.Nil(t, zone)
	})
}

func TestDNSService_CreateZone(t *testing.T) {
	t.Run("creates zone successfully", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedZone := &internalclient.ZoneResponse{
			Id:         456,
			Domain:     "newdomain.com",
			Status:     "pending",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		mockClient.EXPECT().
			PostApiDnsZonesWithResponse(mock.Anything, mock.Anything).
			Return(&internalclient.PostApiDnsZonesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      expectedZone,
			}, nil).
			Once()

		zone, err := service.CreateZone(context.Background(), "newdomain.com", []string{"ns1.example.com"})

		require.NoError(t, err)
		assert.NotNil(t, zone)
		assert.Equal(t, 456, zone.Id)
		assert.Equal(t, "newdomain.com", zone.Domain)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			PostApiDnsZonesWithResponse(mock.Anything, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		zone, err := service.CreateZone(context.Background(), "newdomain.com", nil)

		assert.Error(t, err)
		assert.Nil(t, zone)
	})

	t.Run("returns error when JSON201 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			PostApiDnsZonesWithResponse(mock.Anything, mock.Anything).
			Return(&internalclient.PostApiDnsZonesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      nil,
			}, nil).
			Once()

		zone, err := service.CreateZone(context.Background(), "newdomain.com", nil)

		assert.Error(t, err)
		assert.Nil(t, zone)
	})
}

func TestDNSService_DeleteZone(t *testing.T) {
	t.Run("deletes zone successfully", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdWithResponse(mock.Anything, "123").
			Return(&internalclient.DeleteApiDnsZonesIdResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNoContent},
			}, nil).
			Once()

		err := service.DeleteZone(context.Background(), "123")

		assert.NoError(t, err)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdWithResponse(mock.Anything, "123").
			Return(nil, assert.AnError).
			Once()

		err := service.DeleteZone(context.Background(), "123")

		assert.Error(t, err)
	})

	t.Run("returns error on non-200/204 status", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		respBody := []byte(`{"error":"Zone not found"}`)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdWithResponse(mock.Anything, "999").
			Return(&internalclient.DeleteApiDnsZonesIdResponse{
				HTTPResponse: &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       mockReadCloser(respBody),
				},
			}, nil).
			Once()

		err := service.DeleteZone(context.Background(), "999")

		assert.Error(t, err)
	})
}

func TestDNSService_ListRecords(t *testing.T) {
	t.Run("lists records for a zone", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		// The API returns one record at a time
		expectedRecord := internalclient.RecordResponse{
			Name:    "www",
			Type:    "A",
			Content: "1.2.3.4",
			Ttl:     3600,
		}

		mockClient.EXPECT().
			GetApiDnsZonesIdRecordsWithResponse(mock.Anything, "123").
			Return(&internalclient.GetApiDnsZonesIdRecordsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &internalclient.RecordResponseResponse{
					Data: []internalclient.RecordResponse{expectedRecord},
					Total: 1,
				},
			}, nil).
			Once()

		records, err := service.ListRecords(context.Background(), "123")

		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, "www", records[0].Name)
		assert.Equal(t, "A", records[0].Type)
		assert.Equal(t, "1.2.3.4", records[0].Content)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetApiDnsZonesIdRecordsWithResponse(mock.Anything, "123").
			Return(nil, assert.AnError).
			Once()

		records, err := service.ListRecords(context.Background(), "123")

		assert.Error(t, err)
		assert.Nil(t, records)
	})
}

func TestDNSService_CreateRecord(t *testing.T) {
	t.Run("creates record successfully", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedRecord := &RecordResponse{
			Name:    "www",
			Type:    "A",
			Content: "1.2.3.4",
		}

		mockClient.EXPECT().
			PostApiDnsZonesIdRecordsWithResponse(mock.Anything, "123", mock.Anything).
			Return(&internalclient.PostApiDnsZonesIdRecordsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      expectedRecord,
			}, nil).
			Once()

		record := RecordRequest{
			Name:    "www",
			Type:    "A",
			Content: "1.2.3.4",
		}

		result, err := service.CreateRecord(context.Background(), "123", record)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "www", result.Name)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			PostApiDnsZonesIdRecordsWithResponse(mock.Anything, "123", mock.Anything).
			Return(nil, assert.AnError).
			Once()

		result, err := service.CreateRecord(context.Background(), "123", RecordRequest{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON201 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			PostApiDnsZonesIdRecordsWithResponse(mock.Anything, "123", mock.Anything).
			Return(&internalclient.PostApiDnsZonesIdRecordsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusCreated},
				JSON201:      nil,
			}, nil).
			Once()

		result, err := service.CreateRecord(context.Background(), "123", RecordRequest{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDNSService_GetRecord(t *testing.T) {
	t.Run("gets specific record", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedRecord := &RecordResponse{
			Name:    "www",
			Type:    "A",
			Content: "1.2.3.4",
		}

		mockClient.EXPECT().
			GetApiDnsZonesIdRecordsNameTypeWithResponse(mock.Anything, "123", "www", "A").
			Return(&internalclient.GetApiDnsZonesIdRecordsNameTypeResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedRecord,
			}, nil).
			Once()

		record, err := service.GetRecord(context.Background(), "123", "www", "A")

		require.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, "www", record.Name)
		assert.Equal(t, "A", record.Type)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetApiDnsZonesIdRecordsNameTypeWithResponse(mock.Anything, "123", "www", "A").
			Return(nil, assert.AnError).
			Once()

		record, err := service.GetRecord(context.Background(), "123", "www", "A")

		assert.Error(t, err)
		assert.Nil(t, record)
	})
}

func TestDNSService_UpdateRecord(t *testing.T) {
	t.Run("updates record successfully", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedRecord := &RecordResponse{
			Name:    "www",
			Type:    "A",
			Content: "4.5.6.7",
		}

		mockClient.EXPECT().
			PutApiDnsZonesIdRecordsNameTypeWithResponse(mock.Anything, "123", "www", "A", mock.Anything).
			Return(&internalclient.PutApiDnsZonesIdRecordsNameTypeResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedRecord,
			}, nil).
			Once()

		record := RecordRequest{
			Name:    "www",
			Type:    "A",
			Content: "4.5.6.7",
		}

		result, err := service.UpdateRecord(context.Background(), "123", "www", "A", record)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "4.5.6.7", result.Content)
	})
}

func TestDNSService_DeleteRecord(t *testing.T) {
	t.Run("deletes record successfully", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdRecordsNameTypeWithBodyWithResponse(mock.Anything, "123", "www", "A", "application/json", mock.Anything).
			Return(&internalclient.DeleteApiDnsZonesIdRecordsNameTypeResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).
			Once()

		err := service.DeleteRecord(context.Background(), "123", "www", "A")

		assert.NoError(t, err)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdRecordsNameTypeWithBodyWithResponse(mock.Anything, "123", "www", "A", "application/json", mock.Anything).
			Return(nil, assert.AnError).
			Once()

		err := service.DeleteRecord(context.Background(), "123", "www", "A")

		assert.Error(t, err)
	})

	t.Run("deletes single record by content", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		var capturedBody internalclient.RecordDeleteRequest
		mockClient.EXPECT().
			DeleteApiDnsZonesIdRecordsNameTypeWithResponse(mock.Anything, "123", "www", "TXT", mock.Anything).
			Run(func(_ context.Context, _, _, _ string, body internalclient.RecordDeleteRequest, _ ...internalclient.RequestEditorFn) {
				capturedBody = body
			}).
			Return(&internalclient.DeleteApiDnsZonesIdRecordsNameTypeResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNoContent},
			}, nil).
			Once()

		err := service.DeleteRecord(context.Background(), "123", "www", "TXT", "v=spf1 include:mxroute.com -all")

		assert.NoError(t, err)
		require.NotNil(t, capturedBody.Content, "content should be present")
		assert.Equal(t, "v=spf1 include:mxroute.com -all", *capturedBody.Content)
	})

	t.Run("no content sends no body (whole RRSet)", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			DeleteApiDnsZonesIdRecordsNameTypeWithBodyWithResponse(mock.Anything, "123", "www", "A", "application/json", mock.Anything).
			Return(&internalclient.DeleteApiDnsZonesIdRecordsNameTypeResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNoContent},
			}, nil).
			Once()

		err := service.DeleteRecord(context.Background(), "123", "www", "A")

		assert.NoError(t, err)
	})

	t.Run("rejects multiple content selectors", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		err := service.DeleteRecord(context.Background(), "123", "www", "TXT", "v=spf1 include:mxroute.com -all", "another-value")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at most one content selector")
		mockClient.AssertNotCalled(t, "DeleteApiDnsZonesIdRecordsNameTypeWithResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		mockClient.AssertNotCalled(t, "DeleteApiDnsZonesIdRecordsNameTypeWithBodyWithResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("rejects empty content selector", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		err := service.DeleteRecord(context.Background(), "123", "www", "TXT", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty content selector")
		mockClient.AssertNotCalled(t, "DeleteApiDnsZonesIdRecordsNameTypeWithResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		mockClient.AssertNotCalled(t, "DeleteApiDnsZonesIdRecordsNameTypeWithBodyWithResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestDNSService_BulkCreateRecords(t *testing.T) {
	t.Run("creates multiple records", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedRecords := []RecordResponse{
			{Name: "www", Type: "A", Content: "1.2.3.4"},
			{Name: "mail", Type: "A", Content: "5.6.7.8"},
		}

		mockClient.EXPECT().
			PostApiDnsZonesIdRecordsBulkWithResponse(mock.Anything, "123", mock.MatchedBy(func(bulkReq BulkRecordRequest) bool {
				return len(bulkReq.Records) == 2
			})).
			Return(&internalclient.PostApiDnsZonesIdRecordsBulkResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &internalclient.BulkRecordsResponse{
					Records: expectedRecords,
				},
			}, nil).
			Once()

		records := []RecordRequest{
			{Name: "www", Type: "A", Content: "1.2.3.4"},
			{Name: "mail", Type: "A", Content: "5.6.7.8"},
		}

		results, err := service.BulkCreateRecords(context.Background(), "123", records)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("handles empty record list", func(t *testing.T) {
		service := NewDNSService(nil)
		results, err := service.BulkCreateRecords(context.Background(), "123", []RecordRequest{})

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestDNSService_BulkDeleteRecords(t *testing.T) {
	t.Run("deletes multiple records", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedResults := []RecordResult{
			{Name: "www", Type: "A", Status: "success"},
			{Name: "mail", Type: "A", Status: "success"},
		}

		mockClient.EXPECT().
			PostApiDnsZonesIdRecordsBulkDeleteWithResponse(mock.Anything, "123", mock.Anything).
			Return(&internalclient.PostApiDnsZonesIdRecordsBulkDeleteResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &internalclient.BulkDeleteResponse{
					Results: expectedResults,
				},
			}, nil).
			Once()

		identifiers := []RecordIdentifier{
			{Name: "www", Type: "A"},
			{Name: "mail", Type: "A"},
		}

		results, err := service.BulkDeleteRecords(context.Background(), "123", identifiers)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("handles empty identifier list", func(t *testing.T) {
		service := NewDNSService(nil)
		results, err := service.BulkDeleteRecords(context.Background(), "123", []RecordIdentifier{})

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestDNSService_GetZoneStatus(t *testing.T) {
	t.Run("returns zone status on success", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedZone := &internalclient.ZoneResponse{
			Id:     123,
			Domain: "example.com",
			Status: "active",
		}

		mockClient.EXPECT().
			GetApiDnsZonesIdStatusWithResponse(mock.Anything, "123").
			Return(&internalclient.GetApiDnsZonesIdStatusResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedZone,
			}, nil).
			Once()

		zone, err := service.GetZoneStatus(context.Background(), "123")

		require.NoError(t, err)
		assert.NotNil(t, zone)
		assert.Equal(t, 123, zone.Id)
		assert.Equal(t, "active", zone.Status)
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetApiDnsZonesIdStatusWithResponse(mock.Anything, "999").
			Return(&internalclient.GetApiDnsZonesIdStatusResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).
			Once()

		zone, err := service.GetZoneStatus(context.Background(), "999")

		assert.Error(t, err)
		assert.Nil(t, zone)
	})
}

func TestDNSService_ValidateZone(t *testing.T) {
	t.Run("returns validation result on success", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		validatedAt := time.Now()
		expectedValidation := &internalclient.ValidationResponse{
			Valid:       true,
			Message:     "Zone is properly configured",
			CheckedAt:   validatedAt,
			Nameservers: &[]string{"ns1.example.com", "ns2.example.com"},
		}

		mockClient.EXPECT().
			PostApiDnsZonesIdValidateWithResponse(mock.Anything, "123").
			Return(&internalclient.PostApiDnsZonesIdValidateResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedValidation,
			}, nil).
			Once()

		validation, err := service.ValidateZone(context.Background(), "123")

		require.NoError(t, err)
		assert.NotNil(t, validation)
		assert.True(t, validation.Valid)
		assert.Equal(t, "Zone is properly configured", validation.Message)
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			PostApiDnsZonesIdValidateWithResponse(mock.Anything, "999").
			Return(&internalclient.PostApiDnsZonesIdValidateResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).
			Once()

		validation, err := service.ValidateZone(context.Background(), "999")

		assert.Error(t, err)
		assert.Nil(t, validation)
	})
}

func TestDNSService_PushCert(t *testing.T) {
	t.Run("returns cert push response on success", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedResp := &internalclient.CertPushResponse{
			Ok:        true,
			OwnerName: "example.com.",
			Tlsa:      "3 1 1 abc123",
		}

		req := CertPushRequest{
			CertPem:   "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			Domain:    "example.com",
			Namespace: "_443._tcp",
		}

		mockClient.EXPECT().
			PostInternalDnsCertWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsCertResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.PushCert(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Ok)
		assert.Equal(t, "example.com.", result.OwnerName)
		assert.Equal(t, "3 1 1 abc123", result.Tlsa)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := CertPushRequest{
			CertPem:   "invalid",
			Domain:    "example.com",
			Namespace: "_443._tcp",
		}

		mockClient.EXPECT().
			PostInternalDnsCertWithResponse(mock.Anything, req).
			Return(nil, assert.AnError).
			Once()

		result, err := service.PushCert(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := CertPushRequest{
			CertPem:   "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			Domain:    "example.com",
			Namespace: "_443._tcp",
		}

		mockClient.EXPECT().
			PostInternalDnsCertWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsCertResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.PushCert(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on bad request", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := CertPushRequest{
			CertPem:   "bad-cert",
			Domain:    "example.com",
			Namespace: "_443._tcp",
		}

		mockClient.EXPECT().
			PostInternalDnsCertWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsCertResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "invalid certificate data"}},
			}, nil).
			Once()

		result, err := service.PushCert(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDNSService_UpdateTLSA(t *testing.T) {
	t.Run("returns TLSA update response on success", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedResp := &internalclient.CertPushResponse{
			Ok:        true,
			OwnerName: "example.com.",
			Tlsa:      "3 1 1 abc123",
		}

		req := TLSAUpdateRequest{
			CertPem:   "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			Domain:    "example.com",
			Namespace: "_443._tcp",
			Tlsa:      "3 1 1 abc123",
		}

		mockClient.EXPECT().
			PostInternalDnsTlsaWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsTlsaResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.UpdateTLSA(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Ok)
		assert.Equal(t, "3 1 1 abc123", result.Tlsa)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := TLSAUpdateRequest{
			CertPem:   "invalid",
			Domain:    "example.com",
			Namespace: "_443._tcp",
			Tlsa:      "3 1 1 abc123",
		}

		mockClient.EXPECT().
			PostInternalDnsTlsaWithResponse(mock.Anything, req).
			Return(nil, assert.AnError).
			Once()

		result, err := service.UpdateTLSA(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := TLSAUpdateRequest{
			CertPem:   "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			Domain:    "example.com",
			Namespace: "_443._tcp",
			Tlsa:      "3 1 1 abc123",
		}

		mockClient.EXPECT().
			PostInternalDnsTlsaWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsTlsaResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.UpdateTLSA(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on bad request", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		req := TLSAUpdateRequest{
			CertPem:   "bad-cert",
			Domain:    "example.com",
			Namespace: "_443._tcp",
			Tlsa:      "invalid-tlsa",
		}

		mockClient.EXPECT().
			PostInternalDnsTlsaWithResponse(mock.Anything, req).
			Return(&internalclient.PostInternalDnsTlsaResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusBadRequest},
				JSON400:      &internalclient.ErrorResponse{Error: internalclient.ErrorDetail{Reason: "invalid TLSA data"}},
			}, nil).
			Once()

		result, err := service.UpdateTLSA(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDNSService_GetCert(t *testing.T) {
	t.Run("returns stored DANE cert and key on success", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		expectedResp := &internalclient.CertGetResponse{
			Ok:            true,
			Domain:        "example",
			Namespace:     "hns",
			CertPem:       "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			PrivateKeyPem: "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
			Tlsa:          "3 1 1 abc123",
			OwnerName:     "_443._tcp.example.",
		}

		mockClient.EXPECT().
			GetInternalDnsCertDomainWithResponse(mock.Anything, "example", &internalclient.GetInternalDnsCertDomainParams{Namespace: new("hns")}).
			Return(&internalclient.GetInternalDnsCertDomainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expectedResp,
			}, nil).
			Once()

		result, err := service.GetCert(context.Background(), "example", "hns")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Ok)
		assert.Equal(t, expectedResp.PrivateKeyPem, result.PrivateKeyPem)
		assert.Equal(t, expectedResp.CertPem, result.CertPem)
		assert.Equal(t, expectedResp.Tlsa, result.Tlsa)
	})

	t.Run("returns error on HTTP error", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetInternalDnsCertDomainWithResponse(mock.Anything, "example", &internalclient.GetInternalDnsCertDomainParams{}).
			Return(nil, assert.AnError).
			Once()

		result, err := service.GetCert(context.Background(), "example", "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns ErrNotFound on 404", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetInternalDnsCertDomainWithResponse(mock.Anything, "example", &internalclient.GetInternalDnsCertDomainParams{Namespace: new("hns")}).
			Return(&internalclient.GetInternalDnsCertDomainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			}, nil).
			Once()

		result, err := service.GetCert(context.Background(), "example", "hns")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("returns ErrForbidden on 403", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetInternalDnsCertDomainWithResponse(mock.Anything, "example", &internalclient.GetInternalDnsCertDomainParams{Namespace: new("hns")}).
			Return(&internalclient.GetInternalDnsCertDomainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusForbidden},
			}, nil).
			Once()

		result, err := service.GetCert(context.Background(), "example", "hns")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrForbidden)
		assert.Nil(t, result)
	})

	t.Run("returns error when JSON200 is nil", func(t *testing.T) {
		service, mockClient := testDNSService(t)

		mockClient.EXPECT().
			GetInternalDnsCertDomainWithResponse(mock.Anything, "example", &internalclient.GetInternalDnsCertDomainParams{Namespace: new("hns")}).
			Return(&internalclient.GetInternalDnsCertDomainResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := service.GetCert(context.Background(), "example", "hns")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// mockReadCloser creates a ReadCloser from byte slice
type mockReadCloser []byte

func (m mockReadCloser) Read(p []byte) (int, error) {
	return copy(p, m), nil
}

func (m mockReadCloser) Close() error {
	return nil
}
