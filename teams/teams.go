// Package teams provides a high-level client for Postman's Teams API.
//
// The Teams API manages a Postman team: list and create teams, inspect a
// team's membership and settings, and manage access requests and member
// roles. Most endpoints require Team Admin (or higher) permissions on the
// target team.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	teams, _ := client.Teams().List(ctx, nil)
//	team, _ := client.Teams().Get(ctx, "12345", nil)
package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Role is a team member role, used both to request a role and to describe an
// existing member's or group's assigned role.
type Role string

// Role values.
const (
	RoleTeamManager          Role = "TEAM_MANAGER"
	RoleTeamDeveloper        Role = "TEAM_DEVELOPER"
	RoleTeamGuestDeveloper   Role = "TEAM_GUEST_DEVELOPER"
	RoleTeamGuestViewer      Role = "TEAM_GUEST_VIEWER"
	RoleTeamPartnerManager   Role = "TEAM_PARTNER_MANAGER"
	RoleTeamPartnerLead      Role = "TEAM_PARTNER_LEAD"
	RoleTeamGuest            Role = "TEAM_GUEST"
	RoleTeamPartner          Role = "TEAM_PARTNER"
	RoleTeamCommunityManager Role = "TEAM_COMMUNITY_MANAGER"
)

// EntityType identifies the kind of entity referenced by an access request or
// a team-membership change.
type EntityType string

// Entity type values.
const (
	EntityTypeUser         EntityType = "user"
	EntityTypeGroup        EntityType = "group"
	EntityTypeTeam         EntityType = "team"
	EntityTypeOrganization EntityType = "organization"
)

// RequestType is the kind of access request.
type RequestType string

// Request type values.
const (
	RequestTypeAddMembers  RequestType = "REQUEST_TO_ADD_MEMBERS"
	RequestTypeJoin        RequestType = "REQUEST_TO_JOIN"
	RequestTypeUpgradeRole RequestType = "UPGRADE_ROLE"
)

// AccessAction approves or denies a pending access request.
type AccessAction string

// Access action values.
const (
	AccessActionApprove AccessAction = "approve"
	AccessActionDeny    AccessAction = "deny"
)

// Include adds optional information to a Get response.
type Include string

// Include values.
const (
	// IncludeMembers includes all users and groups with access to the team.
	IncludeMembers Include = "members"
	// IncludeUserRoles includes all the team's assigned user roles.
	IncludeUserRoles Include = "userRoles"
)

// RfaSetting is a "requires further approval" toggle for a team setting.
type RfaSetting string

// RFA setting values.
const (
	RfaEnabled  RfaSetting = "enabled"
	RfaDisabled RfaSetting = "disabled"
)

// Service is the high-level Teams client. Obtain one via postman.Client.Teams.
type Service struct {
	api *api.Client
}

// New creates a Teams service over the given generated API client. Most
// callers should use postman.Client.Teams instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List. A nil or zero
// input lists all teams the caller can see.
type ListInput struct {
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return.
	Limit int
	// Settings, when true, includes each team's settings in the response.
	Settings bool
	// UserRoles, when true, includes each team's assigned user roles.
	UserRoles bool
}

// ListResult is the paginated result of List.
type ListResult struct {
	Teams      []Team
	NextCursor string
}

// Team describes a Postman team.
type Team struct {
	ID             int
	Name           string
	Handle         string
	Description    string
	OrganizationID int
	// CreatedBy is the raw JSON representation of the team's creator (typically
	// a numeric user ID). The reconstructed OpenAPI spec could not resolve this
	// field's exact shape, so it is surfaced verbatim.
	CreatedBy string
	Enabled   bool
	CreatedAt string
	UpdatedAt string
	// MemberCount is only populated by List.
	MemberCount int
}

