// Package collections provides a high-level client for Postman's Collections
// API.
//
// Collections group related API requests together, along with example
// responses, scripts, and variables. This package covers collection CRUD,
// forking and merging, comments, pull requests, roles, publishing
// documentation, and transferring items between collections.
//
// Full collection bodies (for Create, Replace, Update, and Get) are passed
// and returned as raw JSON in the Postman Collection v2.1.0 format
// (https://schema.postman.com/collection/json/v2.1.0/draft-07/docs/index.html)
// rather than as hand-modeled Go structs: the format's item tree (folders,
// requests, responses, auth, scripts) is a large, separately-versioned
// schema that is orthogonal to this service's job of managing collections.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	list, _ := client.Collections().List(ctx, &collections.ListInput{Workspace: wsID})
//	got, _ := client.Collections().Get(ctx, list.Collections[0].ID, nil)
package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// SortDirection orders paginated list results by creation date.
type SortDirection string

// Sort direction values.
const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

// CollectionModel restricts a Get response to summary data.
type CollectionModel string

// Collection model values.
const (
	// CollectionModelMinimal returns only root-level request and folder IDs
	// instead of the full collection element tree.
	CollectionModelMinimal CollectionModel = "minimal"
)

// TransformFormat is the output format for TransformToOpenAPI.
type TransformFormat string

// Transform format values.
const (
	TransformFormatJSON TransformFormat = "json"
	TransformFormatYAML TransformFormat = "yaml"
)

// MergeStrategy controls how a fork's changes are merged with its parent.
type MergeStrategy string

// Merge strategy values. MergeStrategyDefault is only valid for
// MergeForkAsync; MergeFork (deprecated) accepts only the other two values.
const (
	// MergeStrategyDefault merges normally, keeping both collections' history.
	MergeStrategyDefault MergeStrategy = "default"
	// MergeStrategyUpdateSourceWithDestination overwrites the source
	// collection's contents with the destination's.
	MergeStrategyUpdateSourceWithDestination MergeStrategy = "updateSourceWithDestination"
	// MergeStrategyDeleteSource merges changes and deletes the source
	// (forked) collection.
	MergeStrategyDeleteSource MergeStrategy = "deleteSource"
)

// DocLayout is a published documentation page's column layout.
type DocLayout string

// Documentation layout values.
const (
	DocLayoutClassicSingleColumn DocLayout = "classic-single-column"
	DocLayoutClassicDoubleColumn DocLayout = "classic-double-column"
)

// Role is an access level granted to a user, group, or team on a collection.
type Role string

// Role values.
const (
	RoleViewer Role = "VIEWER"
	RoleEditor Role = "EDITOR"
)

// RolePath identifies the kind of principal a role update applies to.
type RolePath string

// Role path values.
const (
	RolePathUser  RolePath = "/user"
	RolePathGroup RolePath = "/group"
	RolePathTeam  RolePath = "/team"
)

// TransferMode selects whether transferred items are copied or moved.
type TransferMode string

// Transfer mode values.
const (
	TransferModeCopy TransferMode = "copy"
	TransferModeMove TransferMode = "move"
)

// TransferTargetModel identifies the kind of resource items are transferred into.
type TransferTargetModel string

// Transfer target model values.
const (
	TransferTargetModelCollection TransferTargetModel = "collection"
	TransferTargetModelFolder     TransferTargetModel = "folder"
	TransferTargetModelRequest    TransferTargetModel = "request"
)

// TransferPosition places transferred items relative to a sibling location.
type TransferPosition string

// Transfer position values.
const (
	TransferPositionStart  TransferPosition = "start"
	TransferPositionEnd    TransferPosition = "end"
	TransferPositionBefore TransferPosition = "before"
	TransferPositionAfter  TransferPosition = "after"
)

// DuplicateTaskStatus is the status of a collection duplication task.
type DuplicateTaskStatus string

// Duplicate task status values.
const (
	DuplicateTaskProcessing DuplicateTaskStatus = "processing"
	DuplicateTaskCompleted  DuplicateTaskStatus = "completed"
	DuplicateTaskFailed     DuplicateTaskStatus = "failed"
)

// AsyncTaskStatus is the status of an asynchronous merge or collection update task.
type AsyncTaskStatus string

// Async task status values.
const (
	AsyncTaskSuccessful AsyncTaskStatus = "successful"
	AsyncTaskInProgress AsyncTaskStatus = "in-progress"
	AsyncTaskFailed     AsyncTaskStatus = "failed"
)

// PullRequestStatus is the review status of a collection pull request.
type PullRequestStatus string

// Pull request status values.
const (
	PullRequestStatusOpen     PullRequestStatus = "open"
	PullRequestStatusApproved PullRequestStatus = "approved"
	PullRequestStatusDeclined PullRequestStatus = "declined"
	PullRequestStatusMerged   PullRequestStatus = "merged"
)

// Service is the high-level Collections client. Obtain one via
// postman.Client.Collections.
type Service struct {
	api *api.Client
}

// New creates a Collections service over the given generated API client.
// Most callers should use postman.Client.Collections instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// Workspace limits results to collections in this workspace.
	Workspace string
	// Name filters results to collections whose name exactly matches this
	// value. Cannot be combined with Limit or Offset.
	Name string
	// Limit is the maximum number of rows to return.
	Limit int
	// Offset is the zero-based offset of the first item to return.
	Offset int
}

