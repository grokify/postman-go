// Package collectionitems provides a high-level client for managing the
// individual items (folders, requests, and responses) nested inside a
// Postman collection.
//
// Each item is addressed by its own ID plus the ID of the collection that
// contains it. This package covers each item's basic properties (name,
// description, URL, method, headers, and similar). It does not model the
// more elaborate Postman Collection Format structures (auth, pre-request and
// test scripts, form-data/GraphQL bodies) — those are left untouched by
// Create/Update calls made through this package. See
// https://schema.postman.com/collection/json/v2.1.0/draft-07/docs/index.html
// for the full format.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	folder, _ := client.CollectionItems().CreateFolder(ctx, "collection-id", &collectionitems.CreateFolderInput{
//		Name: "New Folder",
//	})
package collectionitems

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-faster/jx"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level collection items client. Obtain one via
// postman.Client.CollectionItems.
type Service struct {
	api *api.Client
}

// New creates a CollectionItems service over the given generated API client.
// Most callers should use postman.Client.CollectionItems instead of calling
// this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// GetOptions are the common query options accepted by the Get* methods.
type GetOptions struct {
	// IDs, when true, returns only properties that contain ID values.
	IDs bool
	// UID, when true, returns all IDs in UID format (userId-id).
	UID bool
	// Populate, when true, returns all of the item's contents.
	Populate bool
}

func additionalPropsToMap[M ~map[string]jx.Raw](m M) map[string]json.RawMessage {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		out[k] = json.RawMessage(v)
	}
	return out
}

func additionalPropsFromMap[M ~map[string]jx.Raw](m map[string]json.RawMessage) M {
	if len(m) == 0 {
		return nil
	}
	out := make(M, len(m))
	for k, v := range m {
		out[k] = jx.Raw(v)
	}
	return out
}

// --- Folder ---------------------------------------------------------------

// Folder describes a folder nested in a collection.
type Folder struct {
	ID          string
	Name        string
	Description string
	Owner       string
	// ParentFolder is the ID of the folder's parent folder, if any.
	ParentFolder  string
	Folders       []string
	Requests      []string
	Collection    string
	LastUpdatedBy string
	LastRevision  int
	CreatedAt     string
	UpdatedAt     string
	// AdditionalProperties holds any other Postman Collection Format fields
	// present on the folder, keyed by field name.
	AdditionalProperties map[string]json.RawMessage
}

// FolderResult wraps a Folder together with the collection update envelope
// returned by CreateFolder/UpdateFolder.
type FolderResult struct {
	Folder Folder
	// ModelID identifies the affected collection model.
	ModelID string
	// Revision is the collection's revision number after this change. Not
	// reported by GetFolder.
	Revision int
}

// DeletedFolder identifies a deleted folder.
type DeletedFolder struct {
	ID    string
	Owner string
}

// CreateFolderInput holds the fields for creating a folder.
type CreateFolderInput struct {
	// Name is the folder's name. If empty, Postman creates the folder with a blank name.
	Name string
	// Folder is the ID of the parent folder to nest this folder inside. If
	// empty, the folder is created at the collection's root.
	Folder string
	// AdditionalProperties sets any other Postman Collection Format fields.
	AdditionalProperties map[string]json.RawMessage
}

