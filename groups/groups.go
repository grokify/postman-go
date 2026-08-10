// Package groups provides a high-level client for Postman's Groups API.
//
// Postman user groups let team admins organize users for access control. See
// https://learning.postman.com/docs/collaborating-in-postman/user-groups/ for
// background.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	groups, _ := client.Groups().List(ctx)
//	group, _ := client.Groups().Get(ctx, "1")
package groups

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level Groups client. Obtain one via
// postman.Client.Groups.
type Service struct {
	api *api.Client
}

// New creates a Groups service over the given generated API client. Most
// callers should use postman.Client.Groups instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List ----------------------------------------------------------------

// Group is a summary of a Postman user group, as returned by List.
type Group struct {
	ID        int
	TeamID    int
	Name      string
	Summary   string
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Members   []int
	Roles     []string
}

// List returns all of a team's Postman groups.
func (s *Service) List(ctx context.Context) ([]Group, error) {
	res, err := s.api.GetGroups(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PostmanGroupsInformation:
		out := make([]Group, 0, len(r.Data))
		for _, g := range r.Data {
			out = append(out, Group{
				ID:        g.ID.Or(0),
				TeamID:    g.TeamId.Or(0),
				Name:      g.Name.Or(""),
				Summary:   g.Summary.Or(""),
				CreatedBy: g.CreatedBy.Or(0),
				CreatedAt: g.CreatedAt.Or(""),
				UpdatedAt: g.UpdatedAt.Or(""),
				Members:   g.Members,
				Roles:     g.Roles,
			})
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

// --- Get -------------------------------------------------------------------

// GroupDetail is detailed information about a single Postman group, as
// returned by Get.
type GroupDetail struct {
	ID        int
	TeamID    int
	Name      string
	Summary   string
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Members   []int
	Roles     []string
	Managers  []int
}

// Get returns information about a Postman user group.
func (s *Service) Get(ctx context.Context, groupID string) (*GroupDetail, error) {
	params := api.GetGroupParams{GroupId: groupID}

	res, err := s.api.GetGroup(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PostmanGroupInformation:
		return &GroupDetail{
			ID:        r.ID.Or(0),
			TeamID:    r.TeamId.Or(0),
			Name:      r.Name.Or(""),
			Summary:   r.Summary.Or(""),
			CreatedBy: r.CreatedBy.Or(0),
			CreatedAt: r.CreatedAt.Or(""),
			UpdatedAt: r.UpdatedAt.Or(""),
			Members:   r.Members,
			Roles:     r.Roles,
			Managers:  r.Managers,
		}, nil
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

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
