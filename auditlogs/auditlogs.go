// Package auditlogs provides a high-level client for Postman's Audit Logs
// API.
//
// It lists a team's generated audit events and the catalog of audit log
// event actions.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	logs, _ := client.AuditLogs().List(ctx, &auditlogs.ListInput{Limit: 50})
//	actions, _ := client.AuditLogs().Actions(ctx)
package auditlogs

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Order is the sort order for paginated audit log results.
type Order string

// Order values.
const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// Service is the high-level Audit Logs client. Obtain one via
// postman.Client.AuditLogs.
type Service struct {
	api *api.Client
}

// New creates an Audit Logs service over the given generated API client.
// Most callers should use postman.Client.AuditLogs instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List -------------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
//
// NOTE: Postman's OpenAPI source defines two colliding query parameters for
// sort order, `orderBy` and `order_by`, that normalize to the same name.
// The generated client keeps only `orderBy`; `order_by` cannot be recovered
// (see scripts/gen-openapi/README.md, "Known approximations").
type ListInput struct {
	// UserID returns only results that match the given user ID.
	UserID int
	// Action filters results by an audit log action (see Actions).
	Action string
	// Since returns logs created after the given time, in YYYY-MM-DD format.
	Since string
	// Until returns logs created before the given time, in YYYY-MM-DD format.
	Until string
	// Limit is the maximum number of audit events to return at once.
	Limit int
	// Cursor is the pagination cursor.
	Cursor string
	// OrderBy returns the records in ascending or descending order.
	OrderBy Order
}

// ListResult is the page of audit events returned by List.
type ListResult struct {
	Trails []Event
}

// Event is a single audit log event.
type Event struct {
	ID        int
	IP        string
	UserAgent string
	Action    string
	Timestamp string
	Message   string
	Data      EventData
}

// EventData is the actor/user/team context of an audit log event.
type EventData struct {
	Actor Actor
	User  User
	Team  Team
}

// Actor is the entity that performed an audited action.
type Actor struct {
	Name     string
	Username string
	Email    string
	ID       int
	Active   bool
}

// User is the user affected by an audited action.
type User struct {
	Name     string
	Username string
	Email    string
	ID       int
}

// Team is the team affected by an audited action.
type Team struct {
	Name string
	ID   int
}

// List returns a team's generated audit events.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetAuditLogsParams{}
	if in.UserID != 0 {
		params.UserId = api.NewOptInt(in.UserID)
	}
	if in.Action != "" {
		params.Action = api.NewOptString(in.Action)
	}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.OrderBy != "" {
		params.OrderBy = api.NewOptAscDescDefaultDesc(api.AscDescDefaultDesc(in.OrderBy))
	}

	res, err := s.api.GetAuditLogs(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAuditLogs:
		out := &ListResult{}
		for _, t := range r.Trails {
			ev := Event{
				ID:        t.ID.Or(0),
				IP:        t.IP.Or(""),
				UserAgent: t.UserAgent.Or(""),
				Action:    t.Action.Or(""),
				Timestamp: t.Timestamp.Or(""),
				Message:   t.Message.Or(""),
			}
			if data, ok := t.Data.Get(); ok {
				if actor, ok := data.Actor.Get(); ok {
					ev.Data.Actor = Actor{
						Name:     actor.Name.Or(""),
						Username: actor.Username.Or(""),
						Email:    actor.Email.Or(""),
						ID:       actor.ID.Or(0),
						Active:   actor.Active.Or(false),
					}
				}
				if user, ok := data.User.Get(); ok {
					ev.Data.User = User{
						Name:     user.Name.Or(""),
						Username: user.Username.Or(""),
						Email:    user.Email.Or(""),
						ID:       user.ID.Or(0),
					}
				}
				if team, ok := data.Team.Get(); ok {
					ev.Data.Team = Team{
						Name: team.Name.Or(""),
						ID:   team.ID.Or(0),
					}
				}
			}
			out.Trails = append(out.Trails, ev)
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

// --- Actions ----------------------------------------------------------------

// Action describes an available audit log event action.
type Action struct {
	Name        string
	DisplayName string
}

// Actions returns the complete list of available audit log event actions.
// Use an action's Name in ListInput.Action.
func (s *Service) Actions(ctx context.Context) ([]Action, error) {
	res, err := s.api.GetAuditLogEventActions(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAuditLogEventActionsOKApplicationJSON:
		out := make([]Action, 0, len(*r))
		for _, a := range *r {
			out = append(out, Action{
				Name:        a.Name.Or(""),
				DisplayName: a.DisplayName.Or(""),
			})
		}
		return out, nil
	case *api.GetAuditLogEventActionsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetAuditLogEventActionsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
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
