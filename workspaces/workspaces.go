// Package workspaces provides a high-level client for Postman's Workspaces
// API.
//
// Workspaces group related collections, environments, mocks, monitors, and
// specs, and control who can access them. See
// https://learning.postman.com/docs/collaborating-in-postman/using-workspaces/creating-workspaces/.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	list, _ := client.Workspaces().List(ctx, nil)
//	ws, _ := client.Workspaces().Get(ctx, list.Workspaces[0].ID, nil)
package workspaces

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// WorkspaceType is the type of a workspace, and the equivalent value used for
// a workspace's visibility. Visibility determines who can access the
// workspace:
//   - personal — Only the owner can access the workspace.
//   - team — All team members can access the workspace.
//   - private — Only invited team members can access the workspace (Team and
//     Enterprise plans only).
//   - public — Everyone can access the workspace.
//   - partner — Only invited team members and partners can access the
//     workspace (Team and Enterprise plans only).
type WorkspaceType string

// WorkspaceType / visibility values.
const (
	WorkspaceTypePersonal WorkspaceType = "personal"
	WorkspaceTypeTeam     WorkspaceType = "team"
	WorkspaceTypePrivate  WorkspaceType = "private"
	WorkspaceTypePublic   WorkspaceType = "public"
	WorkspaceTypePartner  WorkspaceType = "partner"
)

// IncludeOption adds extra information to a workspace response.
type IncludeOption string

// Include option values.
const (
	// IncludeMocksDeactivated includes all deactivated mock servers in the response.
	IncludeMocksDeactivated IncludeOption = "mocks:deactivated"
	// IncludeSCIM returns the SCIM user IDs of the workspace creator and who last modified it.
	IncludeSCIM IncludeOption = "scim"
)

// ElementQueryType is the type of element used to filter List results by
// ElementID.
type ElementQueryType string

// Element query type values.
const (
	ElementQueryTypeCollection    ElementQueryType = "collection"
	ElementQueryTypeSpecification ElementQueryType = "specification"
)

// TransferElementType is the type of Postman element being transferred
// between workspaces.
type TransferElementType string

// Transfer element type values.
const (
	TransferElementTypeCollection  TransferElementType = "collection"
	TransferElementTypeEnvironment TransferElementType = "environment"
	TransferElementTypeAPI         TransferElementType = "api"
	TransferElementTypeFlow        TransferElementType = "flow"
	TransferElementTypeMock        TransferElementType = "mock"
	TransferElementTypeMonitor     TransferElementType = "monitor"
)

// GlobalVariableType is the type of a workspace global variable.
type GlobalVariableType string

// Global variable type values.
const (
	GlobalVariableTypeDefault GlobalVariableType = "default"
	GlobalVariableTypeSecret  GlobalVariableType = "secret"
)

// RolesPath identifies the kind of principal a role operation applies to.
type RolesPath string

// Roles path values.
const (
	RolesPathUser      RolesPath = "/user"
	RolesPathUsergroup RolesPath = "/usergroup"
	RolesPathPartner   RolesPath = "/partner"
)

// UpdateCategory categorizes a workspace update.
type UpdateCategory string

// Update category values.
const (
	UpdateCategoryImprovement    UpdateCategory = "improvement"
	UpdateCategoryNewFeature     UpdateCategory = "new_feature"
	UpdateCategoryBugFix         UpdateCategory = "bug_fix"
	UpdateCategoryBreakingChange UpdateCategory = "breaking_change"
	UpdateCategoryAnnouncement   UpdateCategory = "announcement"
)

// Service is the high-level Workspaces client. Obtain one via
// postman.Client.Workspaces.
type Service struct {
	api *api.Client
}

// New creates a Workspaces service over the given generated API client. Most
// callers should use postman.Client.Workspaces instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- ManagePartnerInvites ----------------------------------------------------

// ManagePartnerInvitesInput holds the request body for ManagePartnerInvites.
//
// Postman's OpenAPI schema for this endpoint's request body is a union of
// three shapes the generator cannot resolve statically (invite partners by
// email, remove partners from a workspace, or remove partners from a team
// partnership), so it collapses to free-form JSON. Body must therefore be one
// of:
//
//	{"action":"invite_partner","targetEntity":"workspace","targetEntityId":"<workspaceId>","roleId":"<roleId>","target":{"emails":["a@example.com"]}}
//	{"action":"remove_partner","targetEntity":"workspace","targetEntityId":"<workspaceId>","target":{...}}
//	{"action":"remove_partner","targetEntity":"team","targetEntityId":"<teamId>","target":{...}}
//
// See https://learning.postman.com/docs/collaborating-in-postman/using-workspaces/partner-workspaces/manage/.
type ManagePartnerInvitesInput struct {
	Body json.RawMessage
}

// ManagePartnerInvitesResult is the raw JSON response body, whose shape
// depends on the requested action (see ManagePartnerInvitesInput).
type ManagePartnerInvitesResult struct {
	Body json.RawMessage
}

