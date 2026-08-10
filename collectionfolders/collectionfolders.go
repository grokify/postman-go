// Package collectionfolders provides a high-level client for managing
// comments on folders within a Postman collection.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	comments, _ := client.CollectionFolders().Get(ctx, collectionID, folderID)
//	created, _ := client.CollectionFolders().Create(ctx, collectionID, folderID, &collectionfolders.CreateInput{
//		Body: "Looks good to me.",
//	})
package collectionfolders

import (
	"context"
	"net/http"
	"strconv"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level collection folder comments client. Obtain one
// via postman.Client.CollectionFolders.
type Service struct {
	api *api.Client
}

// New creates a collection folder comments service over the given
// generated API client. Most callers should use
// postman.Client.CollectionFolders instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Tag identifies a single user tagged (@mentioned) in a comment body.
//
// The underlying generated API type only supports a single tagged user per
// request or response body: Postman's real schema keys tags by a
// `{{userName}}` placeholder map, which the SDK generator collapses to one
// fixed field (see scripts/gen-openapi/README.md, "Known approximations").
type Tag struct {
	// Type is the tag kind. Postman currently only defines "user".
	Type string
	// ID is the tagged user's ID.
	ID string
}

// Comment is a single comment left on a folder.
type Comment struct {
	ID        int
	ThreadID  int
	Status    string
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Body      string
}

// GetResult is the set of comments on a folder.
type GetResult struct {
	Comments []Comment
}

// Get returns all comments left by users on a folder.
func (s *Service) Get(ctx context.Context, collectionID, folderID string) (*GetResult, error) {
	params := api.GetFolderCommentsParams{
		CollectionId: collectionID,
		FolderId:     folderID,
	}

	res, err := s.api.GetFolderComments(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentResponseObject:
		out := &GetResult{}
		for _, d := range r.Data {
			out.Comments = append(out.Comments, commentFromAPI(d))
		}
		return out, nil
	case *api.GetFolderCommentsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetFolderCommentsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetFolderCommentsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetFolderCommentsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func commentFromAPI(d api.CommentData) Comment {
	return Comment{
		ID:        d.ID.Or(0),
		ThreadID:  d.ThreadId.Or(0),
		Status:    d.Status.Or(""),
		CreatedBy: d.CreatedBy.Or(0),
		CreatedAt: d.CreatedAt.Or(""),
		UpdatedAt: d.UpdatedAt.Or(""),
		Body:      d.Body.Or(""),
	}
}

// CreateInput holds the fields for creating a comment on a folder.
type CreateInput struct {
	// Body is the contents of the comment. Required, max 10,000 characters.
	Body string
	// ThreadID, when set, creates the comment as a reply to an existing thread.
	ThreadID int
	// Tag, when set, tags a user in the comment body. See Tag for a
	// documented limitation of this field.
	Tag *Tag
}

// CommentResult is the comment returned after a create or update operation.
type CommentResult struct {
	ID        int
	ThreadID  int
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Body      string
}

// Create creates a comment on a folder. To create a reply on an existing
// comment, set CreateInput.ThreadID.
//
// This endpoint accepts a max of 10,000 characters.
func (s *Service) Create(ctx context.Context, collectionID, folderID string, in *CreateInput) (*CommentResult, error) {
	if in == nil {
		in = &CreateInput{}
	}
	req := &api.CommentCreate{Body: in.Body}
	if in.ThreadID != 0 {
		req.ThreadId = api.NewOptInt(in.ThreadID)
	}
	if in.Tag != nil {
		req.Tags = tagToAPI(in.Tag)
	}
	params := api.CreateFolderCommentParams{
		CollectionId: collectionID,
		FolderId:     folderID,
	}

	res, err := s.api.CreateFolderComment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentUpdatedCreatedObject:
		return commentResultFromAPI(r), nil
	case *api.CreateFolderCommentUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.CreateFolderCommentForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.CreateFolderCommentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.CreateFolderCommentInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateInput holds the fields for updating a comment on a folder.
type UpdateInput struct {
	// Body is the new contents of the comment. Required, max 10,000 characters.
	Body string
	// Tag, when set, tags a user in the comment body. See Tag for a
	// documented limitation of this field.
	Tag *Tag
}

// Update updates a comment on a folder.
//
// This endpoint accepts a max of 10,000 characters.
func (s *Service) Update(ctx context.Context, collectionID, folderID string, commentID int, in *UpdateInput) (*CommentResult, error) {
	if in == nil {
		in = &UpdateInput{}
	}
	req := &api.CommentUpdate{Body: in.Body}
	if in.Tag != nil {
		req.Tags = tagToAPI(in.Tag)
	}
	params := api.UpdateFolderCommentParams{
		CollectionId: collectionID,
		FolderId:     folderID,
		CommentId:    strconv.Itoa(commentID),
	}

	res, err := s.api.UpdateFolderComment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentUpdatedCreatedObject:
		return commentResultFromAPI(r), nil
	case *api.UpdateFolderCommentUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.UpdateFolderCommentForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdateFolderCommentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateFolderCommentInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func commentResultFromAPI(r *api.CommentUpdatedCreatedObject) *CommentResult {
	out := &CommentResult{}
	if d, ok := r.Data.Get(); ok {
		out.ID = d.ID.Or(0)
		out.ThreadID = d.ThreadId.Or(0)
		out.CreatedBy = d.CreatedBy.Or(0)
		out.CreatedAt = d.CreatedAt.Or("")
		out.UpdatedAt = d.UpdatedAt.Or("")
		out.Body = d.Body.Or("")
	}
	return out
}

func tagToAPI(t *Tag) api.OptTaggedUsers {
	return api.NewOptTaggedUsers(api.TaggedUsers{
		UserName: api.NewOptUserName(api.UserName{Type: t.Type, ID: t.ID}),
	})
}

// Delete deletes a comment from a folder. Deleting the first comment of a
// thread deletes all the comments in the thread.
func (s *Service) Delete(ctx context.Context, collectionID, folderID string, commentID int) error {
	params := api.DeleteFolderCommentParams{
		CollectionId: collectionID,
		FolderId:     folderID,
		CommentId:    strconv.Itoa(commentID),
	}

	res, err := s.api.DeleteFolderComment(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteFolderCommentOK:
		return nil
	case *api.DeleteFolderCommentUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.DeleteFolderCommentForbidden:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.DeleteFolderCommentNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteFolderCommentInternalServerError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
