// Package privateapinetwork provides a high-level client for Postman's
// Private API Network API.
//
// The Private API Network lets a team publish and discover internal API
// workspaces, and manage requests from members to add their workspaces to it.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	elements, _ := client.PrivateAPINetwork().List(ctx, nil)
package privateapinetwork

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// SortField controls which timestamp List and ListAddRequests results are
// ordered by. Used together with a SortDirection.
type SortField string

// SortField values.
const (
	SortFieldCreatedAt SortField = "createdAt"
	SortFieldUpdatedAt SortField = "updatedAt"
)

// SortDirection controls ascending or descending order. Used together with a
// SortField.
type SortDirection string

// SortDirection values.
const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// RequestStatus filters Private API Network add requests by status.
type RequestStatus string

// RequestStatus values.
const (
	RequestStatusPending RequestStatus = "pending"
	RequestStatusDenied  RequestStatus = "denied"
)

// Decision is the approve/deny decision passed to RespondAddRequest.
type Decision string

// Decision values.
const (
	DecisionApproved Decision = "approved"
	DecisionDenied   Decision = "denied"
)

// Service is the high-level Private API Network client. Obtain one via
// postman.Client.PrivateAPINetwork.
type Service struct {
	api *api.Client
}

// New creates a Private API Network service over the given generated API
// client. Most callers should use postman.Client.PrivateAPINetwork instead of
// calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Element is a workspace published to the team's Private API Network.
type Element struct {
	ID             string
	Name           string
	Type           string
	Summary        string
	Description    string
	Href           string
	ParentFolderID int
	AddedAt        string
	AddedBy        int
	CreatedAt      string
	CreatedBy      int
	UpdatedAt      string
	UpdatedBy      int
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// Name, Summary, and Description filter to workspaces whose corresponding
	// field includes the given value. Matching is not case-sensitive.
	Name        string
	Summary     string
	Description string
	// Since returns only results created since this time, in RFC 3339 format.
	Since string
	// Until returns only results created until this time, in RFC 3339 format.
	Until string
	// AddedBy limits results to workspaces published by the given user ID.
	AddedBy int
	// Sort and Direction together control result ordering; both must be set
	// to have an effect.
	Sort      SortField
	Direction SortDirection
	// CreatedBy limits results to workspaces created by the given user ID.
	CreatedBy int
	// Offset is the zero-based offset of the first item to return.
	Offset int
	// Limit is the maximum number of results to return, up to 1000.
	Limit int
}

// ListResult is the set of workspaces in the team's Private API Network.
type ListResult struct {
	Elements   []Element
	Offset     int
	TotalCount int
}

