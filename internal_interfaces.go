package ipfs

import (
	"context"

	dnsreq "go.lumeweb.com/ipfs-sdk/internal/dnsreq"
	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
)

// IPNSClientWithResponsesInterface defines the methods needed from the generated internal client for IPNS
type IPNSClientWithResponsesInterface interface {
	GetApiIpnsKeysWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsKeysResponse, error)
	GetApiIpnsKeysIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsKeysIdResponse, error)
	PostApiIpnsKeysWithResponse(ctx context.Context, body internalclient.IPNSKeyRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsKeysResponse, error)
	DeleteApiIpnsKeysIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiIpnsKeysIdResponse, error)
	PostApiIpnsPublishWithResponse(ctx context.Context, body internalclient.IPNSPublishRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsPublishResponse, error)
	PostApiIpnsRepublishWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsRepublishResponse, error)
	GetApiIpnsResolveNameWithResponse(ctx context.Context, name string, params *internalclient.GetApiIpnsResolveNameParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsResolveNameResponse, error)
}

// internalClientToIPNSAdapter adapts internalclient.ClientWithResponses to IPNSClientWithResponsesInterface
type internalClientToIPNSAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToIPNSAdapter) GetApiIpnsKeysWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsKeysResponse, error) {
	return a.client.GetApiIpnsKeysWithResponse(ctx, reqEditors...)
}

func (a *internalClientToIPNSAdapter) GetApiIpnsKeysIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsKeysIdResponse, error) {
	return a.client.GetApiIpnsKeysIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToIPNSAdapter) PostApiIpnsKeysWithResponse(ctx context.Context, body internalclient.IPNSKeyRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsKeysResponse, error) {
	return a.client.PostApiIpnsKeysWithResponse(ctx, body, reqEditors...)
}

func (a *internalClientToIPNSAdapter) DeleteApiIpnsKeysIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiIpnsKeysIdResponse, error) {
	return a.client.DeleteApiIpnsKeysIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToIPNSAdapter) PostApiIpnsPublishWithResponse(ctx context.Context, body internalclient.IPNSPublishRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsPublishResponse, error) {
	return a.client.PostApiIpnsPublishWithResponse(ctx, body, reqEditors...)
}

func (a *internalClientToIPNSAdapter) PostApiIpnsRepublishWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiIpnsRepublishResponse, error) {
	return a.client.PostApiIpnsRepublishWithResponse(ctx, reqEditors...)
}

func (a *internalClientToIPNSAdapter) GetApiIpnsResolveNameWithResponse(ctx context.Context, name string, params *internalclient.GetApiIpnsResolveNameParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiIpnsResolveNameResponse, error) {
	return a.client.GetApiIpnsResolveNameWithResponse(ctx, name, params, reqEditors...)
}

// ConvertClientToIPNS converts a ClientWithResponses to IPNSClientWithResponsesInterface
func ConvertClientToIPNS(client *internalclient.ClientWithResponses) IPNSClientWithResponsesInterface {
	return &internalClientToIPNSAdapter{client: client}
}

// DNSClientWithResponsesInterface defines the methods needed from the generated internal client for DNS
type DNSClientWithResponsesInterface interface {
	GetApiDnsZonesWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesResponse, error)
	GetApiDnsZonesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdResponse, error)
	PostApiDnsZonesWithResponse(ctx context.Context, body dnsreq.ZoneRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesResponse, error)
	DeleteApiDnsZonesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiDnsZonesIdResponse, error)
	GetApiDnsZonesIdStatusWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdStatusResponse, error)
	PostApiDnsZonesIdValidateWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdValidateResponse, error)
	GetApiDnsZonesIdRecordsWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdRecordsResponse, error)
	PostApiDnsZonesIdRecordsWithResponse(ctx context.Context, id string, body dnsreq.RecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsResponse, error)
	GetApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdRecordsNameTypeResponse, error)
	PutApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, body dnsreq.RecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiDnsZonesIdRecordsNameTypeResponse, error)
	DeleteApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiDnsZonesIdRecordsNameTypeResponse, error)
	PostApiDnsZonesIdRecordsBulkWithResponse(ctx context.Context, id string, body dnsreq.BulkRecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsBulkResponse, error)
	PostApiDnsZonesIdRecordsBulkDeleteWithResponse(ctx context.Context, id string, body dnsreq.BulkDeleteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsBulkDeleteResponse, error)
}

