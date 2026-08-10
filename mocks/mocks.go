// Package mocks provides a high-level client for Postman's Mock Server API.
//
// A mock server simulates the behavior of an existing collection so that
// clients can develop and test against realistic responses before the real
// API exists (or is available). This package covers managing mock servers
// themselves, their call logs, and their server-level (5xx) responses.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	created, _ := client.Mocks().Create(ctx, &mocks.CreateInput{
//		Workspace:  "workspace-id",
//		Collection: "collection-id",
//	})
package mocks

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// DelayType is the type of artificial delay applied to a mock server's responses.
type DelayType string

// DelayTypeFixed is the only supported delay type: a fixed delay in milliseconds.
const DelayTypeFixed DelayType = "fixed"

// DelayPreset is one of Postman's predefined delay durations. The values are
// opaque identifiers assigned by Postman rather than human-readable names.
type DelayPreset string

// Delay preset values.
const (
	DelayPreset1 DelayPreset = "1"
	DelayPreset2 DelayPreset = "2"
)

// ServerResponseLanguage is the content/syntax-highlighting language of a mock
// server's server response body.
type ServerResponseLanguage string

// Server response language values.
const (
	ServerResponseLanguageText       ServerResponseLanguage = "text"
	ServerResponseLanguageJavascript ServerResponseLanguage = "javascript"
	ServerResponseLanguageJSON       ServerResponseLanguage = "json"
	ServerResponseLanguageHTML       ServerResponseLanguage = "html"
	ServerResponseLanguageXML        ServerResponseLanguage = "xml"
)

// CallLogSort is a field that call logs can be sorted by.
type CallLogSort string

// CallLogSortServedAt sorts call logs by the time they were served.
const CallLogSortServedAt CallLogSort = "servedAt"

// SortDirection is an ascending or descending sort direction.
type SortDirection string

// Sort direction values.
const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

// Service is the high-level Mock Server client. Obtain one via
// postman.Client.Mocks.
type Service struct {
	api *api.Client
}

// New creates a Mocks service over the given generated API client. Most
// callers should use postman.Client.Mocks instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Delay describes an artificial delay applied to a mock server's responses.
type Delay struct {
	Type     DelayType
	Duration int
	Preset   DelayPreset
}

// Config holds a mock server's matching and response-delay configuration.
type Config struct {
	// MatchBody, when true, matches incoming requests on their body.
	MatchBody bool
	// MatchHeader, when true, matches incoming requests on their headers.
	// Only reported by the create/update endpoints.
	MatchHeader bool
	// MatchQueryParams, when true, matches incoming requests on their query parameters.
	MatchQueryParams bool
	// MatchWildcards, when true, matches path variables using wildcards.
	MatchWildcards bool
	// Headers lists the header names used for MatchHeader.
	Headers []string
	// Delay is the artificial response delay, if configured.
	Delay *Delay
	// ServerResponseID is the ID of the active server-level (5xx) response, if any.
	ServerResponseID string
}

// Mock describes a mock server.
type Mock struct {
	ID          string
	Owner       string
	UID         string
	Collection  string
	MockURL     string
	Name        string
	Environment string
	Config      *Config
	// IsPublic and Deactivated are only reported by Get.
	IsPublic    bool
	Deactivated bool
	CreatedAt   string
	UpdatedAt   string
}

// DeletedMock identifies a deleted mock server.
type DeletedMock struct {
	ID  string
	UID string
}

// PublishedMock identifies a published or unpublished mock server.
type PublishedMock struct {
	ID string
}

func delayFromCreateUpdate(d api.MockCreateUpdateResponseMockConfigDelay) *Delay {
	return &Delay{
		Type:     DelayType(d.Type.Or("")),
		Duration: d.Duration.Or(0),
		Preset:   DelayPreset(d.Preset.Or("")),
	}
}