func folderFromCreatedData(d api.CollectionFolderCreatedData) Folder {
	return Folder{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Owner:                d.Owner.Or(""),
		ParentFolder:         d.Folder.Or(""),
		Folders:              d.Folders,
		Requests:             d.Requests,
		Collection:           d.Collection.Or(""),
		Description:          d.Description.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// CreateFolder creates a folder in a collection. It is recommended to set
// Name; otherwise Postman creates the folder with a blank name.
func (s *Service) CreateFolder(ctx context.Context, collectionID string, in *CreateFolderInput) (*FolderResult, error) {
	if in == nil {
		in = &CreateFolderInput{}
	}
	req := &api.CreateFolder{}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Folder != "" {
		req.Folder = api.NewOptString(in.Folder)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptCreateFolderAdditionalProperties(
			additionalPropsFromMap[api.CreateFolderAdditionalProperties](in.AdditionalProperties))
	}

	res, err := s.api.CreateCollectionFolder(ctx, req, api.CreateCollectionFolderParams{CollectionId: collectionID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionFolderCreated:
		out := &FolderResult{ModelID: r.ModelId.Or(""), Revision: r.Revision.Or(0)}
		if d, ok := r.Data.Get(); ok {
			out.Folder = folderFromCreatedData(d)
		}
		return out, nil
	case *api.CreateCollectionFolderBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateCollectionFolderUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func folderFromInfoData(d api.CollectionFolderInfoData) Folder {
	return Folder{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Description:          d.Description.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		Owner:                d.Owner.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		Collection:           d.Collection.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// GetFolder returns information about a folder in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) GetFolder(ctx context.Context, collectionID, folderID string, opts *GetOptions) (*FolderResult, error) {
	params := api.GetCollectionFolderParams{CollectionId: collectionID, FolderId: folderID}
	if opts != nil {
		if opts.IDs {
			params.Ids = api.NewOptBool(true)
		}
		if opts.UID {
			params.UID = api.NewOptBool(true)
		}
		if opts.Populate {
			params.Populate = api.NewOptBool(true)
		}
	}

	res, err := s.api.GetCollectionFolder(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionFolderInfo:
		out := &FolderResult{ModelID: r.ModelId.Or("")}
		if d, ok := r.Data.Get(); ok {
			out.Folder = folderFromInfoData(d)
		}
		return out, nil
	case *api.GetCollectionFolderNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetCollectionFolderUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateFolderInput holds the fields for updating a folder. This behaves like
// a PATCH: only non-zero fields are sent, and moving the folder to a
// different parent is not supported by this endpoint.
type UpdateFolderInput struct {
	Name                 string
	Description          string
	AdditionalProperties map[string]json.RawMessage
}

func folderFromUpdatedData(d api.CollectionFolderUpdatedData) Folder {
	return Folder{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Description:          d.Description.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		Owner:                d.Owner.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		Collection:           d.Collection.Or(""),
		ParentFolder:         d.Folder.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// UpdateFolder updates a folder's name and description.
func (s *Service) UpdateFolder(ctx context.Context, collectionID, folderID string, in *UpdateFolderInput) (*FolderResult, error) {
	if in == nil {
		in = &UpdateFolderInput{}
	}
	req := &api.UpdateFolder{}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptString(in.Description)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptUpdateFolderAdditionalProperties(
			additionalPropsFromMap[api.UpdateFolderAdditionalProperties](in.AdditionalProperties))
	}

	res, err := s.api.UpdateCollectionFolder(ctx, req, api.UpdateCollectionFolderParams{CollectionId: collectionID, FolderId: folderID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionFolderUpdated:
		out := &FolderResult{}
		if d, ok := r.Data.Get(); ok {
			out.Folder = folderFromUpdatedData(d)
		}
		return out, nil
	case *api.UpdateCollectionFolderBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateCollectionFolderNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateCollectionFolderUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// DeleteFolder deletes a folder in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) DeleteFolder(ctx context.Context, collectionID, folderID string) (*DeletedFolder, error) {
	res, err := s.api.DeleteCollectionFolder(ctx, api.DeleteCollectionFolderParams{CollectionId: collectionID, FolderId: folderID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionFolderDeleted:
		out := &DeletedFolder{}
		if d, ok := r.Data.Get(); ok {
			out.ID = d.ID.Or("")
			out.Owner = d.Owner.Or("")
		}
		return out, nil
	case *api.DeleteCollectionFolderNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteCollectionFolderUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Request ----------------------------------------------------------------

// Header is an HTTP header key/value pair.
type Header struct {
	Key         string
	Value       string
	Description string
}

// QueryParam is a URL query parameter.
type QueryParam struct {
	Key         string
	Value       string
	Description string
	Enabled     bool
}

// Request describes a request nested in a collection.
type Request struct {
	ID          string
	Name        string
	Description string
	Owner       string
	// ParentFolder is the ID of the folder that contains the request, if any.
	ParentFolder         string
	Responses            []string
	Collection           string
	LastUpdatedBy        string
	LastRevision         int
	CreatedAt            string
	UpdatedAt            string
	AdditionalProperties map[string]json.RawMessage
}

// RequestResult wraps a Request together with the collection update envelope
// returned by CreateRequest/UpdateRequest.
type RequestResult struct {
	Request Request
	ModelID string
	// Revision is the collection's revision number after this change. Not
	// reported by GetRequest.
	Revision int
}

// DeletedRequest identifies a deleted request.
type DeletedRequest struct {
	ID    string
	Owner string
}

// CreateRequestInput holds the fields for creating a request.
type CreateRequestInput struct {
	// FolderID is the ID of the folder to create the request in. If empty,
	// the request is created at the collection's root.
	FolderID    string
	Name        string
	Description string
	Method      string
	URL         string
	Headers     []Header
	QueryParams []QueryParam
	// DataMode selects the request body's encoding, e.g. "raw", "urlencoded", "formdata".
	DataMode string
	// RawModeData is the request body when DataMode is "raw".
	RawModeData          string
	AdditionalProperties map[string]json.RawMessage
}

func requestHeadersFromInput(hs []Header) []api.RequestHeaderData {
	if len(hs) == 0 {
		return nil
	}
	out := make([]api.RequestHeaderData, 0, len(hs))
	for _, h := range hs {
		rh := api.RequestHeaderData{}
		if h.Key != "" {
			rh.Key = api.NewOptString(h.Key)
		}
		if h.Value != "" {
			rh.Value = api.NewOptString(h.Value)
		}
		if h.Description != "" {
			rh.Description = api.NewOptString(h.Description)
		}
		out = append(out, rh)
	}
	return out
}

func requestQueryParamsFromInput(qs []QueryParam) []api.RequestQueryParams {
	if len(qs) == 0 {
		return nil
	}
	out := make([]api.RequestQueryParams, 0, len(qs))
	for _, q := range qs {
		rq := api.RequestQueryParams{}
		if q.Key != "" {
			rq.Key = api.NewOptString(q.Key)
		}
		if q.Value != "" {
			rq.Value = api.NewOptString(q.Value)
		}
		if q.Description != "" {
			rq.Description = api.NewOptString(q.Description)
		}
		if q.Enabled {
			rq.Enabled = api.NewOptBool(true)
		}
		out = append(out, rq)
	}
	return out
}

func requestFromCreatedData(d api.CollectionRequestCreatedData) Request {
	return Request{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Owner:                d.Owner.Or(""),
		ParentFolder:         d.Folder.Or(""),
		Responses:            d.Responses,
		Collection:           d.Collection.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// CreateRequest creates a request in a collection. It is recommended to set
// Name; otherwise Postman creates the request with a blank name.
func (s *Service) CreateRequest(ctx context.Context, collectionID string, in *CreateRequestInput) (*RequestResult, error) {
	if in == nil {
		in = &CreateRequestInput{}
	}
	req := &api.CreateRequest{
		HeaderData:  requestHeadersFromInput(in.Headers),
		QueryParams: requestQueryParamsFromInput(in.QueryParams),
	}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptNilString(in.Description)
	}
	if in.Method != "" {
		req.Method = api.NewOptString(in.Method)
	}
	if in.URL != "" {
		req.URL = api.NewOptNilString(in.URL)
	}
	if in.DataMode != "" {
		req.DataMode = api.NewOptString(in.DataMode)
	}
	if in.RawModeData != "" {
		req.RawModeData = api.NewOptNilString(in.RawModeData)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptCreateRequestAdditionalProperties(
			additionalPropsFromMap[api.CreateRequestAdditionalProperties](in.AdditionalProperties))
	}

	params := api.CreateCollectionRequestParams{CollectionId: collectionID}
	if in.FolderID != "" {
		params.FolderId = api.NewOptString(in.FolderID)
	}

	res, err := s.api.CreateCollectionRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionRequestCreated:
		out := &RequestResult{ModelID: r.ModelId.Or(""), Revision: r.Revision.Or(0)}
		if d, ok := r.Data.Get(); ok {
			out.Request = requestFromCreatedData(d)
		}
		return out, nil
	case *api.CreateCollectionRequestBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateCollectionRequestUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func requestFromInfoData(d api.CollectionRequestInfoData) Request {
	return Request{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Owner:                d.Owner.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// GetRequest returns information about a request in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) GetRequest(ctx context.Context, collectionID, requestID string, opts *GetOptions) (*RequestResult, error) {
	params := api.GetCollectionRequestParams{CollectionId: collectionID, RequestId: requestID}
	if opts != nil {
		if opts.IDs {
			params.Ids = api.NewOptBool(true)
		}
		if opts.UID {
			params.UID = api.NewOptBool(true)
		}
		if opts.Populate {
			params.Populate = api.NewOptBool(true)
		}
	}

	res, err := s.api.GetCollectionRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionRequestInfo:
		out := &RequestResult{ModelID: r.ModelId.Or("")}
		if d, ok := r.Data.Get(); ok {
			out.Request = requestFromInfoData(d)
		}
		return out, nil
	case *api.GetCollectionRequestNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetCollectionRequestUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateRequestInput holds the fields for updating a request. This endpoint
// does not support moving the request to a different folder.
type UpdateRequestInput struct {
	Name                 string
	Description          string
	Method               string
	URL                  string
	Headers              []Header
	QueryParams          []QueryParam
	DataMode             string
	RawModeData          string
	AdditionalProperties map[string]json.RawMessage
}

func requestFromUpdatedData(d api.CollectionRequestUpdatedData) Request {
	return Request{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Description:          d.Description.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		Owner:                d.Owner.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// UpdateRequest updates a request's properties.
func (s *Service) UpdateRequest(ctx context.Context, collectionID, requestID string, in *UpdateRequestInput) (*RequestResult, error) {
	if in == nil {
		in = &UpdateRequestInput{}
	}
	req := &api.UpdateRequest{
		HeaderData:  requestHeadersFromInput(in.Headers),
		QueryParams: requestQueryParamsFromInput(in.QueryParams),
	}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptNilString(in.Description)
	}
	if in.Method != "" {
		req.Method = api.NewOptString(in.Method)
	}
	if in.URL != "" {
		req.URL = api.NewOptNilString(in.URL)
	}
	if in.DataMode != "" {
		req.DataMode = api.NewOptString(in.DataMode)
	}
	if in.RawModeData != "" {
		req.RawModeData = api.NewOptNilString(in.RawModeData)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptUpdateRequestAdditionalProperties(
			additionalPropsFromMap[api.UpdateRequestAdditionalProperties](in.AdditionalProperties))
	}

	res, err := s.api.UpdateCollectionRequest(ctx, req, api.UpdateCollectionRequestParams{CollectionId: collectionID, RequestId: requestID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionRequestUpdated:
		out := &RequestResult{}
		if d, ok := r.Data.Get(); ok {
			out.Request = requestFromUpdatedData(d)
		}
		return out, nil
	case *api.UpdateCollectionRequestBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateCollectionRequestNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateCollectionRequestUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// DeleteRequest deletes a request in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) DeleteRequest(ctx context.Context, collectionID, requestID string) (*DeletedRequest, error) {
	res, err := s.api.DeleteCollectionRequest(ctx, api.DeleteCollectionRequestParams{CollectionId: collectionID, RequestId: requestID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionRequestDeleted:
		out := &DeletedRequest{}
		if d, ok := r.Data.Get(); ok {
			out.ID = d.ID.Or("")
			out.Owner = d.Owner.Or("")
		}
		return out, nil
	case *api.DeleteCollectionRequestNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteCollectionRequestUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Response -----------------------------------------------------------

// ResponseCode is an HTTP status code with its reason phrase, as recorded on
// a saved example response.
type ResponseCode struct {
	Code int
	Name string
}

// Response describes a saved example response nested in a collection.
type Response struct {
	ID   string
	Name string
	// Request is the ID of the parent request.
	Request              string
	Owner                string
	Collection           string
	LastUpdatedBy        string
	LastRevision         int
	CreatedAt            string
	UpdatedAt            string
	AdditionalProperties map[string]json.RawMessage
}

// ResponseResult wraps a Response together with the collection update
// envelope returned by CreateResponse/UpdateResponse.
type ResponseResult struct {
	Response Response
	ModelID  string
	// Revision is the collection's revision number after this change. Not
	// reported by GetResponse.
	Revision int
}

// DeletedResponse identifies a deleted response.
type DeletedResponse struct {
	ID    string
	Owner string
}

// responseFields holds the fields shared by CreateResponseInput and
// UpdateResponseInput.
type responseFields struct {
	Name          string
	Description   string
	URL           string
	Method        string
	Headers       []Header
	DataMode      string
	RawModeData   string
	ResponseCode  *ResponseCode
	Status        string
	Time          string
	Cookies       string
	Mime          string
	Text          string
	Language      string
	RawDataType   string
	RequestObject string
}

func responseHeadersFromInput(hs []Header) []api.ResponseHeader22 {
	if len(hs) == 0 {
		return nil
	}
	out := make([]api.ResponseHeader22, 0, len(hs))
	for _, h := range hs {
		rh := api.ResponseHeader22{Key: h.Key, Value: h.Value}
		if h.Description != "" {
			rh.Description = api.NewOptNilString(h.Description)
		}
		out = append(out, rh)
	}
	return out
}

// CreateResponseInput holds the fields for creating a response.
type CreateResponseInput struct {
	// RequestID is the ID of the parent request. Required.
	RequestID string
	responseFields
	AdditionalProperties map[string]json.RawMessage
}

func responseFromCreatedData(d api.CollectionResponseCreatedData) Response {
	return Response{
		ID:                   d.ID.Or(""),
		Owner:                d.Owner.Or(""),
		Request:              d.Request.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// CreateResponse creates a saved example response for a request in a
// collection. It is recommended to set Name; otherwise Postman creates the
// response with a blank name.
func (s *Service) CreateResponse(ctx context.Context, collectionID string, in *CreateResponseInput) (*ResponseResult, error) {
	if in == nil {
		in = &CreateResponseInput{}
	}
	req := &api.CreateCollectionResponseRequest{Headers: responseHeadersFromInput(in.Headers)}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptNilString(in.Description)
	}
	if in.URL != "" {
		req.URL = api.NewOptNilString(in.URL)
	}
	if in.Method != "" {
		req.Method = api.NewOptString(in.Method)
	}
	if in.DataMode != "" {
		req.DataMode = api.NewOptString(in.DataMode)
	}
	if in.RawModeData != "" {
		req.RawModeData = api.NewOptNilString(in.RawModeData)
	}
	if in.ResponseCode != nil {
		req.ResponseCode = api.NewOptCollectionResponseCreatedResponseCode(api.CollectionResponseCreatedResponseCode{
			Code: api.NewOptInt(in.ResponseCode.Code),
			Name: api.NewOptString(in.ResponseCode.Name),
		})
	}
	if in.Status != "" {
		req.Status = api.NewOptNilString(in.Status)
	}
	if in.Time != "" {
		req.Time = api.NewOptString(in.Time)
	}
	if in.Cookies != "" {
		req.Cookies = api.NewOptNilString(in.Cookies)
	}
	if in.Mime != "" {
		req.Mime = api.NewOptNilString(in.Mime)
	}
	if in.Text != "" {
		req.Text = api.NewOptString(in.Text)
	}
	if in.Language != "" {
		req.Language = api.NewOptString(in.Language)
	}
	if in.RawDataType != "" {
		req.RawDataType = api.NewOptNilString(in.RawDataType)
	}
	if in.RequestObject != "" {
		req.RequestObject = api.NewOptString(in.RequestObject)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptCreateCollectionResponseRequestAdditionalProperties(
			additionalPropsFromMap[api.CreateCollectionResponseRequestAdditionalProperties](in.AdditionalProperties))
	}

	params := api.CreateCollectionResponseParams{CollectionId: collectionID, Request: in.RequestID}

	res, err := s.api.CreateCollectionResponse(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateCollectionResponseOkResponse:
		out := &ResponseResult{ModelID: r.ModelId.Or(""), Revision: r.Revision.Or(0)}
		if d, ok := r.Data.Get(); ok {
			out.Response = responseFromCreatedData(d)
		}
		return out, nil
	case *api.CreateCollectionResponseBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateCollectionResponseUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func responseFromInfoData(d api.CollectionResponseInfoData) Response {
	return Response{
		ID:                   d.ID.Or(""),
		Request:              d.Request.Or(""),
		Name:                 d.Name.Or(""),
		Owner:                d.Owner.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// GetResponse returns information about a response in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) GetResponse(ctx context.Context, collectionID, responseID string, opts *GetOptions) (*ResponseResult, error) {
	params := api.GetCollectionResponseParams{CollectionId: collectionID, ResponseId: responseID}
	if opts != nil {
		if opts.IDs {
			params.Ids = api.NewOptBool(true)
		}
		if opts.UID {
			params.UID = api.NewOptBool(true)
		}
		if opts.Populate {
			params.Populate = api.NewOptBool(true)
		}
	}

	res, err := s.api.GetCollectionResponse(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionResponseInfo:
		out := &ResponseResult{ModelID: r.ModelId.Or("")}
		if d, ok := r.Data.Get(); ok {
			out.Response = responseFromInfoData(d)
		}
		return out, nil
	case *api.GetCollectionResponseNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetCollectionResponseUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateResponseInput holds the fields for updating a response.
type UpdateResponseInput struct {
	responseFields
	AdditionalProperties map[string]json.RawMessage
}

func responseFromUpdatedData(d api.CollectionResponseUpdatedData) Response {
	return Response{
		ID:                   d.ID.Or(""),
		Name:                 d.Name.Or(""),
		Owner:                d.Owner.Or(""),
		CreatedAt:            d.CreatedAt.Or(""),
		UpdatedAt:            d.UpdatedAt.Or(""),
		LastRevision:         d.LastRevision.Or(0),
		LastUpdatedBy:        d.LastUpdatedBy.Or(""),
		AdditionalProperties: additionalPropsToMap(d.AdditionalProperties.Or(nil)),
	}
}

// UpdateResponse updates a response's properties.
func (s *Service) UpdateResponse(ctx context.Context, collectionID, responseID string, in *UpdateResponseInput) (*ResponseResult, error) {
	if in == nil {
		in = &UpdateResponseInput{}
	}
	req := &api.UpdateCollectionResponse1{Headers: responseHeadersFromInput(in.Headers)}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptNilString(in.Description)
	}
	if in.URL != "" {
		req.URL = api.NewOptNilString(in.URL)
	}
	if in.Method != "" {
		req.Method = api.NewOptString(in.Method)
	}
	if in.DataMode != "" {
		req.DataMode = api.NewOptString(in.DataMode)
	}
	if in.RawModeData != "" {
		req.RawModeData = api.NewOptNilString(in.RawModeData)
	}
	if in.ResponseCode != nil {
		req.ResponseCode = api.NewOptUpdateCollectionResponseResponseCode(api.UpdateCollectionResponseResponseCode{
			Code: api.NewOptInt(in.ResponseCode.Code),
			Name: api.NewOptString(in.ResponseCode.Name),
		})
	}
	if in.Status != "" {
		req.Status = api.NewOptNilString(in.Status)
	}
	if in.Time != "" {
		req.Time = api.NewOptString(in.Time)
	}
	if in.Cookies != "" {
		req.Cookies = api.NewOptNilString(in.Cookies)
	}
	if in.Mime != "" {
		req.Mime = api.NewOptNilString(in.Mime)
	}
	if in.Text != "" {
		req.Text = api.NewOptString(in.Text)
	}
	if in.Language != "" {
		req.Language = api.NewOptString(in.Language)
	}
	if in.RawDataType != "" {
		req.RawDataType = api.NewOptNilString(in.RawDataType)
	}
	if in.RequestObject != "" {
		req.RequestObject = api.NewOptString(in.RequestObject)
	}
	if len(in.AdditionalProperties) > 0 {
		req.AdditionalProperties = api.NewOptUpdateCollectionResponse1AdditionalProperties(
			additionalPropsFromMap[api.UpdateCollectionResponse1AdditionalProperties](in.AdditionalProperties))
	}

	res, err := s.api.UpdateCollectionResponse(ctx, req, api.UpdateCollectionResponseParams{CollectionId: collectionID, ResponseId: responseID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionResponseUpdated:
		out := &ResponseResult{}
		if d, ok := r.Data.Get(); ok {
			out.Response = responseFromUpdatedData(d)
		}
		return out, nil
	case *api.UpdateCollectionResponseBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateCollectionResponseNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateCollectionResponseUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// DeleteResponse deletes a response in a collection.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) DeleteResponse(ctx context.Context, collectionID, responseID string) (*DeletedResponse, error) {
	res, err := s.api.DeleteCollectionResponse(ctx, api.DeleteCollectionResponseParams{CollectionId: collectionID, ResponseId: responseID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionResponseDeleted:
		out := &DeletedResponse{}
		if d, ok := r.Data.Get(); ok {
			out.ID = d.ID.Or("")
			out.Owner = d.Owner.Or("")
		}
		return out, nil
	case *api.DeleteCollectionResponseNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteCollectionResponseUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
