// Package users provides a high-level client for retrieving information
// about the authenticated user and users on a Postman team.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	me, _ := client.Users().Me(ctx)
//	all, _ := client.Users().List(ctx, nil)
//	one, _ := client.Users().Get(ctx, all.Users[0].ID)
package users

import (
	"context"
	"net/http"
	"strconv"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level users client. Obtain one via postman.Client.Users.
type Service struct {
	api *api.Client
}

// New creates a users service over the given generated API client. Most
// callers should use postman.Client.Users instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Me -----------------------------------------------------------------

// AuthenticatedUser describes the user authenticated by the request's API key.
type AuthenticatedUser struct {
	ID            int
	Sub           string
	Username      string
	Email         string
	FullName      string
	Avatar        string
	IsPublic      bool
	EmailVerified bool
	TeamID        int
	TeamName      string
	TeamDomain    string
	Roles         []string
}

// Operation describes usage of a metered Postman operation (e.g. API calls,
// mock usage) for the authenticated user's plan.
type Operation struct {
	Name    string
	Usage   int
	Limit   int
	Overage int
}

// MeResult is the response from Me.
type MeResult struct {
	User       AuthenticatedUser
	Operations []Operation
}

// Me returns information about the authenticated user.
//
// This endpoint returns a different response for users with the Guest and
// Partner roles. The Operations usage data only returns for users on Free
// plans.
func (s *Service) Me(ctx context.Context) (*MeResult, error) {
	res, err := s.api.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAuthenticatedUserOkResponse:
		out := &MeResult{}
		if u, ok := r.User.Get(); ok {
			out.User = AuthenticatedUser{
				ID:            u.ID.Or(0),
				Sub:           u.Sub.Or(""),
				Username:      u.Username.Or(""),
				Email:         u.Email.Or(""),
				FullName:      u.FullName.Or(""),
				Avatar:        u.Avatar.Or(""),
				IsPublic:      u.IsPublic.Or(false),
				EmailVerified: u.EmailVerified.Or(false),
				TeamID:        u.TeamId.Or(0),
				TeamName:      u.TeamName.Or(""),
				TeamDomain:    u.TeamDomain.Or(""),
				Roles:         u.Roles,
			}
		}
		for _, op := range r.Operations {
			out.Operations = append(out.Operations, Operation{
				Name:    op.Name.Or(""),
				Usage:   op.Usage.Or(0),
				Limit:   op.Limit.Or(0),
				Overage: op.Overage.Or(0),
			})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- List -----------------------------------------------------------------

// TeamUser describes a user on the Postman team.
type TeamUser struct {
	ID       int
	Name     string
	Username string
	Email    string
	Roles    []string
	JoinedAt string
}

// ListInput holds the filters for List.
type ListInput struct {
	// GroupID, when set, limits results to members of the given user group.
	// To get group IDs, use the Groups API.
	GroupID int
}

// ListResult is the response from List.
type ListResult struct {
	Users []TeamUser
}

// List returns information about all users on the Postman team.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetTeamUsersParams{}
	if in.GroupID != 0 {
		params.GroupId = api.NewOptInt(in.GroupID)
	}

	res, err := s.api.GetTeamUsers(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TeamUsersInformation:
		out := &ListResult{}
		for _, u := range r.Data {
			out.Users = append(out.Users, teamUserFromAPI(u))
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

func teamUserFromAPI(u api.UserInformation) TeamUser {
	return TeamUser{
		ID:       u.ID.Or(0),
		Name:     u.Name.Or(""),
		Username: u.Username.Or(""),
		Email:    u.Email.Or(""),
		Roles:    u.Roles,
		JoinedAt: u.JoinedAt.Or(""),
	}
}

// --- Get --------------------------------------------------------------------

// Get returns information about a single user on the Postman team.
func (s *Service) Get(ctx context.Context, userID int) (*TeamUser, error) {
	params := api.GetTeamUserParams{UserId: strconv.Itoa(userID)}

	res, err := s.api.GetTeamUser(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UserInformation:
		out := teamUserFromAPI(*r)
		return &out, nil
	case *api.Common400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
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

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
