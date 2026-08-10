// Package collectionaccesskeys provides a high-level client for Postman's
// Collection Access Keys API.
//
// Collection access keys let integrations authenticate requests scoped to a
// single collection. See
// https://learning.postman.com/docs/developer/postman-api/authentication/#generate-a-collection-access-key
// for background.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	keys, _ := client.CollectionAccessKeys().List(ctx, nil)
//	_ = client.CollectionAccessKeys().Delete(ctx, "key-id")
package collectionaccesskeys

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Status is the status of a collection access key.
type Status string

// Collection access key status values.
const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Service is the high-level Collection Access Keys client. Obtain one via
// postman.Client.CollectionAccessKeys.
type Service struct {
	api *api.Client
}

// New creates a Collection Access Keys service over the given generated API
// client. Most callers should use postman.Client.CollectionAccessKeys instead
// of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List ------------------------------------------------------------------

// ListInput holds the filters and pagination options for List. A nil or zero
// value returns all results.
type ListInput struct {
	// CollectionID filters the results by a collection's unique ID.
	CollectionID string
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
}

// AccessKey is a personal or team collection access key.
//
// ExpiresAfter is the date and time at which the access key expires.
// Collection access keys are valid for 60 days; if unused, the key expires
// after 60 days, and each use extends the expiration by another 60 days.
// LastUsedAt is empty if the key has never been used.
type AccessKey struct {
	ID           string
	Token        string
	Status       Status
	TeamID       int
	UserID       int
	CollectionID string
	ExpiresAfter string
	LastUsedAt   string
	CreatedAt    string
	UpdatedAt    string
	DeletedAt    string
}

// ListResult is the paginated set of collection access keys.
type ListResult struct {
	Keys       []AccessKey
	NextCursor string
	PrevCursor string
}

// List returns the authenticated user's personal and team collection access
// keys.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetCollectionAccessKeysParams{}
	if in.CollectionID != "" {
		params.CollectionId = api.NewOptString(in.CollectionID)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}

	res, err := s.api.GetCollectionAccessKeys(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionAccessKeys:
		out := &ListResult{}
		for _, k := range r.Data {
			out.Keys = append(out.Keys, AccessKey{
				ID:           k.ID.Or(""),
				Token:        k.Token.Or(""),
				Status:       Status(k.Status.Or("")),
				TeamID:       k.TeamId.Or(0),
				UserID:       k.UserId.Or(0),
				CollectionID: k.CollectionId.Or(""),
				ExpiresAfter: k.ExpiresAfter.Or(""),
				LastUsedAt:   k.LastUsedAt.Or(""),
				CreatedAt:    k.CreatedAt.Or(""),
				UpdatedAt:    k.UpdatedAt.Or(""),
				DeletedAt:    k.DeletedAt.Value,
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
			out.PrevCursor = meta.PrevCursor.Or("")
		}
		return out, nil
	case *api.Common400Error:
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

// --- Delete ------------------------------------------------------------

// Delete deletes a collection access key. Use List to find a key's ID.
func (s *Service) Delete(ctx context.Context, keyID string) error {
	params := api.DeleteCollectionAccessKeyParams{KeyId: keyID}

	res, err := s.api.DeleteCollectionAccessKey(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteCollectionAccessKeyOK:
		return nil
	case *api.Common401Error:
		return postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.DeleteCollectionAccessKeyNotFoundResponse:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