func configFromCreateUpdate(c api.MockCreateUpdateResponseMockConfig) *Config {
	out := &Config{
		MatchBody:        c.MatchBody.Or(false),
		MatchHeader:      c.MatchHeader.Or(false),
		MatchQueryParams: c.MatchQueryParams.Or(false),
		MatchWildcards:   c.MatchWildcards.Or(false),
		Headers:          c.Headers,
		ServerResponseID: c.ServerResponseId.Or(""),
	}
	if d, ok := c.Delay.Get(); ok {
		out.Delay = delayFromCreateUpdate(d)
	}
	return out
}

func mockFromCreateUpdate(m api.MockCreateUpdateResponseMock) Mock {
	out := Mock{
		ID:          m.ID.Or(""),
		Owner:       m.Owner.Or(""),
		UID:         m.UID.Or(""),
		Collection:  m.Collection.Or(""),
		MockURL:     m.MockUrl.Or(""),
		Name:        m.Name.Or(""),
		Environment: m.Environment.Or(""),
		CreatedAt:   m.CreatedAt.Or(""),
		UpdatedAt:   m.UpdatedAt.Or(""),
	}
	if c, ok := m.Config.Get(); ok {
		out.Config = configFromCreateUpdate(c)
	}
	return out
}

func delayFromGet(d api.GetMockServerMockConfigDelay) *Delay {
	return &Delay{
		Type:     DelayType(d.Type.Or("")),
		Duration: d.Duration.Or(0),
		Preset:   DelayPreset(d.Preset.Or("")),
	}
}

func configFromGet(c api.GetMockServerMockConfig) *Config {
	out := &Config{
		MatchBody:        c.MatchBody.Or(false),
		MatchQueryParams: c.MatchQueryParams.Or(false),
		MatchWildcards:   c.MatchWildcards.Or(false),
		Headers:          c.Headers,
		ServerResponseID: c.ServerResponseId.Or(""),
	}
	if d, ok := c.Delay.Get(); ok {
		out.Delay = delayFromGet(d)
	}
	return out
}

func mockFromGet(m api.GetMockServerMock) Mock {
	out := Mock{
		ID:          m.ID.Or(""),
		Owner:       m.Owner.Or(""),
		UID:         m.UID.Or(""),
		Collection:  m.Collection.Or(""),
		MockURL:     m.MockUrl.Or(""),
		Name:        m.Name.Or(""),
		Environment: m.Environment.Or(""),
		IsPublic:    m.IsPublic.Or(false),
		Deactivated: m.Deactivated.Or(false),
		CreatedAt:   m.CreatedAt.Or(""),
		UpdatedAt:   m.UpdatedAt.Or(""),
	}
	if c, ok := m.Config.Get(); ok {
		out.Config = configFromGet(c)
	}
	return out
}

func delayFromList(d api.MocksConfigDelay) *Delay {
	return &Delay{
		Type:     DelayType(d.Type.Or("")),
		Duration: d.Duration.Or(0),
		Preset:   DelayPreset(d.Preset.Or("")),
	}
}

func configFromList(c api.MocksConfig) *Config {
	out := &Config{
		MatchBody:        c.MatchBody.Or(false),
		MatchQueryParams: c.MatchQueryParams.Or(false),
		MatchWildcards:   c.MatchWildcards.Or(false),
		Headers:          c.Headers,
		ServerResponseID: c.ServerResponseId.Or(""),
	}
	if d, ok := c.Delay.Get(); ok {
		out.Delay = delayFromList(d)
	}
	return out
}

func mockFromList(m api.GetMockServersMocks) Mock {
	out := Mock{
		ID:          m.ID.Or(""),
		Owner:       m.Owner.Or(""),
		UID:         m.UID.Or(""),
		Collection:  m.Collection.Or(""),
		MockURL:     m.MockUrl.Or(""),
		Name:        m.Name.Or(""),
		Environment: m.Environment.Or(""),
		IsPublic:    m.IsPublic.Or(false),
		Deactivated: m.Deactivated.Or(false),
		CreatedAt:   m.CreatedAt.Or(""),
	}
	if c, ok := m.Config.Get(); ok {
		out.Config = configFromList(c)
	}
	return out
}