// ManagePartnerInvites sends, removes, or revokes Partner Workspace
// invitations. Requires a Postman Team or Enterprise plan.
func (s *Service) ManagePartnerInvites(ctx context.Context, in *ManagePartnerInvitesInput) (*ManagePartnerInvitesResult, error) {
	if in == nil {
		in = &ManagePartnerInvitesInput{}
	}

	res, err := s.api.ManagePartnerWorkspaceInvites(ctx, api.ManagePartnerWorkspaceInvites(in.Body))
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ManagePartnerWorkspaceInvitesResponse:
		return &ManagePartnerInvitesResult{Body: json.RawMessage(*r)}, nil
	case *api.ManagePartnerWorkspaceInvitesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ManagePartnerWorkspaceInvitesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ManagePartnerWorkspaceInvitesForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.ManagePartnerWorkspaceInvitesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- List ---------------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// Type filters the response to workspaces of this type.
	Type WorkspaceType
	// CreatedBy returns only workspaces created by the given user ID.
	CreatedBy int
	// Include adds extra information to the response.
	Include IncludeOption
	// ElementType filters results to the workspace containing the given
	// element. Requires ElementID.
	ElementType ElementQueryType
	// ElementID filters results to the workspace containing the given
	// element's ID. Requires ElementType.
	ElementID string
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return, up to 100.
	Limit int
}

// WorkspaceSummary is a workspace as returned by List.
type WorkspaceSummary struct {
	ID            string
	Name          string
	Type          WorkspaceType
	Visibility    WorkspaceType
	CreatedBy     string
	About         string
	CreatedAt     string
	UpdatedAt     string
	ScimCreatedBy string
}

// ListResult is the paginated result of List.
type ListResult struct {
	Workspaces []WorkspaceSummary
	NextCursor string
}

