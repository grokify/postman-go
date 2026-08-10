// Package pullrequests provides a high-level client for Postman's Pull
// Requests API.
//
// Pull requests let collection collaborators propose, review, and merge
// changes between forked collections. See
// https://learning.postman.com/docs/collaborating-in-postman/using-version-control/reviewing-pull-requests/
// for background.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	pr, _ := client.PullRequests().Get(ctx, "6")
//	_, _ = client.PullRequests().Review(ctx, "6", &pullrequests.ReviewInput{Action: pullrequests.ReviewActionApprove})
package pullrequests

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Status is the status of a pull request.
type Status string

// Pull request status values.
const (
	StatusOpen     Status = "open"
	StatusApproved Status = "approved"
	StatusDeclined Status = "declined"
	StatusMerged   Status = "merged"
)

// ReviewAction is an action taken on a pull request review.
type ReviewAction string

// Review action values.
const (
	// ReviewActionApprove approves the pull request.
	ReviewActionApprove ReviewAction = "approve"
	// ReviewActionDecline declines the pull request.
	ReviewActionDecline ReviewAction = "decline"
	// ReviewActionMerge merges the pull request.
	ReviewActionMerge ReviewAction = "merge"
	// ReviewActionUnapprove removes a previously given approval.
	ReviewActionUnapprove ReviewAction = "unapprove"
)

// MergeStatus is the status of a pull request's merge.
type MergeStatus string

// Merge status values.
const (
	MergeStatusInactive   MergeStatus = "inactive"
	MergeStatusInProgress MergeStatus = "inprogress"
	MergeStatusFailed     MergeStatus = "failed"
)

// Service is the high-level Pull Requests client. Obtain one via
// postman.Client.PullRequests.
type Service struct {
	api *api.Client
}

// New creates a Pull Requests service over the given generated API client.
// Most callers should use postman.Client.PullRequests instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Get ---------------------------------------------------------------

// Source describes the collection fork a pull request was created from.
type Source struct {
	ID       string
	Name     string
	ForkName string
	Exists   bool
}

// Destination describes the collection a pull request merges into.
type Destination struct {
	ID     string
	Name   string
	Exists bool
}

// Merge describes the current state of a pull request's merge.
type Merge struct {
	Status MergeStatus
}

// Reviewer is a user assigned to review a pull request.
type Reviewer struct {
	ID     string
	Status string
}

// PullRequest describes a collection pull request.
type PullRequest struct {
	ID          string
	Title       string
	Description string
	CreatedAt   string
	UpdatedAt   string
	CreatedBy   string
	UpdatedBy   string
	Comment     string
	ForkType    string
	Source      Source
	Destination Destination
	Status      string
	Merge       Merge
	Reviewers   []Reviewer
}