// List returns the workspaces added to your team's
// [Private API Network](https://learning.postman.com/docs/collaborating-in-postman/adding-private-network/).
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.ListPrivateNetworkWorkspacesParams{}
	if in.Name != "" {
		params.Name = api.NewOptString(in.Name)
	}
	if in.Summary != "" {
		params.Summary = api.NewOptString(in.Summary)
	}
	if in.Description != "" {
		params.Description = api.NewOptString(in.Description)
	}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}
	if in.AddedBy > 0 {
		params.AddedBy = api.NewOptInt(in.AddedBy)
	}
	if in.Sort != "" {
		params.Sort = api.NewOptSortCreatedUpdatedAt(api.SortCreatedUpdatedAt(in.Sort))
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}
	if in.CreatedBy > 0 {
		params.CreatedBy = api.NewOptInt(in.CreatedBy)
	}
	if in.Offset > 0 {
		params.Offset = api.NewOptInt(in.Offset)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.ListPrivateNetworkWorkspaces(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ListPrivateNetworkWorkspacesOkResponse:
		out := &ListResult{}
		for _, e := range r.Elements {
			out.Elements = append(out.Elements, Element{
				ID:             e.ID.Or(""),
				Name:           e.Name.Or(""),
				Type:           e.Type.Or(""),
				Summary:        e.Summary.Or(""),
				Description:    e.Description.Or(""),
				Href:           e.Href.Or(""),
				ParentFolderID: e.ParentFolderId.Or(0),
				AddedAt:        e.AddedAt.Or(""),
				AddedBy:        e.AddedBy.Or(0),
				CreatedAt:      e.CreatedAt.Or(""),
				CreatedBy:      e.CreatedBy.Or(0),
				UpdatedAt:      e.UpdatedAt.Or(""),
				UpdatedBy:      e.UpdatedBy.Or(0),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Offset = meta.Offset.Or(0)
			out.TotalCount = meta.TotalCount.Or(0)
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Add --------------------------------------------------------------

// AddInput identifies the workspace to publish to the Private API Network.
type AddInput struct {
	// WorkspaceID is the ID of the workspace to publish. Required.
	WorkspaceID string
	// ParentFolderID is deprecated by the Postman API; leave at zero.
	ParentFolderID int
}

// Add publishes a workspace to your team's
// [Private API Network](https://learning.postman.com/docs/collaborating-in-postman/adding-private-network/).
func (s *Service) Add(ctx context.Context, in *AddInput) (*Element, error) {
	if in == nil {
		in = &AddInput{}
	}
	workspace := api.AddWorkspaceWorkspace{ID: in.WorkspaceID}
	if in.ParentFolderID > 0 {
		workspace.ParentFolderId = api.NewOptInt(in.ParentFolderID)
	}
	req := &api.AddWorkspace{Workspace: workspace}

	res, err := s.api.AddWorkspaceToPrivateNetwork(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ElementCreated:
		return &Element{
			ID:             r.ID.Or(""),
			Name:           r.Name.Or(""),
			Type:           string(r.Type.Or("")),
			Summary:        r.Summary.Or(""),
			Description:    r.Description.Or(""),
			Href:           r.Href.Or(""),
			ParentFolderID: r.ParentFolderId.Or(0),
			AddedAt:        r.AddedAt.Or(""),
			AddedBy:        r.AddedBy.Or(0),
			CreatedAt:      r.CreatedAt.Or(""),
			CreatedBy:      r.CreatedBy.Or(0),
			UpdatedAt:      r.UpdatedAt.Or(""),
			UpdatedBy:      r.UpdatedBy.Or(0),
		}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.AddWorkspaceToPrivateNetworkNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Remove --------------------------------------------------------------

// RemoveResult confirms a workspace was removed from the Private API Network.
type RemoveResult struct {
	WorkspaceID string
}

// Remove removes a workspace from your team's
// [Private API Network](https://learning.postman.com/docs/collaborating-in-postman/adding-private-network/).
// This does not delete the workspace; it only removes it from the Private API
// Network folder.
func (s *Service) Remove(ctx context.Context, workspaceID string) (*RemoveResult, error) {
	params := api.RemoveWorkspaceFromPrivateNetworkParams{WorkspaceId: workspaceID}

	res, err := s.api.RemoveWorkspaceFromPrivateNetwork(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.RemoveWorkspaceFromPrivateNetworkOkResponse:
		out := &RemoveResult{}
		if ws, ok := r.Workspace.Get(); ok {
			out.WorkspaceID = ws.ID.Or("")
		}
		return out, nil
	case *api.Pan400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.AddWorkspaceToPrivateNetworkNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateFolder --------------------------------------------------------------

// UpdateFolderInput moves a workspace to a different Private API Network
// folder.
type UpdateFolderInput struct {
	// ParentFolderID is the ID of the destination folder.
	ParentFolderID int
}

// UpdateFolder moves a workspace between Private API Network folders.
//
// Deprecated: this endpoint is deprecated in the Postman API.
func (s *Service) UpdateFolder(ctx context.Context, workspaceID string, in *UpdateFolderInput) error {
	if in == nil {
		in = &UpdateFolderInput{}
	}
	workspace := api.UpdateWorkspaceWorkspace2{}
	if in.ParentFolderID > 0 {
		workspace.ParentFolderId = api.NewOptInt(in.ParentFolderID)
	}
	req := &api.UpdatePanElementOrFolderRequest{Workspace: api.NewOptUpdateWorkspaceWorkspace2(workspace)}
	params := api.UpdatePanElementOrFolderParams{WorkspaceId: workspaceID}

	res, err := s.api.UpdatePanElementOrFolder(ctx, req, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.UpdatePanElementOrFolderOK:
		_ = r
		return nil
	case *api.Common401Error:
		return postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.AddWorkspaceToPrivateNetworkNotFoundResponse:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- ListAddRequests --------------------------------------------------------------

// RequestElement is the workspace an AddRequest is about.
type RequestElement struct {
	ID          string
	Type        string
	Name        string
	Summary     string
	Description string
}

// RequestResponse is the manager's response to an AddRequest, once decided.
type RequestResponse struct {
	CreatedAt string
	CreatedBy int
	Message   string
}

// AddRequest is a member's request to add a workspace to the team's Private
// API Network.
type AddRequest struct {
	ID        int
	CreatedAt string
	CreatedBy int
	Message   string
	Status    RequestStatus
	Element   RequestElement
	Response  RequestResponse
}

// ListAddRequestsInput holds the filters and pagination options for
// ListAddRequests.
type ListAddRequestsInput struct {
	// Since returns only results created since this time, in RFC 3339 format.
	Since string
	// Until returns only results created until this time, in RFC 3339 format.
	Until string
	// RequestedBy limits results to requests filed by the given user ID.
	RequestedBy int
	// Status filters by the request status.
	Status RequestStatus
	// Name filters to workspaces whose name includes the given value.
	// Matching is not case-sensitive.
	Name string
	// Sort and Direction together control result ordering; both must be set
	// to have an effect.
	Sort      SortField
	Direction SortDirection
	// Offset is the zero-based offset of the first item to return.
	Offset int
	// Limit is the maximum number of results to return, up to 1000.
	Limit int
}

// ListAddRequestsResult is the set of pending and decided Private API Network
// add requests.
type ListAddRequestsResult struct {
	Requests   []AddRequest
	Offset     int
	TotalCount int
}

// ListAddRequests gets all requests to add workspaces to your team's
// [Private API Network](https://learning.postman.com/docs/collaborating-in-postman/adding-private-network/).
func (s *Service) ListAddRequests(ctx context.Context, in *ListAddRequestsInput) (*ListAddRequestsResult, error) {
	if in == nil {
		in = &ListAddRequestsInput{}
	}
	params := api.ListPrivateNetworkAddRequestsParams{}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}
	if in.RequestedBy > 0 {
		params.RequestedBy = api.NewOptInt(in.RequestedBy)
	}
	if in.Status != "" {
		params.Status = api.NewOptPanRequestStatus(api.PanRequestStatus(in.Status))
	}
	if in.Name != "" {
		params.Name = api.NewOptString(in.Name)
	}
	if in.Sort != "" {
		params.Sort = api.NewOptSortCreatedUpdatedAt(api.SortCreatedUpdatedAt(in.Sort))
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}
	if in.Offset > 0 {
		params.Offset = api.NewOptInt(in.Offset)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.ListPrivateNetworkAddRequests(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ListPrivateNetworkAddRequestsOkResponse:
		out := &ListAddRequestsResult{}
		for _, req := range r.Requests {
			ar := AddRequest{
				ID:        req.ID.Or(0),
				CreatedAt: req.CreatedAt.Or(""),
				CreatedBy: req.CreatedBy.Or(0),
				Message:   req.Message.Or(""),
				Status:    RequestStatus(req.Status.Or("")),
			}
			if el, ok := req.Element.Get(); ok {
				ar.Element = RequestElement{
					ID:          el.ID.Or(""),
					Type:        string(el.Type.Or("")),
					Name:        el.Name.Or(""),
					Summary:     el.Summary.Or(""),
					Description: el.Description.Or(""),
				}
			}
			if resp, ok := req.Response.Get(); ok {
				ar.Response = RequestResponse{
					CreatedAt: resp.CreatedAt.Or(""),
					CreatedBy: resp.CreatedBy.Or(0),
					Message:   resp.Message.Or(""),
				}
			}
			out.Requests = append(out.Requests, ar)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Offset = meta.Offset.Or(0)
			out.TotalCount = meta.TotalCount.Or(0)
		}
		return out, nil
	case *api.Pan400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- RespondAddRequest --------------------------------------------------------------

// RespondInput holds the approve/deny decision for RespondAddRequest.
type RespondInput struct {
	// Status is the decision: DecisionApproved or DecisionDenied. Required.
	Status Decision
	// Message is an optional note explaining the decision.
	Message string
}

// RespondResult is the raw decoded response body from RespondAddRequest.
//
// Known limitation: Postman's generated schema for this endpoint's response
// does not resolve to a concrete shape, so the body is exposed here as raw
// JSON rather than typed fields.
type RespondResult struct {
	Raw json.RawMessage
}

// RespondAddRequest responds to a member's request to add a workspace to your
// team's
// [Private API Network](https://learning.postman.com/docs/collaborating-in-postman/adding-private-network/).
// Only managers can approve or deny a request; once approved, the workspace
// appears in the team's Private API Network. RequestID is the numeric ID of
// the request.
func (s *Service) RespondAddRequest(ctx context.Context, requestID string, in *RespondInput) (*RespondResult, error) {
	if in == nil {
		in = &RespondInput{}
	}
	body := api.RespondPanElementAddRequestBody{
		Status: api.RespondPanElementAddRequestBodyStatus(in.Status),
	}
	if in.Message != "" {
		body.Response = api.NewOptRespondPanElementAddRequestBodyResponse1(
			api.RespondPanElementAddRequestBodyResponse1{Message: api.NewOptString(in.Message)},
		)
	}
	params := api.RespondPrivateNetworkAddRequestParams{RequestId: requestID}

	res, err := s.api.RespondPrivateNetworkAddRequest(ctx, &body, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.RespondPanElementAddRequest:
		return &RespondResult{Raw: json.RawMessage(*r)}, nil
	case *api.Pan400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
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
