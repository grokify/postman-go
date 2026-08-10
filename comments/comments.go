// Package comments provides a high-level client for Postman's Comments API.
//
// Comments can be left on collections, folders, requests, and responses.
// This package currently wraps thread resolution; comment threads on those
// resources are returned in each resource's own GET response.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	err := client.Comments().ResolveThread(ctx, threadID)
package comments

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level Comments client. Obtain one via
// postman.Client.Comments.
type Service struct {
	api *api.Client
}

// New creates a Comments service over the given generated API client. Most
// callers should use postman.Client.Comments instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- ResolveThread ------------------------------------------------------------

// ResolveThread resolves a comment and any associated replies in the given
// thread. Comment thread IDs are returned in the GET response for
// collections and collection items. On success the API returns an empty
// body.
func (s *Service) ResolveThread(ctx context.Context, threadID string) error {
	params := api.ResolveCommentThreadParams{ThreadId: threadID}

	res, err := s.api.ResolveCommentThread(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.ResolveCommentThreadOK:
		return nil
	case *api.ResolveCommentThreadUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ResolveCommentThreadNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.ResolveCommentThreadInternalServerError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