// List returns the workspaces you have access to. A nil or empty input
// returns all results.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}

	params := api.GetWorkspacesParams{}
	if in.Type != "" {
		params.Type = api.NewOptWorkspaceTypeQuery(api.WorkspaceTypeQuery(in.Type))
	}
	if in.CreatedBy != 0 {
		params.CreatedBy = api.NewOptInt(in.CreatedBy)
	}
	if in.Include != "" {
		params.Include = api.NewOptWorkspaceIncludeQuery(api.WorkspaceIncludeQuery(in.Include))
	}
	if in.ElementType != "" {
		params.ElementType = api.NewOptWorkspaceElementTypeQuery(api.WorkspaceElementTypeQuery(in.ElementType))
	}
	if in.ElementID != "" {
		params.ElementId = api.NewOptString(in.ElementID)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetWorkspaces(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetWorkspacesOkResponse:
		out := &ListResult{}
		for _, w := range r.Workspaces {
			ws := WorkspaceSummary{
				ID:         w.ID.Or(""),
				Name:       w.Name.Or(""),
				Type:       WorkspaceType(w.Type.Or("")),
				Visibility: WorkspaceType(w.Visibility.Or("")),
				CreatedBy:  w.CreatedBy.Or(""),
				About:      w.About.Or(""),
				CreatedAt:  w.CreatedAt.Or(""),
				UpdatedAt:  w.UpdatedAt.Or(""),
			}
			if scim, ok := w.Scim.Get(); ok {
				ws.ScimCreatedBy = scim.CreatedBy.Or("")
			}
			out.Workspaces = append(out.Workspaces, ws)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.Workspaces400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create -------------------------------------------------------------------

// CreateInput holds the fields for creating a workspace.
type CreateInput struct {
	// Name is required.
	Name string
	// Type is required.
	Type        WorkspaceType
	Description string
	About       string
	// TeamID is required if Postman Organizations is enabled for the team.
	TeamID string
}

// CreateResult is returned after creating a workspace.
type CreateResult struct {
	ID   string
	Name string
}

// Create creates a new workspace.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*CreateResult, error) {
	if in == nil {
		in = &CreateInput{}
	}

	ws := api.CreateWorkspaceWorkspace{
		Name: in.Name,
		Type: api.CreateWorkspaceWorkspaceType(in.Type),
	}
	if in.Description != "" {
		ws.Description = api.NewOptString(in.Description)
	}
	if in.About != "" {
		ws.About = api.NewOptString(in.About)
	}
	if in.TeamID != "" {
		ws.TeamId = api.NewOptString(in.TeamID)
	}
	req := &api.CreateWorkspace{Workspace: api.NewOptCreateWorkspaceWorkspace(ws)}

	res, err := s.api.CreateWorkspace(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateWorkspaceOkResponse:
		out := &CreateResult{}
		if w, ok := r.Workspace.Get(); ok {
			out.ID = w.ID.Or("")
			out.Name = w.Name.Or("")
		}
		return out, nil
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Forbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- AllRoles -----------------------------------------------------------------

// WorkspaceRoleType describes a role that can be assigned in a workspace.
type WorkspaceRoleType struct {
	ID          string
	Description string
	DisplayName string
}

// AllRolesResult is the set of all roles available, grouped by principal kind.
type AllRolesResult struct {
	User      []WorkspaceRoleType
	Usergroup []WorkspaceRoleType
	Partner   []WorkspaceRoleType
}

// AllRoles returns information about all roles in a workspace, based on the
// team's plan.
func (s *Service) AllRoles(ctx context.Context) (*AllRolesResult, error) {
	res, err := s.api.GetAllWorkspaceRoles(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAllWorkspaceRolesOkResponse:
		out := &AllRolesResult{}
		if roles, ok := r.Roles.Get(); ok {
			out.User = workspaceRoleTypesFromAPI(roles.User)
			out.Usergroup = workspaceRoleTypesFromAPI(roles.Usergroup)
			out.Partner = workspaceRoleTypesFromAPI(roles.Partner)
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

func workspaceRoleTypesFromAPI(in []api.WorkspaceRoleData) []WorkspaceRoleType {
	out := make([]WorkspaceRoleType, 0, len(in))
	for _, r := range in {
		out = append(out, WorkspaceRoleType{
			ID:          r.ID.Or(""),
			Description: r.Description.Or(""),
			DisplayName: r.DisplayName.Or(""),
		})
	}
	return out
}

// --- Get ------------------------------------------------------------------

// GetInput holds the options for Get.
type GetInput struct {
	// Include adds extra information to the response.
	Include IncludeOption
}

// WorkspaceRef is a lightweight reference to a Postman element (collection,
// environment, or spec) in a workspace.
type WorkspaceRef struct {
	ID   string
	Name string
	UID  string
}

// WorkspaceMockRef is a lightweight reference to a mock or monitor in a
// workspace.
type WorkspaceMockRef struct {
	ID          string
	Name        string
	UID         string
	Deactivated bool
}

// GetResult is the full detail of a workspace, as returned by Get.
type GetResult struct {
	ID            string
	Name          string
	Type          WorkspaceType
	Visibility    WorkspaceType
	Description   string
	CreatedBy     string
	UpdatedBy     string
	CreatedAt     string
	UpdatedAt     string
	About         string
	Collections   []WorkspaceRef
	Environments  []WorkspaceRef
	Mocks         []WorkspaceMockRef
	Monitors      []WorkspaceMockRef
	Specs         []WorkspaceRef
	ScimCreatedBy string
	ScimUpdatedBy string
}

// Get returns information about a workspace.
func (s *Service) Get(ctx context.Context, workspaceID string, in *GetInput) (*GetResult, error) {
	if in == nil {
		in = &GetInput{}
	}
	params := api.GetWorkspaceParams{WorkspaceId: workspaceID}
	if in.Include != "" {
		params.Include = api.NewOptWorkspaceIncludeQuery(api.WorkspaceIncludeQuery(in.Include))
	}

	res, err := s.api.GetWorkspace(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetWorkspaceOkResponse:
		out := &GetResult{}
		if w, ok := r.Workspace.Get(); ok {
			out.ID = w.ID.Or("")
			out.Name = w.Name.Or("")
			out.Type = WorkspaceType(w.Type.Or(""))
			out.Visibility = WorkspaceType(w.Visibility.Or(""))
			out.Description = w.Description.Or("")
			out.CreatedBy = w.CreatedBy.Or("")
			out.UpdatedBy = w.UpdatedBy.Or("")
			out.CreatedAt = w.CreatedAt.Or("")
			out.UpdatedAt = w.UpdatedAt.Or("")
			out.About = w.About.Or("")
			for _, c := range w.Collections {
				out.Collections = append(out.Collections, WorkspaceRef{ID: c.ID.Or(""), Name: c.Name.Or(""), UID: c.UID.Or("")})
			}
			for _, e := range w.Environments {
				out.Environments = append(out.Environments, WorkspaceRef{ID: e.ID.Or(""), Name: e.Name.Or(""), UID: e.UID.Or("")})
			}
			for _, m := range w.Mocks {
				out.Mocks = append(out.Mocks, WorkspaceMockRef{ID: m.ID.Or(""), Name: m.Name.Or(""), UID: m.UID.Or(""), Deactivated: m.Deactivated.Or(false)})
			}
			for _, m := range w.Monitors {
				out.Monitors = append(out.Monitors, WorkspaceMockRef{ID: m.ID.Or(""), Name: m.Name.Or(""), UID: m.UID.Or(""), Deactivated: m.Deactivated.Or(false)})
			}
			for _, sp := range w.Specs {
				out.Specs = append(out.Specs, WorkspaceRef{ID: sp.ID.Or(""), Name: sp.Name.Or(""), UID: sp.UID.Or("")})
			}
			if scim, ok := w.Scim.Get(); ok {
				out.ScimCreatedBy = scim.CreatedBy.Or("")
				out.ScimUpdatedBy = scim.UpdatedBy.Or("")
			}
		}
		return out, nil
	case *api.GetWorkspaceNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Update -----------------------------------------------------------------

// UpdateInput holds the fields to update on a workspace. Zero-value fields
// are left unchanged.
type UpdateInput struct {
	Name        string
	Type        WorkspaceType
	Description string
	About       string
}

// UpdateResult is returned after updating a workspace.
type UpdateResult struct {
	ID   string
	Name string
}

// Update updates a workspace's name, type/visibility, description, or about
// text.
//
// This endpoint does not support changing visibility from private to public,
// public to private, or (on Free and Solo plans) private to personal; nor
// public to personal for team users.
func (s *Service) Update(ctx context.Context, workspaceID string, in *UpdateInput) (*UpdateResult, error) {
	if in == nil {
		in = &UpdateInput{}
	}

	ws := api.UpdateWorkspaceWorkspace1{}
	if in.Name != "" {
		ws.Name = api.NewOptString(in.Name)
	}
	if in.Type != "" {
		ws.Type = api.NewOptUpdateWorkspaceWorkspaceType(api.UpdateWorkspaceWorkspaceType(in.Type))
	}
	if in.Description != "" {
		ws.Description = api.NewOptString(in.Description)
	}
	if in.About != "" {
		ws.About = api.NewOptString(in.About)
	}
	req := &api.UpdateWorkspaceRequest{Workspace: api.NewOptUpdateWorkspaceWorkspace1(ws)}
	params := api.UpdateWorkspaceParams{WorkspaceId: workspaceID}

	res, err := s.api.UpdateWorkspace(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceUpdated:
		out := &UpdateResult{}
		if w, ok := r.Workspace.Get(); ok {
			out.ID = w.ID.Or("")
			out.Name = w.Name.Or("")
		}
		return out, nil
	case *api.UpdateWorkspaceBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateWorkspaceForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdateWorkspaceNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete -------------------------------------------------------------------

// DeleteResult is returned after deleting a workspace.
type DeleteResult struct {
	ID string
}

// Delete deletes a workspace.
func (s *Service) Delete(ctx context.Context, workspaceID string) (*DeleteResult, error) {
	params := api.DeleteWorkspaceParams{WorkspaceId: workspaceID}

	res, err := s.api.DeleteWorkspace(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceDeleted:
		out := &DeleteResult{}
		if w, ok := r.Workspace.Get(); ok {
			out.ID = w.ID.Or("")
		}
		return out, nil
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- ActivityFeed -------------------------------------------------------------

// ActivityFeedInput holds the filters and pagination options for
// ActivityFeed.
type ActivityFeedInput struct {
	// UserID filters results by the given user ID.
	UserID string
	// ElementType is a comma-separated list of elements to filter the results by.
	ElementType string
	// Limit is the maximum number of rows to return.
	Limit int
	// Cursor is the pagination cursor (use ActivityFeedResult.NextCursor to page).
	Cursor string
}

// ActivityFeedUser identifies the user behind an activity feed entry.
type ActivityFeedUser struct {
	ID        int
	Username  string
	IsPartner bool
	Name      string
}

// ActivityFeedEntry is a single event in a workspace's activity feed.
type ActivityFeedEntry struct {
	WorkspaceID string
	CreatedAt   string
	UpdatedAt   string
	ID          int
	User        ActivityFeedUser
	// Action is one of "create", "update", or "destroy".
	Action      string
	ElementType string
	Trigger     string
	ElementID   string
	ElementName string
}

// ActivityFeedResult is the paginated activity feed returned by ActivityFeed.
type ActivityFeedResult struct {
	Entries    []ActivityFeedEntry
	NextCursor string
}

// ActivityFeed returns information about who added or removed collections,
// environments, or other elements from a workspace, and users that joined or
// left it.
func (s *Service) ActivityFeed(ctx context.Context, workspaceID string, in *ActivityFeedInput) (*ActivityFeedResult, error) {
	if in == nil {
		in = &ActivityFeedInput{}
	}
	params := api.GetWorkspaceActivityFeedParams{WorkspaceId: workspaceID}
	if in.UserID != "" {
		params.UserId = api.NewOptString(in.UserID)
	}
	if in.ElementType != "" {
		params.ElementType = api.NewOptString(in.ElementType)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}

	res, err := s.api.GetWorkspaceActivityFeed(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceActivityFeed:
		out := &ActivityFeedResult{}
		for _, d := range r.Data {
			entry := ActivityFeedEntry{
				WorkspaceID: d.WorkspaceId.Or(""),
				CreatedAt:   d.CreatedAt.Or(""),
				UpdatedAt:   d.UpdatedAt.Or(""),
				ID:          d.ID.Or(0),
				Action:      string(d.Action.Or("")),
				ElementType: d.ElementType.Or(""),
				Trigger:     d.Trigger.Or(""),
				ElementID:   d.ElementId.Or(""),
				ElementName: d.ElementName.Or(""),
			}
			if u, ok := d.User.Get(); ok {
				entry.User = ActivityFeedUser{
					ID:        u.ID.Or(0),
					Username:  u.Username.Or(""),
					IsPartner: u.IsPartner.Or(false),
					Name:      u.Name.Or(""),
				}
			}
			out.Entries = append(out.Entries, entry)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetWorkspaceActivityFeedBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetWorkspaceActivityFeedNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- TransferElement -----------------------------------------------------------

// TransferElementInput holds the fields for transferring a Postman element
// to another workspace.
type TransferElementInput struct {
	// ElementID is the ID of the element to transfer. For collections, this
	// is the collection's unique ID (userId-collection).
	ElementID string
	// ElementType is required.
	ElementType TransferElementType
	// To is the destination workspace's ID.
	To string
}

// TransferElementResult describes the result of a single element transfer.
type TransferElementResult struct {
	Type string
	From string
	ID   string
	To   string
}

// TransferElement transfers a collection, environment, mock, monitor, or
// Flows module/action from one workspace to another. Transfers from team
// workspaces to personal workspaces are not supported.
func (s *Service) TransferElement(ctx context.Context, workspaceID string, in *TransferElementInput) (*TransferElementResult, error) {
	if in == nil {
		in = &TransferElementInput{}
	}
	idJSON, err := json.Marshal(in.ElementID)
	if err != nil {
		return nil, err
	}
	req := &api.TransferWorkspaceElement{
		ID:   api.ID(idJSON),
		Type: api.TransferWorkspaceElementType(in.ElementType),
		To:   in.To,
	}
	params := api.TransferWorkspaceElementParams{WorkspaceId: workspaceID}

	res, err := s.api.TransferWorkspaceElement(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TransferWorkspaceElementResponse:
		out := &TransferElementResult{}
		if w, ok := r.Workspace.Get(); ok {
			if t, ok := w.ElementTransfers.Get(); ok {
				out.Type = t.Type.Or("")
				out.From = t.From.Or("")
				out.ID = t.ID.Or("")
				out.To = t.To.Or("")
			}
		}
		return out, nil
	case *api.TransferWorkspaceElementBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.TransferWorkspaceElementForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.TransferWorkspaceElementNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GlobalVariables ------------------------------------------------------------

// GlobalVariable is a workspace-scoped variable, available throughout the
// workspace across collections, requests, scripts, and environments.
type GlobalVariable struct {
	Key         string
	Type        GlobalVariableType
	Value       string
	Enabled     bool
	Description string
}

// GlobalVariablesResult is the set of a workspace's global variables.
type GlobalVariablesResult struct {
	Values []GlobalVariable
}

// GlobalVariables returns a workspace's global variables.
func (s *Service) GlobalVariables(ctx context.Context, workspaceID string) (*GlobalVariablesResult, error) {
	params := api.GetWorkspaceGlobalVariablesParams{WorkspaceId: workspaceID}

	res, err := s.api.GetWorkspaceGlobalVariables(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetWorkspaceGlobalVariablesOkResponse:
		return &GlobalVariablesResult{Values: globalVariablesFromAPI(r.Values)}, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func globalVariablesFromAPI(in []api.GlobalVariableInfo) []GlobalVariable {
	out := make([]GlobalVariable, 0, len(in))
	for _, v := range in {
		gv := GlobalVariable{
			Key:     v.Key.Or(""),
			Type:    GlobalVariableType(v.Type.Or("")),
			Value:   v.Value.Or(""),
			Enabled: v.Enabled.Or(false),
		}
		// Description is a raw JSON field (see UpdateGlobalVariablesInput);
		// decoding is best-effort since a malformed value should not fail
		// the whole read.
		var desc string
		if len(v.Description) > 0 {
			_ = json.Unmarshal(v.Description, &desc)
		}
		gv.Description = desc
		out = append(out, gv)
	}
	return out
}

// UpdateGlobalVariablesInput holds the new set of global variables. This
// replaces all existing global variables in the workspace.
type UpdateGlobalVariablesInput struct {
	Values []GlobalVariable
}

// UpdateGlobalVariables replaces all of a workspace's global variables.
func (s *Service) UpdateGlobalVariables(ctx context.Context, workspaceID string, in *UpdateGlobalVariablesInput) (*GlobalVariablesResult, error) {
	if in == nil {
		in = &UpdateGlobalVariablesInput{}
	}

	req := &api.UpdateGlobalVariables{}
	for _, v := range in.Values {
		descJSON, err := json.Marshal(v.Description)
		if err != nil {
			return nil, err
		}
		item := api.GlobalVariableInfo{Description: descJSON}
		if v.Key != "" {
			item.Key = api.NewOptString(v.Key)
		}
		if v.Type != "" {
			item.Type = api.NewOptGlobalVariableInfoType(api.GlobalVariableInfoType(v.Type))
		}
		if v.Value != "" {
			item.Value = api.NewOptString(v.Value)
		}
		item.Enabled = api.NewOptBool(v.Enabled)
		req.Values = append(req.Values, item)
	}
	params := api.UpdateWorkspaceGlobalVariablesParams{WorkspaceId: workspaceID}

	res, err := s.api.UpdateWorkspaceGlobalVariables(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GlobalVariablesUpdated:
		return &GlobalVariablesResult{Values: globalVariablesFromAPI(r.Values)}, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Roles ----------------------------------------------------------------

// RolesInput holds the options for Roles.
type RolesInput struct {
	// IncludeSCIM returns IDs as SCIM user and group IDs instead of Postman IDs.
	IncludeSCIM bool
}

// RoleAssignment lists the principals of one kind (users, user groups, or
// partners) assigned each role.
type RoleAssignment struct {
	ID          string
	User        []string
	Usergroup   []string
	Partner     []string
	DisplayName string
}

// RolesResult is the roles assigned in a workspace, as returned by Roles.
type RolesResult struct {
	Roles []RoleAssignment
}

// Roles returns the roles of users, user groups, and partners in a
// workspace. Partner roles don't support SCIM IDs.
func (s *Service) Roles(ctx context.Context, workspaceID string, in *RolesInput) (*RolesResult, error) {
	if in == nil {
		in = &RolesInput{}
	}
	params := api.GetWorkspaceRolesParams{WorkspaceId: workspaceID}
	if in.IncludeSCIM {
		params.Include = api.NewOptWorkspaceIncludeScimQuery(api.WorkspaceIncludeScimQueryScim)
	}

	res, err := s.api.GetWorkspaceRoles(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceRoles:
		out := &RolesResult{}
		for _, role := range r.Roles {
			out.Roles = append(out.Roles, RoleAssignment{
				ID:          role.ID.Or(""),
				User:        role.User,
				Usergroup:   role.Usergroup,
				Partner:     role.Partner,
				DisplayName: role.DisplayName.Or(""),
			})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateRoles ------------------------------------------------------------

// RoleChange assigns or removes a role for a single user, user group, or
// partner.
type RoleChange struct {
	// ID is the user, user group, or partner ID.
	ID string
	// Role is the role ID to assign.
	Role string
}

// RoleOperation is a single JSON Patch-style role change.
type RoleOperation struct {
	// Op is the patch operation, e.g. "add" or "remove".
	Op string
	// Path selects the kind of principal this operation applies to.
	Path  RolesPath
	Value []RoleChange
}

// UpdateRolesInput holds the role operations to apply. This endpoint is
// restricted to 50 operations per call; each operation must be unique per
// principal (for example, you cannot add and remove roles for the same user
// in one call).
type UpdateRolesInput struct {
	Operations []RoleOperation
}

// UpdatedRoleAssignment lists the principals assigned each role after an
// UpdateRoles call.
type UpdatedRoleAssignment struct {
	ID          string
	User        []string
	Group       []string
	DisplayName string
}

// UpdateRolesResult is returned after updating workspace roles.
type UpdateRolesResult struct {
	Roles []UpdatedRoleAssignment
}

// UpdateRoles updates the roles of users, user groups, or partners in a
// workspace. User groups require a Postman Enterprise plan. This endpoint
// doesn't support the external Guest role, and doesn't support updating
// partner and user roles in the same call.
//
// Postman's identifierType header (to address principals by SCIM ID) is not
// exposed here: the generated OpenAPI spec dropped this parameter, so the
// underlying client has no way to send it.
func (s *Service) UpdateRoles(ctx context.Context, workspaceID string, in *UpdateRolesInput) (*UpdateRolesResult, error) {
	if in == nil {
		in = &UpdateRolesInput{}
	}

	req := &api.UpdateWorkspaceRoles{}
	for _, op := range in.Operations {
		item := api.UpdateWorkspaceRolesRoles{
			Op:   op.Op,
			Path: api.UpdateWorkspaceRolesRolesPath(op.Path),
		}
		for _, v := range op.Value {
			item.Value = append(item.Value, api.UpdateWorkspaceRolesRolesValue{ID: v.ID, Role: v.Role})
		}
		req.Roles = append(req.Roles, item)
	}
	params := api.UpdateWorkspaceRolesParams{WorkspaceId: workspaceID}

	res, err := s.api.UpdateWorkspaceRoles(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceRolesUpdated:
		out := &UpdateRolesResult{}
		for _, role := range r.Roles {
			out.Roles = append(out.Roles, UpdatedRoleAssignment{
				ID:          role.ID.Or(""),
				User:        role.User,
				Group:       role.Group,
				DisplayName: role.DisplayName.Or(""),
			})
		}
		return out, nil
	case *api.PartnerAndPersonalWorkspaceRolesUnsupported:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnprocessableEntity)
	case *api.UpdateWorkspaceRolesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateWorkspaceRolesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- TransferToTeam -----------------------------------------------------------

// TransferToTeamInput holds the source and destination teams for a workspace
// transfer.
type TransferToTeamInput struct {
	// Source is the source team's ID.
	Source string
	// Destination is the destination team's ID.
	Destination string
}

// TransferToTeamResult is returned after transferring a workspace between
// teams.
type TransferToTeamResult struct {
	ID          string
	Source      string
	Destination string
}

// TransferToTeam transfers a workspace from one team to another. Requires
// Postman Enterprise with Postman Organizations enabled. Team user roles are
// modified as part of the transfer: a user who has a role in the source team
// but not the destination team loses their role on the workspace.
func (s *Service) TransferToTeam(ctx context.Context, workspaceID string, in *TransferToTeamInput) (*TransferToTeamResult, error) {
	if in == nil {
		in = &TransferToTeamInput{}
	}
	req := &api.TransferWorkspaceToTeam{Source: in.Source, Destination: in.Destination}
	params := api.TransferWorkspaceToTeamParams{WorkspaceId: workspaceID}

	res, err := s.api.TransferWorkspaceToTeam(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TransferWorkspaceToTeamResponse:
		out := &TransferToTeamResult{}
		if w, ok := r.Workspace.Get(); ok {
			if t, ok := w.Transfer.Get(); ok {
				out.ID = t.ID.Or("")
				out.Source = t.Source.Or("")
				out.Destination = t.Destination.Or("")
			}
		}
		return out, nil
	case *api.TransferWorkspaceToTeamBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.TransferWorkspaceToTeamUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TransferWorkspaceToTeamForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.TransferWorkspaceToTeamNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.TransferWorkspaceToTeamInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Updates ------------------------------------------------------------------

// UpdatesInput holds the filters and pagination options for Updates.
type UpdatesInput struct {
	// Cursor is the pagination cursor (use UpdatesResult.NextCursor to page).
	Cursor string
	// Category is a comma-separated list of categories to filter results by.
	Category string
}

// WorkspaceUpdateAuthor identifies who authored a workspace update, as
// returned by Updates.
type WorkspaceUpdateAuthor struct {
	ID       int
	Name     string
	Username string
}

// WorkspaceUpdateEntry is a single workspace update, as returned by Updates.
type WorkspaceUpdateEntry struct {
	ID               int
	Topic            string
	Description      string
	WorkspaceID      string
	CreatedBy        WorkspaceUpdateAuthor
	UpdatedBy        int
	CreatedAt        string
	UpdatedAt        string
	Category         UpdateCategory
	IsPinned         bool
	RelatedResources json.RawMessage
}

// UpdatesResult is the paginated list of workspace updates returned by
// Updates.
type UpdatesResult struct {
	Updates    []WorkspaceUpdateEntry
	NextCursor string
}

// Updates returns the workspace updates posted in the given workspace.
// Workspace updates keep watchers informed about changes such as new
// features, bug fixes, breaking changes, and announcements.
func (s *Service) Updates(ctx context.Context, workspaceID string, in *UpdatesInput) (*UpdatesResult, error) {
	if in == nil {
		in = &UpdatesInput{}
	}
	params := api.GetWorkspaceUpdatesParams{WorkspaceId: workspaceID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Category != "" {
		params.Category = api.NewOptString(in.Category)
	}

	res, err := s.api.GetWorkspaceUpdates(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetWorkspaceUpdates:
		out := &UpdatesResult{}
		for _, d := range r.Data {
			entry := WorkspaceUpdateEntry{
				ID:               d.ID.Or(0),
				Topic:            d.Topic.Or(""),
				Description:      d.Description.Or(""),
				WorkspaceID:      d.WorkspaceId.Or(""),
				UpdatedBy:        d.UpdatedBy.Or(0),
				CreatedAt:        d.CreatedAt.Or(""),
				UpdatedAt:        d.UpdatedAt.Or(""),
				Category:         UpdateCategory(d.Category.Or("")),
				IsPinned:         d.IsPinned.Or(false),
				RelatedResources: json.RawMessage(d.RelatedResources),
			}
			if cb, ok := d.CreatedBy.Get(); ok {
				entry.CreatedBy = WorkspaceUpdateAuthor{ID: cb.ID.Or(0), Name: cb.Name.Or(""), Username: cb.Username.Or("")}
			}
			out.Updates = append(out.Updates, entry)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetWorkspaceUpdatesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetWorkspaceUpdatesForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetWorkspaceUpdatesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CreateUpdate / PatchUpdate ------------------------------------------------

// WorkspaceUpdateRecord is a workspace update as returned by CreateUpdate and
// PatchUpdate.
type WorkspaceUpdateRecord struct {
	ID               int
	Topic            string
	Description      string
	WorkspaceID      string
	CreatedBy        int
	UpdatedBy        int
	CreatedAt        string
	UpdatedAt        string
	Category         UpdateCategory
	IsPinned         bool
	RelatedResources json.RawMessage
}

func workspaceUpdateRecordFromAPI(r *api.WorkspaceUpdatePostPatchResponseData) *WorkspaceUpdateRecord {
	return &WorkspaceUpdateRecord{
		ID:               r.ID.Or(0),
		Topic:            r.Topic.Or(""),
		Description:      r.Description.Or(""),
		WorkspaceID:      r.WorkspaceId.Or(""),
		CreatedBy:        r.CreatedBy.Or(0),
		UpdatedBy:        r.UpdatedBy.Or(0),
		CreatedAt:        r.CreatedAt.Or(""),
		UpdatedAt:        r.UpdatedAt.Or(""),
		Category:         UpdateCategory(r.Category.Or("")),
		IsPinned:         r.IsPinned.Or(false),
		RelatedResources: json.RawMessage(r.RelatedResources),
	}
}

// CreateUpdateInput holds the fields for posting a new workspace update.
type CreateUpdateInput struct {
	Description string
	Topic       string
	// Category is required.
	Category UpdateCategory
	// RelatedResources is free-form JSON describing resources related to
	// this update (its shape is not resolvable from Postman's published
	// schema). A nil value is sent as an empty array.
	RelatedResources json.RawMessage
}

// CreateUpdate posts a new workspace update in the given workspace.
func (s *Service) CreateUpdate(ctx context.Context, workspaceID string, in *CreateUpdateInput) (*WorkspaceUpdateRecord, error) {
	if in == nil {
		in = &CreateUpdateInput{}
	}
	descJSON, err := json.Marshal(in.Description)
	if err != nil {
		return nil, err
	}
	topicJSON, err := json.Marshal(in.Topic)
	if err != nil {
		return nil, err
	}
	req := &api.CreateWorkspaceUpdate{
		Description:      descJSON,
		Topic:            topicJSON,
		Category:         api.WorkspaceUpdateCategoryData(in.Category),
		RelatedResources: relatedResourcesJSON(in.RelatedResources),
	}
	params := api.CreateWorkspaceUpdateParams{WorkspaceId: workspaceID}

	res, err := s.api.CreateWorkspaceUpdate(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceUpdatePostPatchResponseData:
		return workspaceUpdateRecordFromAPI(r), nil
	case *api.CreateWorkspaceUpdateBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.CreateWorkspaceUpdateForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.CreateWorkspaceUpdateNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.CreateWorkspaceUpdateUnprocessableEntity:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnprocessableEntity)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func relatedResourcesJSON(v json.RawMessage) []byte {
	if len(v) == 0 {
		return []byte("[]")
	}
	return v
}

// --- GetUpdate --------------------------------------------------------------

// GetUpdateResult is a workspace update as returned by GetUpdate.
type GetUpdateResult struct {
	ID               int
	Topic            string
	Description      string
	WorkspaceID      string
	CreatedBy        WorkspaceUpdateAuthor
	UpdatedBy        int
	CreatedAt        string
	UpdatedAt        string
	Category         UpdateCategory
	IsPinned         bool
	RelatedResources json.RawMessage
}

// GetUpdate returns information about a single workspace update.
func (s *Service) GetUpdate(ctx context.Context, workspaceID string, updateID int) (*GetUpdateResult, error) {
	params := api.GetWorkspaceUpdateParams{WorkspaceId: workspaceID, UpdateId: itoa(updateID)}

	res, err := s.api.GetWorkspaceUpdate(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceUpdateData:
		out := &GetUpdateResult{
			ID:               r.ID.Or(0),
			Topic:            r.Topic.Or(""),
			Description:      r.Description.Or(""),
			WorkspaceID:      r.WorkspaceId.Or(""),
			UpdatedBy:        r.UpdatedBy.Or(0),
			CreatedAt:        r.CreatedAt.Or(""),
			UpdatedAt:        r.UpdatedAt.Or(""),
			Category:         UpdateCategory(r.Category.Or("")),
			IsPinned:         r.IsPinned.Or(false),
			RelatedResources: json.RawMessage(r.RelatedResources),
		}
		if cb, ok := r.CreatedBy.Get(); ok {
			out.CreatedBy = WorkspaceUpdateAuthor{ID: cb.ID.Or(0), Name: cb.Name.Or(""), Username: cb.Username.Or("")}
		}
		return out, nil
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetWorkspaceUpdateForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetWorkspaceUpdateNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- PatchUpdate --------------------------------------------------------------

// PatchUpdateInput holds the fields to change on a workspace update.
type PatchUpdateInput struct {
	Description string
	Topic       string
	// Category is required.
	Category UpdateCategory
	IsPinned bool
	// RelatedResources is free-form JSON describing resources related to
	// this update. A nil value is sent as an empty array.
	RelatedResources json.RawMessage
}

// PatchUpdate updates a workspace update.
func (s *Service) PatchUpdate(ctx context.Context, workspaceID string, updateID int, in *PatchUpdateInput) (*WorkspaceUpdateRecord, error) {
	if in == nil {
		in = &PatchUpdateInput{}
	}
	descJSON, err := json.Marshal(in.Description)
	if err != nil {
		return nil, err
	}
	topicJSON, err := json.Marshal(in.Topic)
	if err != nil {
		return nil, err
	}
	req := &api.UpdateWorkspaceUpdate{
		Description:      descJSON,
		Topic:            topicJSON,
		Category:         api.WorkspaceUpdateCategoryData(in.Category),
		IsPinned:         api.NewOptBool(in.IsPinned),
		RelatedResources: relatedResourcesJSON(in.RelatedResources),
	}
	params := api.PatchWorkspaceUpdateParams{WorkspaceId: workspaceID, UpdateId: itoa(updateID)}

	res, err := s.api.PatchWorkspaceUpdate(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WorkspaceUpdatePostPatchResponseData:
		return workspaceUpdateRecordFromAPI(r), nil
	case *api.PatchWorkspaceUpdateBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.PatchWorkspaceUpdateForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.PatchWorkspaceUpdateNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.PatchWorkspaceUpdateUnprocessableEntity:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnprocessableEntity)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- DeleteUpdate -------------------------------------------------------------

// DeleteUpdate deletes a workspace update.
func (s *Service) DeleteUpdate(ctx context.Context, workspaceID string, updateID int) error {
	params := api.DeleteWorkspaceUpdateParams{WorkspaceId: workspaceID, UpdateId: itoa(updateID)}

	res, err := s.api.DeleteWorkspaceUpdate(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteWorkspaceUpdateOK:
		return nil
	case *api.ErrorTypeTitleDetailStatusInstance:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.DeleteWorkspaceUpdateForbidden:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.DeleteWorkspaceUpdateNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- helpers ----------------------------------------------------------------

func itoa(n int) string {
	return strconv.Itoa(n)
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
