// Package tags provides a high-level client for Postman's Tags API.
//
// Tags let a team organize and search workspaces, APIs, and collections that
// share a common label. Tagging is available on Postman Solo, Team, and
// Enterprise plans.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	got, _ := client.Tags().Workspace(ctx, workspaceID)
//	_, _ = client.Tags().UpdateWorkspace(ctx, workspaceID, []string{"my-tag"})
package tags

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-faster/jx"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// EntityType filters GetTaggedEntities results by the type of tagged
// resource.
type EntityType string

// EntityType values.
const (
	EntityTypeAPI        EntityType = "api"
	EntityTypeCollection EntityType = "collection"
	EntityTypeWorkspace  EntityType = "workspace"
)

// SortDirection controls ascending or descending order for TaggedEntities
// results, based on the time the entity was tagged.
type SortDirection string

// SortDirection values.
const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// Service is the high-level Tags client. Obtain one via postman.Client.Tags.
type Service struct {
	api *api.Client
}

// New creates a Tags service over the given generated API client. Most
// callers should use postman.Client.Tags instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Collection --------------------------------------------------------------

// Collection returns the tags (as slugs) associated with a collection.
func (s *Service) Collection(ctx context.Context, collectionID string) ([]string, error) {
	params := api.GetCollectionTagsParams{CollectionId: collectionID}

	res, err := s.api.GetCollectionTags(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SuccessResponse:
		return tagsFromRaw(r.Tags), nil
	case *api.GetCollectionTagsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetCollectionTagsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateCollection replaces all of a collection's tags with the given slugs
// (up to 5).
func (s *Service) UpdateCollection(ctx context.Context, collectionID string, slugs []string) ([]string, error) {
	req := &api.UpdateTags{Tags: tagsToRaw(slugs)}
	params := api.UpdateCollectionTagsParams{CollectionId: collectionID}

	res, err := s.api.UpdateCollectionTags(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SuccessResponse:
		return tagsFromRaw(r.Tags), nil
	case *api.ApiTag400Error1:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateCollectionTagsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.UpdateCollectionTagsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdateCollectionTagsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateCollectionTagsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Workspace --------------------------------------------------------------

// Workspace returns the tags (as slugs) associated with a workspace.
func (s *Service) Workspace(ctx context.Context, workspaceID string) ([]string, error) {
	params := api.GetWorkspaceTagsParams{WorkspaceId: workspaceID}

	res, err := s.api.GetWorkspaceTags(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SuccessResponse:
		return tagsFromRaw(r.Tags), nil
	case *api.GetWorkspaceTagsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetWorkspaceTagsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetWorkspaceTagsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetWorkspaceTagsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateWorkspace replaces all of a workspace's tags with the given slugs (up
// to 5).
func (s *Service) UpdateWorkspace(ctx context.Context, workspaceID string, slugs []string) ([]string, error) {
	req := &api.UpdateTags{Tags: tagsToRaw(slugs)}
	params := api.UpdateWorkspaceTagsParams{WorkspaceId: workspaceID}

	res, err := s.api.UpdateWorkspaceTags(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SuccessResponse:
		return tagsFromRaw(r.Tags), nil
	case *api.ApiTag400Error1:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateWorkspaceTagsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.UpdateWorkspaceTagsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdateWorkspaceTagsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateWorkspaceTagsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Entities --------------------------------------------------------------

// Entity is a Postman element (workspace, API, or collection) tagged with a
// given tag.
type Entity struct {
	ID   string
	Type EntityType
}

// EntitiesInput holds pagination and filter options for Entities.
type EntitiesInput struct {
	// Limit is the maximum number of tagged elements to return.
	Limit int
	// Direction sorts by the time of tagging; default is descending.
	Direction SortDirection
	// Cursor is the pagination cursor (use EntitiesResult.NextCursor to page).
	// An invalid value is ignored by the API, which returns the first page.
	Cursor string
	// EntityType filters results to the given entity type.
	EntityType EntityType
}

// EntitiesResult is the paginated set of elements tagged with a given tag.
type EntitiesResult struct {
	Entities   []Entity
	Count      int
	NextCursor string
}

// Entities gets Postman elements (workspaces, APIs, and collections) tagged
// with the given tag. Slug is the tag's ID within a team or individual
// (non-team) user scope.
func (s *Service) Entities(ctx context.Context, slug string, in *EntitiesInput) (*EntitiesResult, error) {
	if in == nil {
		in = &EntitiesInput{}
	}
	params := api.GetTaggedEntitiesParams{Slug: slug}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDescDefaultDesc(api.AscDescDefaultDesc(in.Direction))
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.EntityType != "" {
		params.EntityType = api.NewOptTagsEntityType(api.TagsEntityType(in.EntityType))
	}

	res, err := s.api.GetTaggedEntities(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetTaggedEntitiesOkResponse:
		out := &EntitiesResult{}
		if data, ok := r.Data.Get(); ok {
			for _, e := range data.Entities {
				out.Entities = append(out.Entities, Entity{
					ID:   e.EntityId.Or(""),
					Type: EntityType(e.EntityType.Or("")),
				})
			}
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Count = meta.Count
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetTaggedEntitiesBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetTaggedEntitiesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetTaggedEntitiesForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetTaggedEntitiesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetTaggedEntitiesInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- helpers --------------------------------------------------------------

// tag is the wire shape of a single tag: an object with a slug.
type tag struct {
	Slug string `json:"slug"`
}

// tagsToRaw encodes a list of tag slugs into the raw JSON array Postman
// expects. Postman's generated schema represents the tags field as raw JSON
// (see the "Known approximations" note in scripts/gen-openapi/README.md); the
// concrete `[{"slug": "..."}]` shape comes from the upstream TypeScript SDK's
// Zod models.
func tagsToRaw(slugs []string) jx.Raw {
	tagObjs := make([]tag, 0, len(slugs))
	for _, slug := range slugs {
		tagObjs = append(tagObjs, tag{Slug: slug})
	}
	b, err := json.Marshal(tagObjs)
	if err != nil {
		return nil
	}
	return jx.Raw(b)
}

// tagsFromRaw is the inverse of tagsToRaw.
func tagsFromRaw(raw jx.Raw) []string {
	if len(raw) == 0 {
		return nil
	}
	var tagObjs []tag
	if err := json.Unmarshal(raw, &tagObjs); err != nil {
		return nil
	}
	out := make([]string, 0, len(tagObjs))
	for _, t := range tagObjs {
		out = append(out, t.Slug)
	}
	return out
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
