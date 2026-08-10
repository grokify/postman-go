// Package environments provides a high-level client for Postman's
// Environments API.
//
// Environments store variables (plain or secret-backed) that can be applied
// to requests within a workspace. This package also covers environment
// forking: creating a fork, listing an environment's forks, merging a fork
// back into its parent, and pulling changes from a parent into a fork.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	envs, _ := client.Environments().List(ctx, &environments.ListInput{Workspace: "ws-id"})
package environments

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// VariableType is the type of an environment variable.
type VariableType string

// Variable type values.
const (
	// VariableTypeSecret indicates the variable's value is masked.
	VariableTypeSecret VariableType = "secret"
	// VariableTypeDefault indicates the variable's value is visible in plain text.
	VariableTypeDefault VariableType = "default"
)

// Direction sorts paginated results in ascending or descending order.
type Direction string

// Direction values.
const (
	DirectionAsc  Direction = "asc"
	DirectionDesc Direction = "desc"
)

// Sort selects the field results are sorted by.
type Sort string

// Sort values.
const (
	// SortCreatedAt sorts by the date and time of creation.
	SortCreatedAt Sort = "createdAt"
)

// Service is the high-level Environments client. Obtain one via
// postman.Client.Environments.
type Service struct {
	api *api.Client
}

// New creates an Environments service over the given generated API client.
// Most callers should use postman.Client.Environments instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Variable describes an environment variable. Postman's API represents this
// as one of two overlapping shapes (a plain variable, or one backed by a
// vault secret); this type is a superset of both.
type Variable struct {
	// Enabled is true if the variable is enabled.
	Enabled bool `json:"enabled,omitempty"`
	// Key is the variable's name.
	Key string `json:"key,omitempty"`
	// Value is the variable's value.
	Value string `json:"value,omitempty"`
	// Type is the variable's type.
	Type VariableType `json:"type,omitempty"`
	// Secret is true if the variable is marked as secret, in which case its
	// value is retrieved from the provider named in Source.
	Secret bool `json:"secret,omitempty"`
	// Description is the variable's description (up to 512 characters).
	Description string `json:"description,omitempty"`
	// Source describes where a secret variable's value comes from. Only set
	// for secret variables.
	Source *VariableSource `json:"source,omitempty"`
}

// VariableSource describes the source of a secret variable's value.
type VariableSource struct {
	// Provider is the secret's provider.
	Provider string `json:"provider,omitempty"`
	// Postman holds Postman Vault-specific source information.
	Postman *VariableSourcePostman `json:"postman,omitempty"`
}

// VariableSourcePostman describes a variable's value stored in the Postman
// Vault.
type VariableSourcePostman struct {
	// SecretID is the variable's secret ID.
	SecretID string `json:"secretId,omitempty"`
	// Type is the variable's type, e.g. "cloud".
	Type string `json:"type,omitempty"`
	// VaultID is the variable's ID in the Postman Vault.
	VaultID string `json:"vaultId,omitempty"`
}

func marshalVariables[T ~[]byte](values []Variable) ([]T, error) {
	out := make([]T, 0, len(values))
	for _, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		out = append(out, T(b))
	}
	return out, nil
}

func unmarshalVariable(raw []byte) (Variable, error) {
	var v Variable
	if len(raw) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return Variable{}, err
	}
	return v, nil
}