// internalClientToDNSAdapter adapts internalclient.ClientWithResponses to DNSClientWithResponsesInterface
type internalClientToDNSAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToDNSAdapter) GetApiDnsZonesWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesResponse, error) {
	return a.client.GetApiDnsZonesWithResponse(ctx, reqEditors...)
}

func (a *internalClientToDNSAdapter) GetApiDnsZonesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdResponse, error) {
	return a.client.GetApiDnsZonesIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToDNSAdapter) PostApiDnsZonesWithResponse(ctx context.Context, body dnsreq.ZoneRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesResponse, error) {
	return a.client.PostApiDnsZonesWithResponse(ctx, body, reqEditors...)
}

func (a *internalClientToDNSAdapter) DeleteApiDnsZonesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiDnsZonesIdResponse, error) {
	return a.client.DeleteApiDnsZonesIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToDNSAdapter) GetApiDnsZonesIdStatusWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdStatusResponse, error) {
	return a.client.GetApiDnsZonesIdStatusWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToDNSAdapter) PostApiDnsZonesIdValidateWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdValidateResponse, error) {
	return a.client.PostApiDnsZonesIdValidateWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToDNSAdapter) GetApiDnsZonesIdRecordsWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdRecordsResponse, error) {
	return a.client.GetApiDnsZonesIdRecordsWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToDNSAdapter) PostApiDnsZonesIdRecordsWithResponse(ctx context.Context, id string, body dnsreq.RecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsResponse, error) {
	return a.client.PostApiDnsZonesIdRecordsWithResponse(ctx, id, body, reqEditors...)
}

func (a *internalClientToDNSAdapter) GetApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiDnsZonesIdRecordsNameTypeResponse, error) {
	return a.client.GetApiDnsZonesIdRecordsNameTypeWithResponse(ctx, id, name, pType, reqEditors...)
}

func (a *internalClientToDNSAdapter) PutApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, body dnsreq.RecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiDnsZonesIdRecordsNameTypeResponse, error) {
	return a.client.PutApiDnsZonesIdRecordsNameTypeWithResponse(ctx, id, name, pType, body, reqEditors...)
}

func (a *internalClientToDNSAdapter) DeleteApiDnsZonesIdRecordsNameTypeWithResponse(ctx context.Context, id string, name string, pType string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiDnsZonesIdRecordsNameTypeResponse, error) {
	return a.client.DeleteApiDnsZonesIdRecordsNameTypeWithResponse(ctx, id, name, pType, reqEditors...)
}

func (a *internalClientToDNSAdapter) PostApiDnsZonesIdRecordsBulkWithResponse(ctx context.Context, id string, body dnsreq.BulkRecordRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsBulkResponse, error) {
	return a.client.PostApiDnsZonesIdRecordsBulkWithResponse(ctx, id, body, reqEditors...)
}

func (a *internalClientToDNSAdapter) PostApiDnsZonesIdRecordsBulkDeleteWithResponse(ctx context.Context, id string, body dnsreq.BulkDeleteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiDnsZonesIdRecordsBulkDeleteResponse, error) {
	return a.client.PostApiDnsZonesIdRecordsBulkDeleteWithResponse(ctx, id, body, reqEditors...)
}

// ConvertClientToDNS converts a ClientWithResponses to DNSClientWithResponsesInterface
func ConvertClientToDNS(client *internalclient.ClientWithResponses) DNSClientWithResponsesInterface {
	return &internalClientToDNSAdapter{client: client}
}