// Get returns information about a pull request, such as the source and
// destination details, who reviewed it, the merge's current status, and
// whether the element is accessible.
func (s *Service) Get(ctx context.Context, pullRequestID string) (*PullRequest, error) {
	params := api.GetPullRequestParams{PullRequestId: pullRequestID}

	res, err := s.api.GetPullRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetPullRequestOkResponse:
		out := &PullRequest{
			ID:          r.ID.Or(""),
			Title:       r.Title.Or(""),
			Description: r.Description.Or(""),
			CreatedAt:   r.CreatedAt.Or(""),
			UpdatedAt:   r.UpdatedAt.Or(""),
			CreatedBy:   r.CreatedBy.Or(""),
			UpdatedBy:   r.UpdatedBy.Or(""),
			Comment:     r.Comment.Or(""),
			ForkType:    r.FortkType.Or(""),
			Status:      r.Status.Or(""),
		}
		if src, ok := r.Source.Get(); ok {
			out.Source = Source{
				ID:       src.ID.Or(""),
				Name:     src.Name.Or(""),
				ForkName: src.ForkName.Or(""),
				Exists:   src.Exists.Or(false),
			}
		}
		if dst, ok := r.Destination.Get(); ok {
			out.Destination = Destination{
				ID:     dst.ID.Or(""),
				Name:   dst.Name.Or(""),
				Exists: dst.Exists.Or(false),
			}
		}
		if m, ok := r.Merge.Get(); ok {
			out.Merge = Merge{Status: MergeStatus(m.Status.Or(""))}
		}
		for _, rv := range r.Reviewers {
			out.Reviewers = append(out.Reviewers, Reviewer{
				ID:     rv.ID.Or(""),
				Status: rv.Status.Or(""),
			})
		}
		return out, nil
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Update --------------------------------------------------------------

// UpdateInput holds the fields for updating an open pull request.
type UpdateInput struct {
	// Title is the pull request's new title.
	Title string
	// Description is the pull request's new description.
	Description string
	// Reviewers is the list of user IDs to assign as reviewers.
	Reviewers []string
}

// UpdateResult is returned after updating a pull request.
type UpdateResult struct {
	ID            string
	Title         string
	Description   string
	SourceID      string
	DestinationID string
	ForkType      string
	Status        Status
	CreatedAt     string
	CreatedBy     string
	UpdatedAt     string
}

// Update updates an open pull request's title, description, or reviewers.
func (s *Service) Update(ctx context.Context, pullRequestID string, in *UpdateInput) (*UpdateResult, error) {
	if in == nil {
		in = &UpdateInput{}
	}
	req := &api.UpdatePullRequest{
		Title:     in.Title,
		Reviewers: in.Reviewers,
	}
	if in.Description != "" {
		req.Description = api.NewOptString(in.Description)
	}
	params := api.UpdatePullRequestParams{PullRequestId: pullRequestID}

	res, err := s.api.UpdatePullRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PullRequestUpdated:
		return &UpdateResult{
			ID:            r.ID.Or(""),
			Title:         r.Title.Or(""),
			Description:   r.Description.Or(""),
			SourceID:      r.SourceId.Or(""),
			DestinationID: r.DestinationId.Or(""),
			ForkType:      r.ForkType.Or(""),
			Status:        Status(r.Status.Or("")),
			CreatedAt:     r.CreatedAt.Or(""),
			CreatedBy:     r.CreatedBy.Or(""),
			UpdatedAt:     r.UpdatedAt.Or(""),
		}, nil
	case *api.UpdatePullRequestForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdatePullRequestConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Review ----------------------------------------------------------------

// ReviewInput holds the fields for reviewing a pull request.
type ReviewInput struct {
	// Action is the review action to take.
	Action ReviewAction
	// Comment is an optional comment to attach to the review.
	Comment string
}

// ReviewedBy identifies who reviewed a pull request.
type ReviewedBy struct {
	ID       int
	Name     string
	Username string
}

// ReviewResult is returned after reviewing a pull request.
type ReviewResult struct {
	ID         string
	Status     string
	UpdatedAt  string
	ReviewedBy ReviewedBy
}

// Review updates the review status of a pull request (approve, decline,
// merge, or unapprove).
func (s *Service) Review(ctx context.Context, pullRequestID string, in *ReviewInput) (*ReviewResult, error) {
	if in == nil {
		in = &ReviewInput{}
	}
	req := &api.ReviewPullRequest{
		Action: api.ReviewPullRequestAction(in.Action),
	}
	if in.Comment != "" {
		req.Comment = api.NewOptString(in.Comment)
	}
	params := api.ReviewPullRequestParams{PullRequestId: pullRequestID}

	res, err := s.api.ReviewPullRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ReviewPullRequestOkResponse:
		out := &ReviewResult{
			ID:        r.ID.Or(""),
			Status:    r.Status.Or(""),
			UpdatedAt: r.UpdatedAt.Or(""),
		}
		if rb, ok := r.ReviewedBy.Get(); ok {
			out.ReviewedBy = ReviewedBy{
				ID:       rb.ID.Or(0),
				Name:     rb.Name.Or(""),
				Username: rb.Username.Or(""),
			}
		}
		return out, nil
	case *api.ReviewPullRequestBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ReviewPullRequestForbidden:
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