// ForkOrigin describes the fork relationship of a collection created from
// another collection.
type ForkOrigin struct {
	Label     string
	CreatedAt string
	From      string
}

// CollectionSummary is a single collection entry returned by List.
type CollectionSummary struct {
	ID        string
	Name      string
	Owner     string
	CreatedAt string
	UpdatedAt string
	UID       string
	IsPublic  bool
	// Fork is set when this collection is a fork of another collection.
	Fork *ForkOrigin
}

// ListResult is the paginated result of List.
type ListResult struct {
	Collections []CollectionSummary
	Total       int
	Offset      int
	Limit       int
}

// List returns the authenticated user's subscribed collections. Use
// pagination (Limit/Offset) for large result sets; passing an invalid
// Workspace returns an empty result rather than an error.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetCollectionsParams{}
	if in.Workspace != "" {
		params.Workspace = api.NewOptString(in.Workspace)
	}
	if in.Name != "" {
		params.Name = api.NewOptString(in.Name)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Offset > 0 {
		params.Offset = api.NewOptInt(in.Offset)
	}

	res, err := s.api.GetCollections(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionsList:
		out := &ListResult{}
		for _, c := range r.Collections {
			out.Collections = append(out.Collections, collectionSummaryFromAPI(c))
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Total = meta.Total.Or(0)
			out.Offset = meta.Offset.Or(0)
			out.Limit = meta.Limit.Or(0)
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

func collectionSummaryFromAPI(c api.CollectionsListCollections) CollectionSummary {
	out := CollectionSummary{
		ID:        c.ID.Or(""),
		Name:      c.Name.Or(""),
		Owner:     c.Owner.Or(""),
		CreatedAt: c.CreatedAt.Or(""),
		UpdatedAt: c.UpdatedAt.Or(""),
		UID:       c.UID.Or(""),
		IsPublic:  c.IsPublic.Or(false),
	}
	if f, ok := c.Fork.Get(); ok {
		out.Fork = &ForkOrigin{Label: f.Label.Or(""), CreatedAt: f.CreatedAt.Or(""), From: f.From.Or("")}
	}
	return out
}

// --- Create --------------------------------------------------------------

// CreateResult identifies the collection created by Create.
type CreateResult struct {
	ID   string
	Name string
	UID  string
}

// Create creates a collection in the given workspace from a Postman
// Collection v2.1.0 JSON document. If workspaceID is empty, the collection is
// created in the oldest personal Internal workspace the caller owns.
func (s *Service) Create(ctx context.Context, workspaceID string, collection []byte) (*CreateResult, error) {
	var schema api.CreateCollectionSchema
	if err := json.Unmarshal(collection, &schema); err != nil {
		return nil, fmt.Errorf("collections: decode collection: %w", err)
	}
	req := &api.CreateCollection{Collection: api.NewOptCreateCollectionSchema(schema)}
	params := api.CreateCollectionParams{Workspace: workspaceID}

	res, err := s.api.CreateCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionCreated:
		out := &CreateResult{}
		if c, ok := r.Collection.Get(); ok {
			out.ID, out.Name, out.UID = c.ID.Or(""), c.Name.Or(""), c.UID.Or("")
		}
		return out, nil
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- ForkedByUser --------------------------------------------------------

// ForkedByUserInput holds pagination options for ForkedByUser.
type ForkedByUserInput struct {
	// Cursor pages through results (use ForkedByUserResult.NextCursor).
	Cursor    string
	Limit     int
	Direction SortDirection
}

// UserForkedCollection is a single fork owned by the authenticated user.
type UserForkedCollection struct {
	ForkID    string
	ForkName  string
	SourceID  string
	CreatedAt string
}

// ForkedByUserResult is the paginated result of ForkedByUser.
type ForkedByUserResult struct {
	Forks            []UserForkedCollection
	Total            int
	InaccessibleFork int
	NextCursor       string
}

// ForkedByUser lists all of the authenticated user's forked collections.
func (s *Service) ForkedByUser(ctx context.Context, in *ForkedByUserInput) (*ForkedByUserResult, error) {
	if in == nil {
		in = &ForkedByUserInput{}
	}
	params := api.GetCollectionsForkedByUserParams{}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}

	res, err := s.api.GetCollectionsForkedByUser(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UsersForkedCollections:
		out := &ForkedByUserResult{}
		for _, d := range r.Data {
			out.Forks = append(out.Forks, UserForkedCollection{
				ForkID:    d.ForkId.Or(""),
				ForkName:  d.ForkName.Or(""),
				SourceID:  d.SourceId.Or(""),
				CreatedAt: d.CreatedAt.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Total = meta.Total.Or(0)
			out.InaccessibleFork = meta.InaccessibleFork.Or(0)
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetCollectionsForkedByUserBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Fork ------------------------------------------------------------------

// ForkInput specifies where to create a collection fork.
type ForkInput struct {
	// Workspace is the ID of the workspace to create the fork in. Required.
	Workspace string
	// Label distinguishes the fork from its parent collection.
	Label string
}

// ForkResult identifies the collection fork created by Fork.
type ForkResult struct {
	ID   string
	Name string
	UID  string
	Fork *ForkOrigin
}

// Fork creates a fork of an existing collection into a workspace.
func (s *Service) Fork(ctx context.Context, collectionID string, in *ForkInput) (*ForkResult, error) {
	if in == nil {
		in = &ForkInput{}
	}
	req := &api.CreateCollectionFork{Label: in.Label}
	params := api.CreateCollectionForkParams{CollectionId: collectionID, Workspace: in.Workspace}

	res, err := s.api.CreateCollectionFork(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionForkCreated:
		out := &ForkResult{}
		if c, ok := r.Collection.Get(); ok {
			out.ID, out.Name, out.UID = c.ID.Or(""), c.Name.Or(""), c.UID.Or("")
			if f, ok := c.Fork.Get(); ok {
				out.Fork = &ForkOrigin{Label: f.Label.Or(""), CreatedAt: f.CreatedAt.Or(""), From: f.From.Or("")}
			}
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- MergeFork ---------------------------------------------------------

// MergeForkInput specifies a fork merge.
type MergeForkInput struct {
	Source      string
	Destination string
	// Strategy controls how conflicts are resolved.
	Strategy MergeStrategy
}

// MergeForkResult identifies the collection produced by MergeFork.
type MergeForkResult struct {
	ID  string
	UID string
}

// MergeFork merges a forked collection back into its parent collection. The
// caller must have the Editor role on the collection.
//
// Deprecated: Postman has deprecated this endpoint; use MergeForkAsync instead.
func (s *Service) MergeFork(ctx context.Context, in *MergeForkInput) (*MergeForkResult, error) {
	if in == nil {
		in = &MergeForkInput{}
	}
	req := &api.MergeCollectionFork{Source: in.Source, Destination: in.Destination}
	if in.Strategy != "" {
		req.Strategy = api.NewOptMergeCollectionForkStrategy(api.MergeCollectionForkStrategy(in.Strategy))
	}

	res, err := s.api.MergeCollectionFork(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionForkMerged:
		out := &MergeForkResult{}
		if c, ok := r.Collection.Get(); ok {
			out.ID, out.UID = c.ID.Or(""), c.UID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- MergeForkAsync / MergeForkAsyncStatus --------------------------------

// MergeForkAsyncInput requests an asynchronous merge or pull between a forked
// collection and its parent.
type MergeForkAsyncInput struct {
	// Source is the source collection's ID. To pull changes into a fork, this
	// is the parent collection's ID.
	Source string
	// Destination is the destination collection's ID. To pull changes into a
	// fork, this is the forked collection's ID.
	Destination string
	// Strategy controls how conflicts are resolved. Defaults to
	// MergeStrategyDefault when empty.
	Strategy MergeStrategy
}

// MergeTask identifies an asynchronous merge or pull task.
type MergeTask struct {
	ID     string
	Status string
}

// MergeForkAsync merges a forked (source) collection and its parent
// (destination) collection asynchronously. To pull changes into a fork, pass
// the forked collection's ID as Destination and the parent collection's ID as
// Source. Use MergeForkAsyncStatus to track the returned task.
func (s *Service) MergeForkAsync(ctx context.Context, in *MergeForkAsyncInput) (*MergeTask, error) {
	if in == nil {
		in = &MergeForkAsyncInput{}
	}
	strategy := in.Strategy
	if strategy == "" {
		strategy = MergeStrategyDefault
	}
	req := &api.MergePullCollectionChanges{
		Strategy:    api.MergePullCollectionChangesStrategy(strategy),
		Source:      in.Source,
		Destination: in.Destination,
	}

	res, err := s.api.AsyncMergePullCollectionFork(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.AsyncMergePullCollectionForkOkResponse:
		out := &MergeTask{}
		if t, ok := r.Task.Get(); ok {
			out.ID, out.Status = t.ID.Or(""), t.Status.Or("")
		}
		return out, nil
	case *api.AsyncMergePullCollectionForkBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.AsyncMergePullCollectionForkForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// MergeTaskStatus is the status of an asynchronous merge or pull task.
type MergeTaskStatus struct {
	ID     string
	Status AsyncTaskStatus
	// ErrorMessage is set when Status is AsyncTaskFailed.
	ErrorMessage string
}

// MergeForkAsyncStatus gets the status of a task started by MergeForkAsync.
// After the task finishes, its status remains available for 24 hours;
// afterwards this returns a not-found error.
func (s *Service) MergeForkAsyncStatus(ctx context.Context, taskID string) (*MergeTaskStatus, error) {
	params := api.AsyncMergePullCollectionTaskStatusParams{TaskId: taskID}

	res, err := s.api.AsyncMergePullCollectionTaskStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TaskStatusResponse:
		out := &MergeTaskStatus{ID: r.ID.Or(""), Status: AsyncTaskStatus(r.Status.Or(""))}
		if d, ok := r.Details.Get(); ok {
			if e, ok := d.Error.Get(); ok {
				out.ErrorMessage = e.Message.Or("")
			}
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.AsyncMergePullCollectionTaskStatusForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.AsyncMergePullCollectionTaskStatusNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Get ---------------------------------------------------------------

// GetInput holds the options for Get.
type GetInput struct {
	// AccessKey is a collection's read-only access key. When set, no API key
	// is required to call this method.
	AccessKey string
	// Model, when CollectionModelMinimal, returns only root-level request and
	// folder IDs instead of the full collection element tree.
	Model CollectionModel
}

// GetResult holds a collection's contents.
type GetResult struct {
	// Collection is the collection in Postman Collection v2.1.0 JSON format.
	Collection []byte
}

// Get returns a collection's contents.
func (s *Service) Get(ctx context.Context, collectionID string, in *GetInput) (*GetResult, error) {
	if in == nil {
		in = &GetInput{}
	}
	params := api.GetCollectionParams{CollectionId: collectionID}
	if in.AccessKey != "" {
		params.AccessKey = api.NewOptString(in.AccessKey)
	}
	if in.Model != "" {
		params.Model = api.NewOptCollectionModelQuery(api.CollectionModelQuery(in.Model))
	}

	res, err := s.api.GetCollection(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionInformation:
		out := &GetResult{}
		if c, ok := r.Collection.Get(); ok {
			b, err := json.Marshal(&c)
			if err != nil {
				return nil, fmt.Errorf("collections: encode collection: %w", err)
			}
			out.Collection = b
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

// --- Replace -------------------------------------------------------------

// ReplaceResult acknowledges a successful Replace. Postman's API does not
// return collection details for this endpoint.
type ReplaceResult struct{}

// Replace replaces a collection's contents from a Postman Collection v2.1.0
// JSON document. Include the collection's existing id/uid values in the
// document; otherwise this endpoint removes the existing items and recreates
// them with new IDs. The maximum accepted document size is 100 MB.
func (s *Service) Replace(ctx context.Context, collectionID string, collection []byte) (*ReplaceResult, error) {
	var schema api.ModifyCollectionSchema
	if err := json.Unmarshal(collection, &schema); err != nil {
		return nil, fmt.Errorf("collections: decode collection: %w", err)
	}
	req := &api.ReplaceCollectionData{Collection: api.NewOptModifyCollectionSchema(schema)}
	params := api.PutCollectionParams{CollectionId: collectionID}

	res, err := s.api.PutCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PutCollectionOK:
		return &ReplaceResult{}, nil
	case *api.CollectionPut400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CollectionPut404Errors:
		return nil, postmanerr.Empty(http.StatusNotFound)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Update ----------------------------------------------------------------

// UpdateResult identifies the collection updated by Update.
type UpdateResult struct {
	ID          string
	Name        string
	Description string
}

// Update updates specific collection information (such as name, events, or
// variables) from a partial Postman Collection v2.1.0 JSON document.
func (s *Service) Update(ctx context.Context, collectionID string, collection []byte) (*UpdateResult, error) {
	req := &api.UpdateCollection{Collection: collection}
	params := api.PatchCollectionParams{CollectionId: collectionID}

	res, err := s.api.PatchCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PatchCollectionOkResponse:
		out := &UpdateResult{}
		if c, ok := r.Collection.Get(); ok {
			out.ID, out.Name, out.Description = c.ID.Or(""), c.Name.Or(""), c.Description.Or("")
		}
		return out, nil
	case *api.CollectionPatch400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete ------------------------------------------------------------

// DeleteResult identifies the collection removed by Delete.
type DeleteResult struct {
	ID  string
	UID string
}

// Delete deletes a collection.
func (s *Service) Delete(ctx context.Context, collectionID string) (*DeleteResult, error) {
	params := api.DeleteCollectionParams{CollectionId: collectionID}

	res, err := s.api.DeleteCollection(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionDeleted:
		out := &DeleteResult{}
		if c, ok := r.Collection.Get(); ok {
			out.ID, out.UID = c.ID.Or(""), c.UID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Comments ------------------------------------------------------------

// Comment is a single comment left on a collection.
type Comment struct {
	ID        int
	ThreadID  int
	Status    string
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Body      string
}

// Comments returns all comments left by users on a collection.
func (s *Service) Comments(ctx context.Context, collectionID string) ([]Comment, error) {
	params := api.GetCollectionCommentsParams{CollectionId: collectionID}

	res, err := s.api.GetCollectionComments(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentResponseObject:
		out := make([]Comment, 0, len(r.Data))
		for _, d := range r.Data {
			out = append(out, Comment{
				ID:        d.ID.Or(0),
				ThreadID:  d.ThreadId.Or(0),
				Status:    d.Status.Or(""),
				CreatedBy: d.CreatedBy.Or(0),
				CreatedAt: d.CreatedAt.Or(""),
				UpdatedAt: d.UpdatedAt.Or(""),
				Body:      d.Body.Or(""),
			})
		}
		return out, nil
	case *api.GetCollectionCommentsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetCollectionCommentsForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.GetCollectionCommentsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetCollectionCommentsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// CommentResult is a created or updated comment.
type CommentResult struct {
	ID        int
	ThreadID  int
	CreatedBy int
	CreatedAt string
	UpdatedAt string
	Body      string
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

// CreateComment adds a comment to a collection. To reply to an existing
// comment, pass the parent comment's ID as threadID; pass 0 to start a new
// thread. The body accepts a maximum of 10,000 characters.
func (s *Service) CreateComment(ctx context.Context, collectionID, body string, threadID int) (*CommentResult, error) {
	req := &api.CommentCreate{Body: body}
	if threadID > 0 {
		req.ThreadId = api.NewOptInt(threadID)
	}
	params := api.CreateCollectionCommentParams{CollectionId: collectionID}

	res, err := s.api.CreateCollectionComment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentUpdatedCreatedObject:
		return commentResultFromAPI(r), nil
	case *api.CreateCollectionCommentUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.CreateCollectionCommentForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.CreateCollectionCommentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.CreateCollectionCommentInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UpdateComment updates a comment on a collection. The body accepts a maximum
// of 10,000 characters.
func (s *Service) UpdateComment(ctx context.Context, collectionID string, commentID int, body string) (*CommentResult, error) {
	req := &api.CommentUpdate{Body: body}
	params := api.UpdateCollectionCommentParams{CollectionId: collectionID, CommentId: strconv.Itoa(commentID)}

	res, err := s.api.UpdateCollectionComment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CommentUpdatedCreatedObject:
		return commentResultFromAPI(r), nil
	case *api.UpdateCollectionCommentUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.UpdateCollectionCommentForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.UpdateCollectionCommentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateCollectionCommentInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// DeleteComment deletes a comment from a collection. Deleting a thread's
// first comment deletes the entire thread.
func (s *Service) DeleteComment(ctx context.Context, collectionID string, commentID int) error {
	params := api.DeleteCollectionCommentParams{CollectionId: collectionID, CommentId: strconv.Itoa(commentID)}

	res, err := s.api.DeleteCollectionComment(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteCollectionCommentOK:
		return nil
	case *api.DeleteCollectionCommentUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.DeleteCollectionCommentForbidden:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.DeleteCollectionCommentNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteCollectionCommentInternalServerError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Duplicate / DuplicateStatus -----------------------------------------

// DuplicateInput specifies where to duplicate a collection.
type DuplicateInput struct {
	// Workspace is the ID of the destination workspace. Required.
	Workspace string
	// Suffix is appended to the duplicated collection's name.
	Suffix string
}

// DuplicateTask identifies an asynchronous collection duplication task.
type DuplicateTask struct {
	ID     string
	Status DuplicateTaskStatus
	// Reason explains a failed task.
	Reason string
}

func duplicateTaskFromAPI(r *api.DuplicateCollectionResponse) *DuplicateTask {
	out := &DuplicateTask{}
	if t, ok := r.Task.Get(); ok {
		out.ID = t.ID.Or("")
		out.Status = DuplicateTaskStatus(t.Status.Or(""))
		out.Reason = t.Reason.Or("")
	}
	return out
}

// Duplicate creates a duplicate of a collection in another workspace. Use
// DuplicateStatus to track the returned task's status.
func (s *Service) Duplicate(ctx context.Context, collectionID string, in *DuplicateInput) (*DuplicateTask, error) {
	if in == nil {
		in = &DuplicateInput{}
	}
	req := &api.DuplicateCollection{Workspace: in.Workspace}
	if in.Suffix != "" {
		req.Suffix = api.NewOptString(in.Suffix)
	}
	params := api.DuplicateCollectionParams{CollectionId: collectionID}

	res, err := s.api.DuplicateCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.DuplicateCollectionResponse:
		return duplicateTaskFromAPI(r), nil
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

// DuplicateStatus gets the status of a duplication task started by Duplicate.
func (s *Service) DuplicateStatus(ctx context.Context, taskID string) (*DuplicateTask, error) {
	params := api.GetDuplicateCollectionTaskStatusParams{TaskId: taskID}

	res, err := s.api.GetDuplicateCollectionTaskStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.DuplicateCollectionResponse:
		return duplicateTaskFromAPI(r), nil
	case *api.GetDuplicateCollectionTaskStatusBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetDuplicateCollectionTaskStatusNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Forks ---------------------------------------------------------------

// ForksInput holds pagination options for Forks.
type ForksInput struct {
	// Cursor pages through results (use ForksResult.NextCursor).
	Cursor    string
	Limit     int
	Direction SortDirection
}

// ForkedCollection is a single fork of a collection.
type ForkedCollection struct {
	ID        string
	Name      string
	CreatedAt string
	CreatedBy string
}

// ForksResult is the paginated result of Forks.
type ForksResult struct {
	Forks      []ForkedCollection
	Total      int
	NextCursor string
}

// Forks lists a collection's forked collections.
func (s *Service) Forks(ctx context.Context, collectionID string, in *ForksInput) (*ForksResult, error) {
	if in == nil {
		in = &ForksInput{}
	}
	params := api.GetCollectionForksParams{CollectionId: collectionID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}

	res, err := s.api.GetCollectionForks(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionForksInfo:
		out := &ForksResult{}
		for _, d := range r.Data {
			out.Forks = append(out.Forks, ForkedCollection{
				ID:        d.ForkId.Or(""),
				Name:      d.ForkName.Or(""),
				CreatedAt: d.CreatedAt.Or(""),
				CreatedBy: d.CreatedBy.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Total = meta.Total.Or(0)
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetCollectionForksNotFoundResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- PublishDocs / UnpublishDocs -----------------------------------------

// DocColorSettings customizes a documentation page's colors. Empty fields
// keep Postman's defaults.
type DocColorSettings struct {
	Highlight    string
	RightSidebar string
	TopBar       string
}

// DocMetaTag is a custom HTML <meta> tag rendered on a documentation page.
type DocMetaTag struct {
	Name  string
	Value string
}

// PublishDocsInput configures a collection's published documentation.
type PublishDocsInput struct {
	// EnvironmentID scopes the documentation to a specific environment.
	EnvironmentID string
	// Layout controls the documentation page's column layout.
	Layout DocLayout
	// CustomColor overrides the documentation's default color scheme.
	CustomColor DocColorSettings
	// MetaTags are custom HTML meta tags injected into the page.
	MetaTags []DocMetaTag
}

// PublishDocsResult describes a collection's published documentation.
type PublishDocsResult struct {
	Published     bool
	Layout        string
	PublishDate   string
	PublisherID   string
	EnvironmentID string
	CustomColor   DocColorSettings
	PublicURL     string
	ID            string
	CollectionID  string
	MetaTags      []DocMetaTag
}

// PublishDocs publishes a collection's documentation, making it publicly
// available to anyone with the link. Publishing is only supported for
// collections with HTTP requests, and the collection must not be attached to
// an API. For Free and Solo plans the caller must have edit permission on the
// collection; Enterprise teams with API Governance and Security enabled
// additionally require the Community Manager role.
func (s *Service) PublishDocs(ctx context.Context, collectionID string, in *PublishDocsInput) (*PublishDocsResult, error) {
	if in == nil {
		in = &PublishDocsInput{}
	}
	req := &api.PublishDocumentation{
		CustomColor: api.DocumentationColorSettings{},
	}
	if in.CustomColor.Highlight != "" {
		req.CustomColor.Highlight = api.NewOptString(in.CustomColor.Highlight)
	}
	if in.CustomColor.RightSidebar != "" {
		req.CustomColor.RightSidebar = api.NewOptString(in.CustomColor.RightSidebar)
	}
	if in.CustomColor.TopBar != "" {
		req.CustomColor.TopBar = api.NewOptString(in.CustomColor.TopBar)
	}
	for _, t := range in.MetaTags {
		req.Customization.MetaTags = append(req.Customization.MetaTags, api.DocumentationMetaTags{Name: t.Name, Value: t.Value})
	}
	if in.EnvironmentID != "" {
		req.EnvironmentUid = api.NewOptString(in.EnvironmentID)
	}
	if in.Layout != "" {
		req.DocumentationLayout = api.NewOptDocumentationLayout(api.DocumentationLayout(in.Layout))
	}
	params := api.PublishDocumentationParams{CollectionId: collectionID}

	res, err := s.api.PublishDocumentation(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PublishDocumentationResponse:
		out := &PublishDocsResult{
			Published:     r.Published.Or(false),
			Layout:        r.DocumentationLayout.Or(""),
			PublishDate:   r.PublishDate.Or(""),
			PublisherID:   r.PublisherId.Or(""),
			EnvironmentID: r.EnvironmentUid.Or(""),
			PublicURL:     r.PublicUrl.Or(""),
			ID:            r.ID.Or(""),
			CollectionID:  r.CollectionId.Or(""),
		}
		if c, ok := r.CustomColor.Get(); ok {
			out.CustomColor = DocColorSettings{Highlight: c.Highlight.Or(""), RightSidebar: c.RightSidebar.Or(""), TopBar: c.TopBar.Or("")}
		}
		if cu, ok := r.Customization.Get(); ok {
			for _, t := range cu.MetaTags {
				out.MetaTags = append(out.MetaTags, DocMetaTag{Name: t.Name, Value: t.Value})
			}
		}
		return out, nil
	case *api.PublishDocumentationBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.PublishDocumentationUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.PublishDocumentationForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.PublishDocumentationInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// UnpublishDocs unpublishes a collection's documentation.
func (s *Service) UnpublishDocs(ctx context.Context, collectionID string) error {
	params := api.UnpublishDocumentationParams{CollectionId: collectionID}

	res, err := s.api.UnpublishDocumentation(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.UnpublishDocumentationOK:
		return nil
	case *api.UnpublishDocumentationBadRequest:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UnpublishDocumentationUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.UnpublishDocumentationNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UnpublishDocumentationInternalServerError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Pull / PullRequests / CreatePullRequest ------------------------------

// PullResult identifies the collections involved in a Pull.
type PullResult struct {
	DestinationID string
	SourceID      string
}

// Pull pulls changes from a forked collection's parent (source) collection
// into the fork.
func (s *Service) Pull(ctx context.Context, collectionID string) (*PullResult, error) {
	params := api.PullCollectionChangesParams{CollectionId: collectionID}

	res, err := s.api.PullCollectionChanges(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionChangesPulled:
		out := &PullResult{}
		if c, ok := r.Collection.Get(); ok {
			out.DestinationID, out.SourceID = c.DestinationId.Or(""), c.SourceId.Or("")
		}
		return out, nil
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorNameMessageDetails:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// PullRequest describes a collection pull request.
type PullRequest struct {
	ID            string
	Title         string
	Description   string
	Status        PullRequestStatus
	SourceID      string
	DestinationID string
	Href          string
	Comment       string
	CreatedAt     string
	CreatedBy     string
	UpdatedAt     string
	UpdatedBy     string
}

// PullRequests returns a collection's pull requests, including their status
// and a URL link to each.
func (s *Service) PullRequests(ctx context.Context, collectionID string) ([]PullRequest, error) {
	params := api.GetCollectionPullRequestsParams{CollectionId: collectionID}

	res, err := s.api.GetCollectionPullRequests(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionPullRequests:
		out := make([]PullRequest, 0, len(r.Data))
		for _, d := range r.Data {
			out = append(out, PullRequest{
				ID:            d.ID.Or(""),
				Title:         d.Title.Or(""),
				Description:   d.Description.Or(""),
				Status:        PullRequestStatus(d.Status.Or("")),
				SourceID:      d.SourceId.Or(""),
				DestinationID: d.DestinationId.Or(""),
				Href:          d.Href.Or(""),
				Comment:       d.Comment.Or(""),
				CreatedAt:     d.CreatedAt.Or(""),
				CreatedBy:     d.CreatedBy.Or(""),
				UpdatedAt:     d.UpdatedAt.Or(""),
				UpdatedBy:     d.UpdatedBy.Or(""),
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

// CreatePullRequestInput specifies a new collection pull request.
type CreatePullRequestInput struct {
	Title       string
	Description string
	Reviewers   []string
	// DestinationID is the parent collection's ID. Required.
	DestinationID string
}

// CreatePullRequest creates a pull request for a forked collection into its
// parent collection.
func (s *Service) CreatePullRequest(ctx context.Context, collectionID string, in *CreatePullRequestInput) (*PullRequest, error) {
	if in == nil {
		in = &CreatePullRequestInput{}
	}
	req := &api.CreatePullRequest{
		Title:         in.Title,
		Reviewers:     in.Reviewers,
		DestinationId: in.DestinationID,
	}
	if in.Description != "" {
		req.Description = api.NewOptString(in.Description)
	}
	params := api.CreateCollectionPullRequestParams{CollectionId: collectionID}

	res, err := s.api.CreateCollectionPullRequest(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PullRequestCreated:
		return &PullRequest{
			ID:            r.ID.Or(""),
			Title:         r.Title.Or(""),
			Description:   r.Description.Or(""),
			Status:        PullRequestStatus(r.Status.Or("")),
			SourceID:      r.SourceId.Or(""),
			DestinationID: r.DestinationId.Or(""),
			CreatedAt:     r.CreatedAt.Or(""),
			CreatedBy:     r.CreatedBy.Or(""),
			UpdatedAt:     r.UpdatedAt.Or(""),
		}, nil
	case *api.CreateCollectionPullRequestBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateCollectionPullRequestForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Roles / UpdateRoles -------------------------------------------------

// RoleAssignment grants an access level to a single user, group, or team.
type RoleAssignment struct {
	ID   int
	Role Role
}

// RolesResult lists the principals with access to a collection.
type RolesResult struct {
	Groups []RoleAssignment
	Teams  []RoleAssignment
	Users  []RoleAssignment
}

// Roles returns the IDs of all users, teams, and groups with access to view
// or edit a collection.
func (s *Service) Roles(ctx context.Context, collectionID string) (*RolesResult, error) {
	params := api.GetCollectionRolesParams{CollectionId: collectionID}

	res, err := s.api.GetCollectionRoles(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionRolesInfo:
		out := &RolesResult{}
		for _, g := range r.Group {
			out.Groups = append(out.Groups, RoleAssignment{ID: g.ID.Or(0), Role: Role(g.Role.Or(""))})
		}
		for _, t := range r.Team {
			out.Teams = append(out.Teams, RoleAssignment{ID: t.ID.Or(0), Role: Role(t.Role.Or(""))})
		}
		for _, u := range r.User {
			out.Users = append(out.Users, RoleAssignment{ID: u.ID.Or(0), Role: Role(u.Role.Or(""))})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTitleType:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// RoleUpdate replaces the role assignments for one kind of principal
// (RolePathUser, RolePathGroup, or RolePathTeam).
type RoleUpdate struct {
	Path   RolePath
	Values []RoleAssignment
}

// UpdateRoles updates the roles of users, groups, or teams in a collection.
// Only collection Editors can call this; it does not support the external
// Partner or Guest roles.
func (s *Service) UpdateRoles(ctx context.Context, collectionID string, updates []RoleUpdate) error {
	req := &api.UpdateCollectionRoles{}
	for _, u := range updates {
		role := api.UpdateCollectionRolesRoles{
			Op:   api.RolesOpUpdate,
			Path: api.UpdateCollectionRolesRolesPath(u.Path),
		}
		for _, v := range u.Values {
			role.Value = append(role.Value, api.UpdateCollectionRolesRolesValue{
				ID:   v.ID,
				Role: api.ValueRole(v.Role),
			})
		}
		req.Roles = append(req.Roles, role)
	}
	params := api.UpdateCollectionRolesParams{CollectionId: collectionID}

	res, err := s.api.UpdateCollectionRoles(ctx, req, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.UpdateCollectionRolesOK:
		return nil
	case *api.ErrorTypeTitleDetailStatus:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTitleType:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- SourceStatus ----------------------------------------------------------

// SourceStatusResult reports whether a fork's parent collection has changes.
type SourceStatusResult struct {
	// IsSourceAhead is true when the parent (source) collection has changes
	// not yet pulled into the fork.
	IsSourceAhead bool
}

// SourceStatus checks whether a forked collection's parent (source)
// collection has changes not yet pulled into the fork. This may take a few
// minutes to reflect a recent change.
func (s *Service) SourceStatus(ctx context.Context, collectionID string) (*SourceStatusResult, error) {
	params := api.GetSourceCollectionStatusParams{CollectionId: collectionID}

	res, err := s.api.GetSourceCollectionStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SourceCollectionStatus:
		out := &SourceStatusResult{}
		if c, ok := r.Collection.Get(); ok {
			if u, ok := c.CollectionUid.Get(); ok {
				out.IsSourceAhead = u.IsSourceAhead.Or(false)
			}
		}
		return out, nil
	case *api.GetSourceCollectionStatusBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- TransformToOpenAPI ----------------------------------------------------

// TransformToOpenAPI transforms a collection into a stringified OpenAPI
// definition; it does not create an API. Format defaults to
// TransformFormatJSON when empty.
func (s *Service) TransformToOpenAPI(ctx context.Context, collectionID string, format TransformFormat) (string, error) {
	params := api.TransformCollectionToOpenApiParams{CollectionId: collectionID}
	if format != "" {
		params.Format = api.NewOptCollectionTransformFormat(api.CollectionTransformFormat(format))
	}

	res, err := s.api.TransformCollectionToOpenApi(ctx, params)
	if err != nil {
		return "", err
	}

	switch r := res.(type) {
	case *api.CollectionTransformed:
		return r.Output.Or(""), nil
	case *api.TransformCollectionToOpenApiUnauthorized:
		return "", postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.TransformCollectionToOpenApiNotFound:
		return "", postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.TransformCollectionToOpenApiInternalServerError:
		return "", postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return "", errUnexpectedResponse(res)
	}
}

// --- TransferFolders / TransferRequests / TransferResponses ---------------

// TransferTarget identifies the destination of a transfer.
type TransferTarget struct {
	ID    string
	Model TransferTargetModel
}

// TransferLocation places transferred items within the target. Leave ID and
// Model empty to place items at the target's root.
type TransferLocation struct {
	// ID is the sibling item's ID that Position is relative to.
	ID string
	// Model is the sibling item's kind ("folder", "request", or "response").
	Model    string
	Position TransferPosition
}

// TransferInput specifies items to copy or move into a collection, folder, or
// request.
type TransferInput struct {
	// IDs are the folder, request, or response IDs to transfer.
	IDs      []string
	Mode     TransferMode
	Target   TransferTarget
	Location TransferLocation
}

// TransferResult lists the IDs of the transferred items.
type TransferResult struct {
	IDs []string
}

func buildTransferItems(in *TransferInput) *api.TransferCollectionItems {
	if in == nil {
		in = &TransferInput{}
	}
	req := &api.TransferCollectionItems{
		Ids:  in.IDs,
		Mode: api.Mode(in.Mode),
		Target: api.Target{
			ID:    in.Target.ID,
			Model: api.TargetModel(in.Target.Model),
		},
		Location: api.Location{
			Position: api.Position(in.Location.Position),
		},
	}
	if in.Location.ID != "" {
		req.Location.ID = api.NewOptNilString(in.Location.ID)
	}
	if in.Location.Model != "" {
		req.Location.Model = api.NewOptNilString(in.Location.Model)
	}
	return req
}

// TransferFolders copies or moves folders into a collection or folder.
func (s *Service) TransferFolders(ctx context.Context, in *TransferInput) (*TransferResult, error) {
	res, err := s.api.TransferCollectionFolders(ctx, buildTransferItems(in))
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionItemsTransferred:
		return &TransferResult{IDs: r.Ids}, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// TransferRequests copies or moves requests into a collection or folder.
func (s *Service) TransferRequests(ctx context.Context, in *TransferInput) (*TransferResult, error) {
	res, err := s.api.TransferCollectionRequests(ctx, buildTransferItems(in))
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionItemsTransferred:
		return &TransferResult{IDs: r.Ids}, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// TransferResponses copies or moves responses into a request.
func (s *Service) TransferResponses(ctx context.Context, in *TransferInput) (*TransferResult, error) {
	res, err := s.api.TransferCollectionResponses(ctx, buildTransferItems(in))
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CollectionItemsTransferred:
		return &TransferResult{IDs: r.Ids}, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateStatus ------------------------------------------------------

// UpdateStatusResult reports the status of an asynchronous collection update
// task.
type UpdateStatusResult struct {
	ID     string
	Status AsyncTaskStatus
}

// UpdateStatus gets the status of an asynchronous collection update task
// (a PUT to a collection performed with the Prefer: respond-async header).
// Note: this SDK always performs Replace synchronously, since the generated
// API client does not model the 202 Accepted response or the Prefer header;
// this method remains available for tracking a task ID obtained by other
// means (e.g. the Postman API called directly, or another SDK).
func (s *Service) UpdateStatus(ctx context.Context, taskID string) (*UpdateStatusResult, error) {
	params := api.GetCollectionUpdatesTasksParams{TaskId: taskID}

	res, err := s.api.GetCollectionUpdatesTasks(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetCollectionUpdateStatus:
		return &UpdateStatusResult{ID: r.ID.Or(""), Status: AsyncTaskStatus(r.Status.Or(""))}, nil
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

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