// List returns all Postman teams in your organization.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetTeamsParams{}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Settings {
		params.Settings = api.NewOptBool(true)
	}
	if in.UserRoles {
		params.UserRoles = api.NewOptBool(true)
	}

	res, err := s.api.GetTeams(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetTeams:
		out := &ListResult{}
		for _, d := range r.Data {
			out.Teams = append(out.Teams, Team{
				ID:             d.ID.Or(0),
				Name:           d.Name.Or(""),
				Handle:         d.Handle.Or(""),
				Description:    d.Description.Or(""),
				OrganizationID: d.OrganizationId.Or(0),
				CreatedBy:      rawJSONText([]byte(d.CreatedBy)),
				Enabled:        d.Enabled.Or(false),
				CreatedAt:      d.CreatedAt.Or(""),
				UpdatedAt:      d.UpdatedAt.Or(""),
				MemberCount:    d.MemberCount.Or(0),
			})
		}
		if meta, ok := r.Metadata.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TeamsApiErrorSchema:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create --------------------------------------------------------------

// CreateInput holds the fields for creating a team.
type CreateInput struct {
	// Name is the team's name.
	Name string
	// Description is the team's description.
	Description string
}

// Create creates a Postman team in your organization.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*Team, error) {
	if in == nil {
		in = &CreateInput{}
	}
	req := &api.CreateTeam{}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Description != "" {
		req.Description = api.NewOptNilString(in.Description)
	}

	res, err := s.api.CreateTeam(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateGetTeamResponse:
		return teamFromResponse(r), nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.CreateTeamForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.CreateTeamInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func teamFromResponse(r *api.CreateGetTeamResponse) *Team {
	t, ok := r.Team.Get()
	if !ok {
		return &Team{}
	}
	return &Team{
		ID:             t.ID.Or(0),
		Name:           t.Name.Or(""),
		Handle:         t.Handle.Or(""),
		Description:    t.Description.Or(""),
		OrganizationID: t.OrganizationId.Or(0),
		CreatedBy:      rawJSONText([]byte(t.CreatedBy)),
		Enabled:        t.Enabled.Or(false),
		CreatedAt:      t.CreatedAt.Or(""),
		UpdatedAt:      t.UpdatedAt.Or(""),
	}
}

// --- Get -------------------------------------------------------------------

// GetInput holds the options for Get.
type GetInput struct {
	// Include adds additional information to the response.
	Include Include
}

// Get returns information about a Postman team.
func (s *Service) Get(ctx context.Context, teamID string, in *GetInput) (*Team, error) {
	if in == nil {
		in = &GetInput{}
	}
	params := api.GetTeamParams{TeamId: teamID}
	if in.Include != "" {
		params.Include = api.NewOptTeamsInclude(api.TeamsInclude(in.Include))
	}

	res, err := s.api.GetTeam(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateGetTeamResponse:
		return teamFromResponse(r), nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TeamsApiErrorSchema:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- AccessRequests ----------------------------------------------------

// AccessRequestsInput holds the pagination options for AccessRequests.
type AccessRequestsInput struct {
	// Cursor is the pagination cursor (use AccessRequestsResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return.
	Limit int
}

// AccessRequestsResult is the paginated result of AccessRequests.
type AccessRequestsResult struct {
	Requests   []AccessRequest
	NextCursor string
}

// AccessRequest is a team's pending (or resolved) access request.
type AccessRequest struct {
	ID          int
	Role        Role
	RequestType string
	Reason      string
	Status      string
	EntityType  string
	// EntityID is the raw JSON representation of the requesting entity's ID
	// (a bare number or a quoted string, depending on entity type).
	EntityID   string
	ObjectType string
	ObjectID   int
	// CreatedBy is the raw JSON representation of the requester.
	CreatedBy string
	CreatedAt string
	UpdatedAt string
}

// AccessRequests returns a team's pending access requests.
func (s *Service) AccessRequests(ctx context.Context, teamID string, in *AccessRequestsInput) (*AccessRequestsResult, error) {
	if in == nil {
		in = &AccessRequestsInput{}
	}
	params := api.GetTeamAccessRequestsParams{TeamId: teamID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetTeamAccessRequests(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetTeamAccessRequests:
		out := &AccessRequestsResult{}
		for _, d := range r.Data {
			out.Requests = append(out.Requests, AccessRequest{
				ID:          d.ID.Or(0),
				Role:        Role(d.Role.Or("")),
				RequestType: d.RequestType.Or(""),
				Reason:      d.Reason.Or(""),
				Status:      d.Status.Or(""),
				EntityType:  d.EntityType.Or(""),
				EntityID:    rawJSONText([]byte(d.EntityId)),
				ObjectType:  d.ObjectType.Or(""),
				ObjectID:    d.ObjectId.Or(0),
				CreatedBy:   rawJSONText([]byte(d.CreatedBy)),
				CreatedAt:   d.CreatedAt.Or(""),
				UpdatedAt:   d.UpdatedAt.Or(""),
			})
		}
		if meta, ok := r.Metadata.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetTeamAccessRequestsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetTeamAccessRequestsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- RequestAccess -----------------------------------------------------

// EntityRef references an entity (user, group, team, or organization) by ID.
type EntityRef struct {
	Type EntityType
	// ID is the entity's ID. Numeric IDs (e.g. user or team IDs) and string
	// IDs (e.g. group IDs) are both accepted.
	ID string
}

// RequestAccessInput holds the fields for RequestAccess.
type RequestAccessInput struct {
	// Entities are the users, groups, teams, or organizations the request
	// applies to.
	Entities []EntityRef
	// Role is the requested role. Leave empty to send a null role.
	Role Role
	// Reason explains the request.
	Reason string
	// RequestType is the kind of access request.
	RequestType RequestType
}

// AccessRequestRecord is the result of processing one entity in RequestAccess.
type AccessRequestRecord struct {
	EntityType   string
	EntityID     string
	Role         string
	PreviousRole string
	Status       string
	Reason       string
}

// RequestAccessResult is the result of RequestAccess.
type RequestAccessResult struct {
	Records []AccessRequestRecord
}

// RequestAccess creates an access request for a team: a request to join, an
// upgrade of the caller's role, or a request to add members. If team
// discovery is enabled, the request is automatically approved.
func (s *Service) RequestAccess(ctx context.Context, teamID string, in *RequestAccessInput) (*RequestAccessResult, error) {
	if in == nil {
		in = &RequestAccessInput{}
	}
	req := &api.CreateAccessRequest{
		Reason:      in.Reason,
		RequestType: api.RequestType(in.RequestType),
	}
	for _, e := range in.Entities {
		req.EntityList = append(req.EntityList, api.TeamEntityInfo{
			EntityType: api.TeamEntityInfoEntityType(e.Type),
			EntityId:   api.TeamEntityInfoEntityId(entityIDToRaw(e.ID)),
		})
	}
	if in.Role != "" {
		req.Role.SetTo(api.CreateAccessRequestRole(in.Role))
	} else {
		req.Role.SetToNull()
	}
	params := api.CreateAccessRequestParams{TeamId: teamID}

	res, err := s.api.CreateAccessRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateAccessRequestResponse:
		out := &RequestAccessResult{}
		for _, d := range r.Result {
			out.Records = append(out.Records, AccessRequestRecord{
				EntityType:   d.EntityType.Or(""),
				EntityID:     rawJSONText([]byte(d.EntityId)),
				Role:         d.Role.Or(""),
				PreviousRole: d.PreviousRole.Or(""),
				Status:       d.Status.Or(""),
				Reason:       d.Reason.Or(""),
			})
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TeamsApiErrorSchema:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- DecideAccessRequest -------------------------------------------------

// DecideAccessRequestResult is the result of DecideAccessRequest.
type DecideAccessRequestResult struct {
	EntityType    string
	EntityID      string
	Role          string
	PreviousRole  string
	Status        string
	AccessRequest *AccessRequestSummary
}

// AccessRequestSummary summarizes the original access request that was
// approved or denied.
type AccessRequestSummary struct {
	ID          int
	RequestType string
	Reason      string
	Status      string
	ObjectType  string
	ObjectID    int
	CreatedBy   int
}

// DecideAccessRequest approves or denies a team's pending access request.
func (s *Service) DecideAccessRequest(ctx context.Context, teamID, requestID string, action AccessAction) (*DecideAccessRequestResult, error) {
	req := &api.ApproveDenyAccessRequest{Action: api.ApproveDenyAccessRequestAction(action)}
	params := api.ApproveDenyAccessRequestParams{TeamId: teamID, RequestId: requestID}

	res, err := s.api.ApproveDenyAccessRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ApproveDenyAccessRequestResponse:
		out := &DecideAccessRequestResult{}
		if result, ok := r.Result.Get(); ok {
			out.EntityType = result.EntityType.Or("")
			out.EntityID = rawJSONText([]byte(result.EntityId))
			out.Role = result.Role.Or("")
			out.PreviousRole = result.PreviousRole.Or("")
			out.Status = result.Status.Or("")
			if ar, ok := result.AccessRequest.Get(); ok {
				out.AccessRequest = &AccessRequestSummary{
					ID:          ar.ID.Or(0),
					RequestType: ar.RequestType.Or(""),
					Reason:      ar.Reason.Or(""),
					Status:      ar.Status.Or(""),
					ObjectType:  ar.ObjectType.Or(""),
					ObjectID:    ar.ObjectId.Or(0),
					CreatedBy:   ar.CreatedBy.Or(0),
				}
			}
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApproveDenyAccessRequestForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.ApproveDenyAccessRequestNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.ApproveDenyAccessRequestInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- ManageMemberRoles ---------------------------------------------------

// MemberRoleChanges groups role changes by entity category.
//
// Note: the reconstructed OpenAPI spec (see scripts/gen-openapi/README.md)
// represents the API's dynamic per-entity-ID JSON object (e.g.
// {"<userId>": ["TEAM_MANAGER"]}) with a single fixed placeholder key,
// because the source TypeScript SDK encodes that key as a literal template
// string rather than a real additionalProperties map. As a result the
// generated client can only address one implicit target per category; the
// slices below list the roles to assign to (or remove from) that category as
// a whole, not per individual entity ID.
type MemberRoleChanges struct {
	Users  []Role
	Groups []Role
	Orgs   []Role
	Teams  []Role
}

// ManageMemberRolesInput holds the additions and removals for
// ManageMemberRoles. Removing a role from a group or team removes that
// role's permissions from all its members.
type ManageMemberRolesInput struct {
	Add    *MemberRoleChanges
	Remove *MemberRoleChanges
}

// MemberRoleResult describes the outcome of one role change.
type MemberRoleResult struct {
	EntityType   string
	EntityID     string
	Role         string
	PreviousRole string
	Status       string
}

// ManageMemberRolesResult is the result of ManageMemberRoles.
type ManageMemberRolesResult struct {
	Results []MemberRoleResult
}

// ManageMemberRoles adds or removes member roles in groups, teams,
// organizations, and for individual users.
func (s *Service) ManageMemberRoles(ctx context.Context, teamID string, in *ManageMemberRolesInput) (*ManageMemberRolesResult, error) {
	if in == nil {
		in = &ManageMemberRolesInput{}
	}
	req := &api.ManageTeamMemberRoles{}
	if in.Add != nil {
		req.Add = api.NewOptManageTeamMemberRolesAdd(memberRoleChangesToAdd(in.Add))
	}
	if in.Remove != nil {
		req.Remove = api.NewOptManageTeamMemberRolesRemove(memberRoleChangesToRemove(in.Remove))
	}
	params := api.ManageTeamMemberRolesParams{TeamId: teamID}

	res, err := s.api.ManageTeamMemberRoles(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ManageTeamMemberRolesResponse:
		out := &ManageMemberRolesResult{}
		for _, d := range r.Result {
			out.Results = append(out.Results, MemberRoleResult{
				EntityType:   d.EntityType.Or(""),
				EntityID:     rawJSONText([]byte(d.EntityId)),
				Role:         d.Role.Or(""),
				PreviousRole: d.PreviousRole.Or(""),
				Status:       d.Status.Or(""),
			})
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TeamsApiErrorSchema:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func rolesToTeamRoles(roles []Role) []api.TeamRoles {
	out := make([]api.TeamRoles, 0, len(roles))
	for _, r := range roles {
		out = append(out, api.TeamRoles(r))
	}
	return out
}

func memberRoleChangesToAdd(c *MemberRoleChanges) api.ManageTeamMemberRolesAdd {
	var add api.ManageTeamMemberRolesAdd
	if len(c.Users) > 0 {
		add.Users = api.NewOptUsersInfo(api.UsersInfo{UserId: rolesToTeamRoles(c.Users)})
	}
	if len(c.Groups) > 0 {
		add.Groups = api.NewOptUserGroupsInfo(api.UserGroupsInfo{UserGroupId: rolesToTeamRoles(c.Groups)})
	}
	if len(c.Orgs) > 0 {
		add.Orgs = api.NewOptOrgsInfo(api.OrgsInfo{OrgId: rolesToTeamRoles(c.Orgs)})
	}
	if len(c.Teams) > 0 {
		add.Teams = api.NewOptTeamsInfo(api.TeamsInfo{TeamId: rolesToTeamRoles(c.Teams)})
	}
	return add
}

func memberRoleChangesToRemove(c *MemberRoleChanges) api.ManageTeamMemberRolesRemove {
	var remove api.ManageTeamMemberRolesRemove
	if len(c.Users) > 0 {
		remove.Users = api.NewOptUsersInfo(api.UsersInfo{UserId: rolesToTeamRoles(c.Users)})
	}
	if len(c.Groups) > 0 {
		remove.Groups = api.NewOptUserGroupsInfo(api.UserGroupsInfo{UserGroupId: rolesToTeamRoles(c.Groups)})
	}
	if len(c.Orgs) > 0 {
		remove.Orgs = api.NewOptOrgsInfo(api.OrgsInfo{OrgId: rolesToTeamRoles(c.Orgs)})
	}
	if len(c.Teams) > 0 {
		remove.Teams = api.NewOptTeamsInfo(api.TeamsInfo{TeamId: rolesToTeamRoles(c.Teams)})
	}
	return remove
}

// --- RemoveMembers -------------------------------------------------------

// RemoveMembers removes entities (users, groups, teams, or organizations)
// from a Postman team.
func (s *Service) RemoveMembers(ctx context.Context, teamID string, entities []EntityRef) error {
	req := &api.RemoveTeamMembers{}
	for _, e := range entities {
		req.Entities = append(req.Entities, api.TeamEntityInfo{
			EntityType: api.TeamEntityInfoEntityType(e.Type),
			EntityId:   api.TeamEntityInfoEntityId(entityIDToRaw(e.ID)),
		})
	}
	params := api.RemoveTeamMembersParams{TeamId: teamID}

	res, err := s.api.RemoveTeamMembers(ctx, req, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.RemoveTeamMembersOK:
		return nil
	case *api.Teams400Error:
		return postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.RemoveTeamMembersForbidden:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.RemoveTeamMembersInternalServerError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Settings / UpdateSettings ----------------------------------------------

// TeamSettings describes a team's approval-required settings.
type TeamSettings struct {
	RfaForAddMember       string
	RfaForAddCollaborator string
}

// Settings returns a team's settings.
func (s *Service) Settings(ctx context.Context, teamID string) (*TeamSettings, error) {
	params := api.GetTeamSettingsParams{TeamId: teamID}

	res, err := s.api.GetTeamSettings(ctx, params)
	if err != nil {
		return nil, err
	}
	return teamSettingsResult(res)
}

// UpdateSettingsInput holds the fields for UpdateSettings. Leave a field
// empty to leave the corresponding setting unchanged.
type UpdateSettingsInput struct {
	RfaForAddMember       RfaSetting
	RfaForAddCollaborator RfaSetting
}

// UpdateSettings updates a team's settings.
func (s *Service) UpdateSettings(ctx context.Context, teamID string, in *UpdateSettingsInput) (*TeamSettings, error) {
	if in == nil {
		in = &UpdateSettingsInput{}
	}
	var settings api.UpdateTeamSettingsSettings
	if in.RfaForAddMember != "" {
		settings.RfaForAddMember = api.NewOptRfaForAddMember(api.RfaForAddMember(in.RfaForAddMember))
	}
	if in.RfaForAddCollaborator != "" {
		settings.RfaForAddCollaborator = api.NewOptRfaForAddCollaborator(api.RfaForAddCollaborator(in.RfaForAddCollaborator))
	}
	req := &api.UpdateTeamSettings{Settings: api.NewOptUpdateTeamSettingsSettings(settings)}
	params := api.UpdateTeamSettingsParams{TeamId: teamID}

	res, err := s.api.UpdateTeamSettings(ctx, req, params)
	if err != nil {
		return nil, err
	}
	return teamSettingsResult(res)
}

// teamSettingsResult maps the shared success/error union of GetTeamSettings
// and UpdateTeamSettings to a *TeamSettings. It takes any since the two
// operations use distinct (but structurally identical) ogen response
// interfaces.
func teamSettingsResult(res any) (*TeamSettings, error) {
	switch r := res.(type) {
	case *api.CreateGetTeamSettingsResponse:
		out := &TeamSettings{}
		if set, ok := r.Settings.Get(); ok {
			out.RfaForAddMember = set.RfaForAddMember.Or("")
			out.RfaForAddCollaborator = set.RfaForAddCollaborator.Or("")
		}
		return out, nil
	case *api.Teams400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TeamsApiErrorSchema:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- helpers -----------------------------------------------------------

// rawJSONText renders a raw JSON scalar the reconstructed OpenAPI spec could
// not resolve to a concrete type as a display string: JSON strings are
// unquoted, other scalars (numbers, objects) pass through verbatim.
func rawJSONText(raw []byte) string {
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return string(raw)
}

// entityIDToRaw encodes an entity ID as the raw JSON scalar the API expects:
// a bare integer for numeric IDs, or a quoted string otherwise.
func entityIDToRaw(id string) []byte {
	if id != "" {
		if _, err := strconv.ParseInt(id, 10, 64); err == nil {
			return []byte(id)
		}
	}
	b, err := json.Marshal(id)
	if err != nil {
		return []byte(`""`)
	}
	return b
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