// --- List ---------------------------------------------------------------

// ListInput holds the filters for List. A nil or empty input returns all mock
// servers you created across all workspaces.
type ListInput struct {
	// TeamID returns only results that belong to the given team ID.
	TeamID string
	// Workspace returns only results found in the given workspace ID. If both
	// TeamID and Workspace are set, only Workspace is sent to the API.
	Workspace string
}

// List returns all active mock servers visible to the caller.
func (s *Service) List(ctx context.Context, in *ListInput) ([]Mock, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetMocksParams{}
	if in.TeamID != "" {
		params.TeamId = api.NewOptString(in.TeamID)
	}
	if in.Workspace != "" {
		params.Workspace = api.NewOptString(in.Workspace)
	}

	res, err := s.api.GetMocks(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetMockServers:
		out := make([]Mock, 0, len(r.Mocks))
		for _, m := range r.Mocks {
			out = append(out, mockFromList(m))
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create ---------------------------------------------------------------

// CreateInput holds the fields for creating a mock server.
type CreateInput struct {
	// Workspace is the ID of the workspace to create the mock server in. If
	// empty, Postman creates it in your oldest personal Internal workspace.
	Workspace string
	// Collection is the ID of the collection to mock. Required.
	Collection string
	// Environment is the ID of an environment to associate with the mock server.
	Environment string
	// Name is the mock server's name.
	Name string
	// Private, when true, restricts the mock server's access control to private.
	Private bool
}

// Create creates a mock server for a collection.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*Mock, error) {
	if in == nil {
		in = &CreateInput{}
	}
	mock := api.CreateMockMock{Collection: in.Collection}
	if in.Environment != "" {
		mock.Environment = api.NewOptString(in.Environment)
	}
	if in.Name != "" {
		mock.Name = api.NewOptString(in.Name)
	}
	if in.Private {
		mock.Private = api.NewOptBool(true)
	}
	req := &api.CreateMock{Mock: api.NewOptCreateMockMock(mock)}
	params := api.CreateMockParams{Workspace: in.Workspace}

	res, err := s.api.CreateMock(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.MockCreateUpdateResponse:
		if m, ok := r.Mock.Get(); ok {
			out := mockFromCreateUpdate(m)
			return &out, nil
		}
		return &Mock{}, nil
	case *api.Common400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Get --------------------------------------------------------------------

// Get returns information about a mock server.
func (s *Service) Get(ctx context.Context, mockID string) (*Mock, error) {
	res, err := s.api.GetMock(ctx, api.GetMockParams{MockId: mockID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetMockServer:
		if m, ok := r.Mock.Get(); ok {
			out := mockFromGet(m)
			return &out, nil
		}
		return &Mock{}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Update -------------------------------------------------------------

// UpdateInput holds the fields for updating a mock server. Only non-zero
// fields are sent, so this behaves like a partial update.
type UpdateInput struct {
	Name        string
	Environment string
	Description string
	Private     bool
	VersionTag  string
	Collection  string
	// ServerResponseID sets the mock server's active server-level response.
	ServerResponseID string
}

// Update updates a mock server's properties, such as its name or collection.
func (s *Service) Update(ctx context.Context, mockID string, in *UpdateInput) (*Mock, error) {
	if in == nil {
		in = &UpdateInput{}
	}
	mock := api.UpdateMockMock{}
	if in.Name != "" {
		mock.Name = api.NewOptString(in.Name)
	}
	if in.Environment != "" {
		mock.Environment = api.NewOptString(in.Environment)
	}
	if in.Description != "" {
		mock.Description = api.NewOptString(in.Description)
	}
	if in.Private {
		mock.Private = api.NewOptBool(true)
	}
	if in.VersionTag != "" {
		mock.VersionTag = api.NewOptString(in.VersionTag)
	}
	if in.Collection != "" {
		mock.Collection = api.NewOptString(in.Collection)
	}
	if in.ServerResponseID != "" {
		mock.Config = api.NewOptUpdateMockMockConfig(api.UpdateMockMockConfig{
			ServerResponseId: api.NewOptNilString(in.ServerResponseID),
		})
	}
	req := &api.UpdateMock{Mock: api.NewOptUpdateMockMock(mock)}

	res, err := s.api.UpdateMock(ctx, req, api.UpdateMockParams{MockId: mockID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.MockCreateUpdateResponse:
		if m, ok := r.Mock.Get(); ok {
			out := mockFromCreateUpdate(m)
			return &out, nil
		}
		return &Mock{}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	case *api.Mock400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete -------------------------------------------------------------

// Delete deletes a mock server.
func (s *Service) Delete(ctx context.Context, mockID string) (*DeletedMock, error) {
	res, err := s.api.DeleteMock(ctx, api.DeleteMockParams{MockId: mockID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.MockDeleted:
		out := &DeletedMock{}
		if m, ok := r.Mock.Get(); ok {
			out.ID = m.ID.Or("")
			out.UID = m.UID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CallLogs -------------------------------------------------------------

// CallLogsInput holds the filters and pagination options for CallLogs.
type CallLogsInput struct {
	// Limit is the maximum number of rows to return.
	Limit int
	// Cursor is the pagination cursor (use CallLogsResult.NextCursor to page).
	Cursor string
	// Since returns only results created since this ISO 8601 time.
	Since string
	// Until returns only results created until this ISO 8601 time. Cannot be earlier than Since.
	Until string
	// ResponseStatusCode filters by the HTTP response status code.
	ResponseStatusCode int
	// ResponseType filters by the response type (case-insensitive).
	ResponseType string
	// RequestMethod filters by the HTTP request method (case-insensitive).
	RequestMethod string
	// RequestPath filters by the request path (case-insensitive).
	RequestPath string
	// Sort orders the results. Requires Direction to also be set.
	Sort CallLogSort
	// Direction is the sort direction. Requires Sort to also be set.
	Direction SortDirection
	// Include adds header/body data to the call logs. Accepts a comma-separated
	// combination of "request.headers", "request.body", "response.headers", and
	// "response.body".
	Include string
}

// Header is an HTTP header key/value pair.
type Header struct {
	Key         string
	Value       string
	Description string
}

// RequestBodyData is the body of a logged mock request.
type RequestBodyData struct {
	Mode string
	Data string
}

// CallLogRequest is the request half of a logged mock call.
type CallLogRequest struct {
	Method  string
	Path    string
	Headers []Header
	Body    *RequestBodyData
}

// ResponseBodyData is the body of a logged mock response.
type ResponseBodyData struct {
	Data string
}

// CallLogResponse is the response half of a logged mock call.
type CallLogResponse struct {
	Type       string
	StatusCode int
	Headers    []Header
	Body       *ResponseBodyData
}

// CallLog is a single exchanged request/response made to a mock server.
type CallLog struct {
	ID           string
	ResponseName string
	ServedAt     string
	Request      *CallLogRequest
	Response     *CallLogResponse
}

// CallLogsResult is the paginated set of a mock server's call logs.
type CallLogsResult struct {
	CallLogs []CallLog
	// NextCursor points to the next page; empty when there are no more results.
	NextCursor string
}

func callLogRequestFromAPI(r api.CallLogsRequest1) *CallLogRequest {
	out := &CallLogRequest{
		Method: r.Method.Or(""),
		Path:   r.Path.Or(""),
	}
	if h, ok := r.Headers.Get(); ok {
		out.Headers = append(out.Headers, Header{Key: h.Key.Or(""), Value: h.Value.Or("")})
	}
	if b, ok := r.Body.Get(); ok {
		out.Body = &RequestBodyData{Mode: b.Mode.Or(""), Data: b.Data.Or("")}
	}
	return out
}

func callLogResponseFromAPI(r api.CallLogsResponse1) *CallLogResponse {
	out := &CallLogResponse{
		Type:       r.Type.Or(""),
		StatusCode: r.StatusCode.Or(0),
	}
	if h, ok := r.Headers.Get(); ok {
		hdr := Header{Key: h.Key.Or(""), Value: h.Value.Or("")}
		if d, ok := h.Description.Get(); ok {
			hdr.Description = d.Content.Or("")
		}
		out.Headers = append(out.Headers, hdr)
	}
	if b, ok := r.Body.Get(); ok {
		out.Body = &ResponseBodyData{Data: b.Data.Or("")}
	}
	return out
}

// CallLogs returns a mock server's call logs. A single call returns at most
// 6.5MB or 100 call logs, whichever limit is reached first.
func (s *Service) CallLogs(ctx context.Context, mockID string, in *CallLogsInput) (*CallLogsResult, error) {
	if in == nil {
		in = &CallLogsInput{}
	}
	params := api.GetMockCallLogsParams{MockId: mockID}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}
	if in.ResponseStatusCode != 0 {
		params.ResponseStatusCode = api.NewOptInt(in.ResponseStatusCode)
	}
	if in.ResponseType != "" {
		params.ResponseType = api.NewOptString(in.ResponseType)
	}
	if in.RequestMethod != "" {
		params.RequestMethod = api.NewOptString(in.RequestMethod)
	}
	if in.RequestPath != "" {
		params.RequestPath = api.NewOptString(in.RequestPath)
	}
	if in.Sort != "" {
		params.Sort = api.NewOptMockSortServedAt(api.MockSortServedAt(in.Sort))
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}
	if in.Include != "" {
		params.Include = api.NewOptString(in.Include)
	}

	res, err := s.api.GetMockCallLogs(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetCallLogs:
		out := &CallLogsResult{}
		for _, l := range r.CallLogs {
			cl := CallLog{
				ID:           l.ID.Or(""),
				ResponseName: l.ResponseName.Or(""),
				ServedAt:     l.ServedAt.Or(""),
			}
			if req, ok := l.Request.Get(); ok {
				cl.Request = callLogRequestFromAPI(req)
			}
			if resp, ok := l.Response.Get(); ok {
				cl.Response = callLogResponseFromAPI(resp)
			}
			out.CallLogs = append(out.CallLogs, cl)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Mock400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Publish / Unpublish --------------------------------------------------

// Publish publishes a mock server, setting its access control to public.
func (s *Service) Publish(ctx context.Context, mockID string) (*PublishedMock, error) {
	res, err := s.api.PublishMock(ctx, api.PublishMockParams{MockId: mockID})
	if err != nil {
		return nil, err
	}
	return publishedMockFromRes(res)
}

// Unpublish unpublishes a mock server, setting its access control to private.
func (s *Service) Unpublish(ctx context.Context, mockID string) (*PublishedMock, error) {
	res, err := s.api.UnpublishMock(ctx, api.UnpublishMockParams{MockId: mockID})
	if err != nil {
		return nil, err
	}
	return publishedMockFromRes(res)
}

func publishedMockFromRes(res any) (*PublishedMock, error) {
	switch r := res.(type) {
	case *api.MockPublishedUnpublished:
		out := &PublishedMock{}
		if m, ok := r.Mock.Get(); ok {
			out.ID = m.ID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.InternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	case *api.Mock400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Server responses -------------------------------------------------------

// ServerResponseSummary is a mock server's server-level (5xx) response, as
// returned by the ServerResponses listing (without headers/body).
type ServerResponseSummary struct {
	ID         string
	Name       string
	StatusCode int
	CreatedAt  string
	UpdatedAt  string
	CreatedBy  string
	UpdatedBy  string
}

// ServerResponses returns all of a mock server's server-level responses.
func (s *Service) ServerResponses(ctx context.Context, mockID string) ([]ServerResponseSummary, error) {
	res, err := s.api.GetMockServerResponses(ctx, api.GetMockServerResponsesParams{MockId: mockID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetMockServerResponsesOKApplicationJSON:
		out := make([]ServerResponseSummary, 0, len(*r))
		for _, sr := range *r {
			out = append(out, ServerResponseSummary{
				ID:         sr.ID.Or(""),
				Name:       sr.Name.Or(""),
				StatusCode: sr.StatusCode.Or(0),
				CreatedAt:  sr.CreatedAt.Or(""),
				UpdatedAt:  sr.UpdatedAt.Or(""),
				CreatedBy:  sr.CreatedBy.Or(""),
				UpdatedBy:  sr.UpdatedBy.Or(""),
			})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	case *api.GetMockServerResponsesNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// ServerResponseHeader is a header sent with a mock server response.
type ServerResponseHeader struct {
	Key   string
	Value string
}

// ServerResponse describes a mock server's server-level (5xx) response.
type ServerResponse struct {
	ID         string
	Name       string
	StatusCode int
	Headers    []ServerResponseHeader
	Language   ServerResponseLanguage
	Body       string
	CreatedAt  string
	UpdatedAt  string
	CreatedBy  string
	UpdatedBy  string
	// Mock is the parent mock server's ID. Only reported by CreateServerResponse,
	// ServerResponse, and UpdateServerResponse.
	Mock string
}

func serverResponseFromOkResponse(r api.CreateMockServerResponseOkResponse) *ServerResponse {
	out := &ServerResponse{
		ID:         r.ID.Or(""),
		Name:       r.Name.Or(""),
		StatusCode: r.StatusCode.Or(0),
		Language:   ServerResponseLanguage(r.Language.Or("")),
		Body:       r.Body.Or(""),
		CreatedAt:  r.CreatedAt.Or(""),
		UpdatedAt:  r.UpdatedAt.Or(""),
		CreatedBy:  r.CreatedBy.Or(""),
		UpdatedBy:  r.UpdatedBy.Or(""),
		Mock:       r.Mock.Or(""),
	}
	for _, h := range r.Headers {
		out.Headers = append(out.Headers, ServerResponseHeader{Key: h.Key.Or(""), Value: h.Value.Or("")})
	}
	return out
}

// CreateServerResponseInput holds the fields for creating a server response.
type CreateServerResponseInput struct {
	// Name is the server response's name. Required.
	Name string
	// StatusCode is the HTTP status code to respond with. Required.
	StatusCode int
	Headers    []ServerResponseHeader
	Language   ServerResponseLanguage
	Body       string
}

// CreateServerResponse creates a server-level (5xx) response for a mock
// server. If set as active, all calls to the mock server return this
// response. Only one server response can be active at a time.
func (s *Service) CreateServerResponse(ctx context.Context, mockID string, in *CreateServerResponseInput) (*ServerResponse, error) {
	if in == nil {
		in = &CreateServerResponseInput{}
	}
	sr := api.CreateMockServerResponseServerResponse{
		Name:       in.Name,
		StatusCode: in.StatusCode,
	}
	for _, h := range in.Headers {
		sr.Headers = append(sr.Headers, api.CreateMockServerResponseServerResponseHeaders{
			Key:   api.NewOptString(h.Key),
			Value: api.NewOptString(h.Value),
		})
	}
	if in.Language != "" {
		sr.Language = api.NewOptNilCreateMockServerResponseServerResponseLanguage(
			api.CreateMockServerResponseServerResponseLanguage(in.Language))
	}
	if in.Body != "" {
		sr.Body = api.NewOptString(in.Body)
	}
	req := &api.CreateMockServerResponse{ServerResponse: api.NewOptCreateMockServerResponseServerResponse(sr)}

	res, err := s.api.CreateMockServerResponse(ctx, req, api.CreateMockServerResponseParams{MockId: mockID})
	if err != nil {
		return nil, err
	}
	return serverResponseFromRes(res)
}

// ServerResponse returns information about a mock server's server response.
func (s *Service) ServerResponse(ctx context.Context, mockID, serverResponseID string) (*ServerResponse, error) {
	res, err := s.api.GetMockServerResponse(ctx, api.GetMockServerResponseParams{
		MockId:           mockID,
		ServerResponseId: serverResponseID,
	})
	if err != nil {
		return nil, err
	}
	return serverResponseFromRes(res)
}

// UpdateServerResponseInput holds the fields for updating a server response.
type UpdateServerResponseInput struct {
	Name       string
	StatusCode int
	Headers    []ServerResponseHeader
	Language   ServerResponseLanguage
	Body       string
}

// UpdateServerResponse updates a mock server's server response.
func (s *Service) UpdateServerResponse(ctx context.Context, mockID, serverResponseID string, in *UpdateServerResponseInput) (*ServerResponse, error) {
	if in == nil {
		in = &UpdateServerResponseInput{}
	}
	sr := api.UpdateMockServerResponseServerResponse{}
	if in.Name != "" {
		sr.Name = api.NewOptString(in.Name)
	}
	if in.StatusCode != 0 {
		sr.StatusCode = api.NewOptInt(in.StatusCode)
	}
	for _, h := range in.Headers {
		sr.Headers = append(sr.Headers, api.UpdateMockServerResponseServerResponseHeaders{
			Key:   api.NewOptString(h.Key),
			Value: api.NewOptString(h.Value),
		})
	}
	if in.Language != "" {
		sr.Language = api.NewOptNilUpdateMockServerResponseServerResponseLanguage(
			api.UpdateMockServerResponseServerResponseLanguage(in.Language))
	}
	if in.Body != "" {
		sr.Body = api.NewOptString(in.Body)
	}
	req := &api.UpdateMockServerResponse{ServerResponse: api.NewOptUpdateMockServerResponseServerResponse(sr)}

	res, err := s.api.UpdateMockServerResponse(ctx, req, api.UpdateMockServerResponseParams{
		MockId:           mockID,
		ServerResponseId: serverResponseID,
	})
	if err != nil {
		return nil, err
	}
	return serverResponseFromRes(res)
}

func serverResponseFromRes(res any) (*ServerResponse, error) {
	switch r := res.(type) {
	case *api.CreateMockServerResponseOkResponse:
		return serverResponseFromOkResponse(*r), nil
	case *api.Common400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	case *api.GetMockServerResponsesNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// DeleteServerResponse deletes a mock server's server response.
func (s *Service) DeleteServerResponse(ctx context.Context, mockID, serverResponseID string) (*ServerResponse, error) {
	res, err := s.api.DeleteMockServerResponse(ctx, api.DeleteMockServerResponseParams{
		MockId:           mockID,
		ServerResponseId: serverResponseID,
	})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ServerResponseDeleted:
		out := &ServerResponse{
			ID:         r.ID.Or(""),
			Name:       r.Name.Or(""),
			StatusCode: r.StatusCode.Or(0),
			Language:   ServerResponseLanguage(r.Language.Or("")),
			Body:       r.Body.Or(""),
			CreatedAt:  r.CreatedAt.Or(""),
			CreatedBy:  r.CreatedBy.Or(""),
			UpdatedBy:  r.UpdatedBy.Or(""),
		}
		for _, h := range r.Headers {
			out.Headers = append(out.Headers, ServerResponseHeader{Key: h.Key.Or(""), Value: h.Value.Or("")})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	case *api.GetMockServerResponsesNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