func unmarshalVariables[T ~[]byte](raws []T) ([]Variable, error) {
	out := make([]Variable, 0, len(raws))
	for _, raw := range raws {
		v, err := unmarshalVariable([]byte(raw))
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// --- List ---------------------------------------------------------------

// ListInput holds the filters for List.
type ListInput struct {
	// Workspace limits results to environments in this workspace's ID. If
	// empty, all environments visible to the caller are returned.
	Workspace string
}

// ListResult is the result of listing environments.
type ListResult struct {
	Environments []EnvironmentSummary
}

// EnvironmentSummary is a single environment as returned by List.
type EnvironmentSummary struct {
	ID        string
	Name      string
	Owner     string
	CreatedAt string
	UpdatedAt string
	UID       string
	IsPublic  bool
}

// List returns information about all of the caller's environments.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetEnvironmentsParams{}
	if in.Workspace != "" {
		params.Workspace = api.NewOptString(in.Workspace)
	}

	res, err := s.api.GetEnvironments(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetEnvironmentsOkResponse:
		out := &ListResult{}
		for _, e := range r.Environments {
			out.Environments = append(out.Environments, EnvironmentSummary{
				ID:        e.ID.Or(""),
				Name:      e.Name.Or(""),
				CreatedAt: e.CreatedAt.Or(""),
				UpdatedAt: e.UpdatedAt.Or(""),
				Owner:     e.Owner.Or(""),
				UID:       e.UID.Or(""),
				IsPublic:  e.IsPublic.Or(false),
			})
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Environments404Error:
		return nil, postmanerr.Empty(http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create ---------------------------------------------------------------

// CreateInput holds the fields for creating an environment.
type CreateInput struct {
	// Workspace is the ID of the workspace to create the environment in.
	// Required. If empty when calling Create, the environment is created in
	// the oldest personal Internal workspace the caller owns.
	Workspace string
	// Name is the environment's name.
	Name string
	// Values holds the environment's variables.
	Values []Variable
}

// CreateResult is returned after creating an environment.
type CreateResult struct {
	ID   string
	Name string
	UID  string
}

// Create creates an environment. The request body (including all variable
// values) cannot exceed 30MB.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*CreateResult, error) {
	if in == nil {
		in = &CreateInput{}
	}
	values, err := marshalVariables[api.CreateEnvironmentEnvironmentValues](in.Values)
	if err != nil {
		return nil, err
	}
	req := &api.CreateEnvironment{
		Environment: api.NewOptCreateEnvironmentEnvironment(api.CreateEnvironmentEnvironment{
			Name:   in.Name,
			Values: values,
		}),
	}
	params := api.CreateEnvironmentParams{Workspace: in.Workspace}

	res, err := s.api.CreateEnvironment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.EnvironmentCreated:
		out := &CreateResult{}
		if env, ok := r.Environment.Get(); ok {
			out.ID = env.ID.Or("")
			out.Name = env.Name.Or("")
			out.UID = env.UID.Or("")
		}
		return out, nil
	case *api.Environment400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateEnvironmentLengthRequired:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusLengthRequired)
	case *api.CreateEnvironmentRequestEntityTooLarge:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusRequestEntityTooLarge)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Get --------------------------------------------------------------------

// GetResult is the result of Get.
type GetResult struct {
	ID        string
	Name      string
	Owner     string
	CreatedAt string
	UpdatedAt string
	IsPublic  bool
	Values    []Variable
}

// Get returns information about an environment, including its variables.
func (s *Service) Get(ctx context.Context, environmentID string) (*GetResult, error) {
	params := api.GetEnvironmentParams{EnvironmentId: environmentID}

	res, err := s.api.GetEnvironment(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetEnvironmentOkResponse:
		out := &GetResult{}
		if env, ok := r.Environment.Get(); ok {
			out.ID = env.ID.Or("")
			out.Name = env.Name.Or("")
			out.Owner = env.Owner.Or("")
			out.CreatedAt = env.CreatedAt.Or("")
			out.UpdatedAt = env.UpdatedAt.Or("")
			out.IsPublic = env.IsPublic.Or(false)
			values, err := unmarshalVariables[api.GetEnvironmentInfoValues](env.Values)
			if err != nil {
				return nil, err
			}
			out.Values = values
		}
		return out, nil
	case *api.Environments404Error:
		// Note: the generated client binds this schema to HTTP 400 for this
		// operation, matching the underlying OpenAPI spec.
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Replace ------------------------------------------------------------

// ReplaceInput holds the fields for replacing an environment's contents.
type ReplaceInput struct {
	// Name is the environment's new name.
	Name string
	// Values replaces the environment's variables.
	Values []Variable
}

// ReplaceResult is returned after replacing an environment's contents.
type ReplaceResult struct {
	ID   string
	Name string
	UID  string
}

// Replace replaces all the contents of an environment with the given
// information. The request body cannot exceed 30MB.
func (s *Service) Replace(ctx context.Context, environmentID string, in *ReplaceInput) (*ReplaceResult, error) {
	if in == nil {
		in = &ReplaceInput{}
	}
	values, err := marshalVariables[api.ReplaceEnvironmentDataEnvironmentValues](in.Values)
	if err != nil {
		return nil, err
	}
	req := &api.ReplaceEnvironmentData{
		Environment: api.NewOptReplaceEnvironmentDataEnvironment(api.ReplaceEnvironmentDataEnvironment{
			Name:   api.NewOptString(in.Name),
			Values: values,
		}),
	}
	params := api.PutEnvironmentParams{EnvironmentId: environmentID}

	res, err := s.api.PutEnvironment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PutEnvironmentOkResponse:
		out := &ReplaceResult{}
		if env, ok := r.Environment.Get(); ok {
			out.ID = env.ID.Or("")
			out.Name = env.Name.Or("")
			out.UID = env.UID.Or("")
		}
		return out, nil
	case *api.Environment400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.PutEnvironmentLengthRequired:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusLengthRequired)
	case *api.PutEnvironmentRequestEntityTooLarge:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusRequestEntityTooLarge)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Patch ------------------------------------------------------------------

// PatchOp is a single JSON Patch-style operation used by Patch. Value is
// required for "add" and "replace" ops (a Variable for "add", a string for
// "replace" or when Path is "/name") and must be omitted for "remove".
type PatchOp struct {
	// Op is the operation: "add", "replace", or "remove".
	Op string `json:"op"`
	// Path is the JSON Pointer (RFC 6901) indicating the entry to update, in
	// "/values/#" format for a variable at index #, or "/name" for the
	// environment's name.
	Path string `json:"path"`
	// Value is the new value. Its shape depends on Op and Path: a Variable
	// value for "add", or a string for "replace"/"/name".
	Value any `json:"value,omitempty"`
}

// PatchInput holds the patch operations for Patch.
type PatchInput struct {
	Ops []PatchOp
}

// PatchResult is the environment as returned after a Patch call.
type PatchResult struct {
	ID        string
	Name      string
	Owner     string
	CreatedAt string
	UpdatedAt string
	UID       string
	Values    []Variable
}

// Patch updates specific environment properties, such as its name and
// variables. Only one type of operation may be performed per call (for
// example, you cannot combine an "add" and a "replace" in the same request).
// The request body cannot exceed 30MB.
func (s *Service) Patch(ctx context.Context, environmentID string, in *PatchInput) (*PatchResult, error) {
	if in == nil {
		in = &PatchInput{}
	}
	body, err := json.Marshal(in.Ops)
	if err != nil {
		return nil, err
	}
	params := api.PatchEnvironmentParams{EnvironmentId: environmentID}

	res, err := s.api.PatchEnvironment(ctx, api.PatchEnvironment(body), params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PatchEnvironmentOkResponse:
		out := &PatchResult{}
		if env, ok := r.Environment.Get(); ok {
			out.ID = env.ID.Or("")
			out.Name = env.Name.Or("")
			out.Owner = env.Owner.Or("")
			out.CreatedAt = env.CreatedAt.Or("")
			out.UpdatedAt = env.UpdatedAt.Or("")
			out.UID = env.UID.Or("")
			values, err := unmarshalVariables[api.PatchEnvironmentInfoValues](env.Values)
			if err != nil {
				return nil, err
			}
			out.Values = values
		}
		return out, nil
	case *api.PatchEnvironmentBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.PatchEnvironmentForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Environments404Error:
		return nil, postmanerr.Empty(http.StatusNotFound)
	case *api.PatchEnvironmentLengthRequired:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusLengthRequired)
	case *api.PatchEnvironmentRequestEntityTooLarge:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusRequestEntityTooLarge)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete -------------------------------------------------------------

// DeleteResult is returned after deleting an environment.
type DeleteResult struct {
	ID  string
	UID string
}

// Delete deletes an environment.
func (s *Service) Delete(ctx context.Context, environmentID string) (*DeleteResult, error) {
	params := api.DeleteEnvironmentParams{EnvironmentId: environmentID}

	res, err := s.api.DeleteEnvironment(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.EnvironmentDeleted:
		out := &DeleteResult{}
		if env, ok := r.Environment.Get(); ok {
			out.ID = env.ID.Or("")
			out.UID = env.UID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Environments404Error:
		return nil, postmanerr.Empty(http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Forks --------------------------------------------------------------

// ForksInput holds the filters and pagination options for Forks.
type ForksInput struct {
	// Cursor is the pagination cursor (use ForksResult.NextCursor to page).
	Cursor string
	// Direction sorts results in ascending or descending order.
	Direction Direction
	// Limit is the maximum number of rows to return.
	Limit int
	// Sort orders the results by the given field.
	Sort Sort
}

// ForksResult is the paginated set of an environment's forks.
type ForksResult struct {
	Forks      []EnvironmentFork
	Total      int
	NextCursor string
}

// EnvironmentFork describes a single forked environment.
type EnvironmentFork struct {
	ForkID    string
	ForkName  string
	CreatedAt string
	CreatedBy string
	UpdatedAt string
}

// Forks returns all of an environment's forked environments.
func (s *Service) Forks(ctx context.Context, environmentID string, in *ForksInput) (*ForksResult, error) {
	if in == nil {
		in = &ForksInput{}
	}
	params := api.GetEnvironmentForksParams{EnvironmentId: environmentID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Direction != "" {
		params.Direction = api.NewOptAscDesc(api.AscDesc(in.Direction))
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Sort != "" {
		params.Sort = api.NewOptSortByCreatedAt(api.SortByCreatedAt(in.Sort))
	}

	res, err := s.api.GetEnvironmentForks(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetEnvironmentForksOkResponse:
		out := &ForksResult{}
		for _, f := range r.Data {
			out.Forks = append(out.Forks, EnvironmentFork{
				ForkID:    f.ForkId.Or(""),
				ForkName:  f.ForkName.Or(""),
				CreatedAt: f.CreatedAt.Or(""),
				CreatedBy: f.CreatedBy.Or(""),
				UpdatedAt: f.UpdatedAt.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Total = meta.Total.Or(0)
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Fork -----------------------------------------------------------------

// ForkInput holds the fields for forking an environment.
type ForkInput struct {
	// WorkspaceID is the ID of the workspace to create the fork in. Required.
	WorkspaceID string
	// ForkName is the new fork's name.
	ForkName string
}

// ForkResult is returned after forking an environment.
type ForkResult struct {
	UID      string
	Name     string
	ForkName string
}

// Fork creates a fork of an existing environment.
func (s *Service) Fork(ctx context.Context, environmentID string, in *ForkInput) (*ForkResult, error) {
	if in == nil {
		in = &ForkInput{}
	}
	req := &api.ForkEnvironment{ForkName: in.ForkName}
	params := api.ForkEnvironmentParams{EnvironmentId: environmentID, WorkspaceId: in.WorkspaceID}

	res, err := s.api.ForkEnvironment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ForkEnvironmentOkResponse:
		out := &ForkResult{}
		if env, ok := r.Environment.Get(); ok {
			out.UID = env.UID.Or("")
			out.Name = env.Name.Or("")
			out.ForkName = env.ForkName.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Merge ------------------------------------------------------------------

// MergeInput holds the fields for merging a forked environment.
type MergeInput struct {
	// Source is the ID of the source (forked) environment to merge from.
	Source string
	// DeleteSource, if true, deletes the source environment after the merge.
	DeleteSource bool
}

// MergeResult is returned after merging a forked environment.
type MergeResult struct {
	UID string
}

// Merge merges a forked environment back into its parent environment.
func (s *Service) Merge(ctx context.Context, environmentID string, in *MergeInput) (*MergeResult, error) {
	if in == nil {
		in = &MergeInput{}
	}
	req := &api.MergeEnvironmentFork{Source: in.Source}
	if in.DeleteSource {
		req.DeleteSource = api.NewOptBool(true)
	}
	params := api.MergeEnvironmentForkParams{EnvironmentId: environmentID}

	res, err := s.api.MergeEnvironmentFork(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.MergeEnvironmentForkOkResponse:
		out := &MergeResult{}
		if env, ok := r.Environment.Get(); ok {
			out.UID = env.UID.Or("")
		}
		return out, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Pull -------------------------------------------------------------------

// PullInput holds the fields for pulling changes into a forked environment.
type PullInput struct {
	// Source is the ID of the source (parent) environment to pull from.
	Source string
}

// PullResult is returned after pulling changes into a forked environment.
type PullResult struct {
	UID string
}

// Pull pulls the changes from a parent (source) environment into the forked
// (destination) environment identified by environmentUID.
func (s *Service) Pull(ctx context.Context, environmentUID string, in *PullInput) (*PullResult, error) {
	if in == nil {
		in = &PullInput{}
	}
	req := &api.PullEnvironmentForkChanges{Source: in.Source}
	params := api.PullEnvironmentParams{EnvironmentUid: environmentUID}

	res, err := s.api.PullEnvironment(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.PullEnvironmentOkResponse:
		out := &PullResult{}
		if env, ok := r.Environment.Get(); ok {
			out.UID = env.UID.Or("")
		}
		return out, nil
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
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
