// Package components provides a high-level client for Postman's API Spec Hub
// Components API.
//
// Components are reusable pieces of an API definition (schemas, parameters,
// responses, and so on) that live in a team's component library. A component
// has a mutable draft and zero or more immutable published versions.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	list, _ := client.Components().List(ctx, nil)
package components

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// ComponentType is the specification type a component's content adheres to.
type ComponentType string

// Component type values.
const (
	ComponentTypeOAS2  ComponentType = "OAS2"
	ComponentTypeOAS3  ComponentType = "OAS3"
	ComponentTypeOAS31 ComponentType = "OAS3_1"
)

// ComponentStatus is a component's lifecycle state.
type ComponentStatus string

// Component status values.
const (
	// ComponentStatusActive components can be edited and published.
	ComponentStatusActive ComponentStatus = "active"
	// ComponentStatusArchived components are read-only; they can't be edited
	// or published, but their existing versions remain accessible.
	ComponentStatusArchived ComponentStatus = "archived"
)

// ContentFormat is the serialization format of a component's content.
type ContentFormat string

// Content format values.
const (
	ContentFormatJSON ContentFormat = "JSON"
	ContentFormatYAML ContentFormat = "YAML"
)

// SourceType selects the source to publish a new component version from.
type SourceType string

// Source type values.
const (
	// SourceTypeDraft publishes from the component's current draft.
	SourceTypeDraft SourceType = "draft"
)

// ComponentInclude selects additional fields to include in List and Get
// responses.
type ComponentInclude string

// Component include values.
const (
	ComponentIncludeHasVersions          ComponentInclude = "hasVersions"
	ComponentIncludeLatestVersion        ComponentInclude = "latestVersion"
	ComponentIncludeLatestVersionContent ComponentInclude = "latestVersion.content"
)

// ComponentExpand selects fields to expand in List and Get responses.
type ComponentExpand string

// Component expand values.
const (
	ComponentExpandLatestVersion ComponentExpand = "latestVersion"
)

// VersionInclude selects additional fields to include in Versions and
// Version responses.
type VersionInclude string

// Version include values.
const (
	VersionIncludeContent VersionInclude = "content"
)

// Service is the high-level Components client. Obtain one via
// postman.Client.Components.
type Service struct {
	api *api.Client
}

// New creates a Components service over the given generated API client. Most
// callers should use postman.Client.Components instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Component describes a component in the team's component library.
type Component struct {
	ID            string
	Name          string
	Type          ComponentType
	Status        ComponentStatus
	CreatedAt     string
	UpdatedAt     string
	CreatedBy     string
	UpdatedBy     string
	HasVersions   bool
	LatestVersion *ComponentVersion
}

// ComponentVersion describes a single published version of a component.
type ComponentVersion struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	URL         string `json:"url,omitempty"`
	Format      string `json:"format,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
	Content     string `json:"content,omitempty"`
}

// parseLatestVersion decodes a ComponentData.LatestVersion raw value, which
// the API returns either as a bare version-ID string, or (when expanded) as a
// full version object.
func parseLatestVersion(raw []byte) (*ComponentVersion, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		return &ComponentVersion{ID: id}, nil
	}
	var v ComponentVersion
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func componentFromAPI(d api.ComponentData) (Component, error) {
	lv, err := parseLatestVersion([]byte(d.LatestVersion))
	if err != nil {
		return Component{}, err
	}
	return Component{
		ID:            d.ID.Or(""),
		Name:          d.Name.Or(""),
		Type:          ComponentType(d.Type.Or("")),
		Status:        ComponentStatus(d.Status.Or("")),
		CreatedAt:     d.CreatedAt.Or(""),
		UpdatedAt:     d.UpdatedAt.Or(""),
		CreatedBy:     d.CreatedBy.Or(""),
		UpdatedBy:     d.UpdatedBy.Or(""),
		HasVersions:   d.HasVersions.Or(false),
		LatestVersion: lv,
	}, nil
}

func componentVersionFromAPI(v api.ComponentVersionData) ComponentVersion {
	return ComponentVersion{
		ID:          v.ID.Or(""),
		Label:       v.Label.Or(""),
		URL:         v.URL.Or(""),
		Format:      v.Format.Or(""),
		PublishedAt: v.PublishedAt.Or(""),
		Content:     v.Content.Or(""),
	}
}

func joinComponentIncludes(vs []ComponentInclude) string {
	ss := make([]string, len(vs))
	for i, v := range vs {
		ss[i] = string(v)
	}
	return strings.Join(ss, ",")
}

func joinComponentExpands(vs []ComponentExpand) string {
	ss := make([]string, len(vs))
	for i, v := range vs {
		ss[i] = string(v)
	}
	return strings.Join(ss, ",")
}

func joinVersionIncludes(vs []VersionInclude) string {
	ss := make([]string, len(vs))
	for i, v := range vs {
		ss[i] = string(v)
	}
	return strings.Join(ss, ",")
}

// --- List ---------------------------------------------------------------

// ListInput holds the filters for List.
type ListInput struct {
	// Type filters results by component type.
	Type ComponentType
	// Status filters results by the component's status.
	Status ComponentStatus
	// HasVersions, if true, returns only components with published versions.
	HasVersions bool
	// Include lists additional fields to include in the response.
	Include []ComponentInclude
	// Expand lists fields to expand in the response.
	Expand []ComponentExpand
}

// ListResult is the result of listing components.
type ListResult struct {
	Components []Component
	NextCursor string
}

// List returns a list of all components in the team's component library.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetAllComponentsParams{}
	if in.Type != "" {
		params.Type = api.NewOptComponentType(api.ComponentType(in.Type))
	}
	if in.Status != "" {
		params.Status = api.NewOptComponentStatus(api.ComponentStatus(in.Status))
	}
	if in.HasVersions {
		params.HasVersions = api.NewOptBool(true)
	}
	if len(in.Include) > 0 {
		params.Include = api.NewOptString(joinComponentIncludes(in.Include))
	}
	if len(in.Expand) > 0 {
		params.Expand = api.NewOptString(joinComponentExpands(in.Expand))
	}

	res, err := s.api.GetAllComponents(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAllComponents:
		out := &ListResult{}
		for _, d := range r.Data {
			c, err := componentFromAPI(d)
			if err != nil {
				return nil, err
			}
			out.Components = append(out.Components, c)
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create ---------------------------------------------------------------

// CreateInput holds the fields for creating a component.
type CreateInput struct {
	// Name is the component's name. Must be unique within the team, and can
	// only contain letters, digits, hyphens, underscores, and periods, up to
	// 60 characters.
	Name string
	// Type is the specification the component's content adheres to.
	Type ComponentType
	// Content is the component's content, up to 500 KB (UTF-8).
	Content string
	// Format is the component's content format.
	Format ContentFormat
}

// CreateResult is returned after creating a component.
type CreateResult struct {
	ID string
}

// Create creates a new component. The component is created in an active
// state with an initial draft; use CreateVersion to publish a version.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*CreateResult, error) {
	if in == nil {
		in = &CreateInput{}
	}
	nameJSON, err := json.Marshal(in.Name)
	if err != nil {
		return nil, err
	}
	req := &api.CreateComponent{
		Name:    nameJSON,
		Type:    api.ComponentType(in.Type),
		Content: in.Content,
	}
	if in.Format != "" {
		req.Format = api.NewOptComponentContentFormat(api.ComponentContentFormat(in.Format))
	}

	res, err := s.api.CreateComponent(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateComponentResponse:
		return &CreateResult{ID: r.ID.Or("")}, nil
	case *api.CreateComponentBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateComponentConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Get ----------------------------------------------------------------

// GetInput holds the options for Get.
type GetInput struct {
	// Include lists additional fields to include in the response.
	Include []ComponentInclude
	// Expand lists fields to expand in the response.
	Expand []ComponentExpand
}

// Get returns information about a component. Use Include and Expand to
// return additional information, such as HasVersions and the latest
// published version.
func (s *Service) Get(ctx context.Context, componentID string, in *GetInput) (*Component, error) {
	if in == nil {
		in = &GetInput{}
	}
	params := api.GetComponentParams{ComponentId: componentID}
	if len(in.Include) > 0 {
		params.Include = api.NewOptString(joinComponentIncludes(in.Include))
	}
	if len(in.Expand) > 0 {
		params.Expand = api.NewOptString(joinComponentExpands(in.Expand))
	}

	res, err := s.api.GetComponent(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetComponent:
		if d, ok := r.Data.Get(); ok {
			c, err := componentFromAPI(d)
			if err != nil {
				return nil, err
			}
			return &c, nil
		}
		return &Component{}, nil
	case *api.GetComponentBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetComponentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Update -----------------------------------------------------------------

// UpdateResult is the component as returned after an Update call.
type UpdateResult struct {
	ID     string
	Name   string
	Status ComponentStatus
}

// Update renames a component or changes its lifecycle status (active or
// archived). Only one of name or status can change per call.
//
// Known limitation: Postman's OpenAPI spec models this request body as an
// untyped union (a name-only patch or a status-only patch) that the
// generated client cannot resolve to concrete fields, so it always sends an
// empty JSON object ("{}") as the body regardless of intent. As a result
// this call cannot presently change the component's name or status; it
// still performs the request and surfaces the component's current state (or
// any error response) from the API.
func (s *Service) Update(ctx context.Context, componentID string) (*UpdateResult, error) {
	params := api.UpdateComponentParams{ComponentId: componentID}

	res, err := s.api.UpdateComponent(ctx, &api.UpdateComponentReq{}, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UpdateComponentResponse:
		return &UpdateResult{
			ID:     r.ID.Or(""),
			Name:   r.Name.Or(""),
			Status: ComponentStatus(r.Status.Or("")),
		}, nil
	case *api.UpdateComponentBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateComponentNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateComponentConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Draft ------------------------------------------------------------------

// DraftResult is a component's current working draft.
type DraftResult struct {
	Content string
	Format  string
}

// Draft returns the current working draft of a component, including its
// content and format. Drafts represent the latest unpublished edits, which
// may differ from the most recently published version.
func (s *Service) Draft(ctx context.Context, componentID string) (*DraftResult, error) {
	params := api.GetComponentDraftParams{ComponentId: componentID}

	res, err := s.api.GetComponentDraft(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetComponentDraft:
		return &DraftResult{Content: r.Content.Or(""), Format: r.Format.Or("")}, nil
	case *api.GetComponentDraftBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetComponentDraftNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateDraft --------------------------------------------------------

// UpdateDraftInput holds the fields for updating a component's draft.
type UpdateDraftInput struct {
	Content string
	Format  ContentFormat
}

// UpdateDraftResult is returned after updating a component's draft.
type UpdateDraftResult struct {
	ID string
}

// UpdateDraft updates a component's draft. Component drafts contain
// unpublished edits, which may differ from a recently published version.
// Archived components can't be updated.
func (s *Service) UpdateDraft(ctx context.Context, componentID string, in *UpdateDraftInput) (*UpdateDraftResult, error) {
	if in == nil {
		in = &UpdateDraftInput{}
	}
	req := &api.UpdateComponentDraft{}
	if in.Content != "" {
		req.Content = api.NewOptString(in.Content)
	}
	if in.Format != "" {
		req.Format = api.NewOptComponentContentFormat(api.ComponentContentFormat(in.Format))
	}
	params := api.UpdateComponentDraftParams{ComponentId: componentID}

	res, err := s.api.UpdateComponentDraft(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UpdateComponentDraftResponse:
		return &UpdateDraftResult{ID: r.ID.Or("")}, nil
	case *api.UpdateComponentDraftBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateComponentDraftNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateComponentDraftConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Versions -----------------------------------------------------------

// VersionsInput holds the options for Versions.
type VersionsInput struct {
	// Include lists additional fields to include in the response.
	Include []VersionInclude
}

// VersionsResult is the result of listing a component's published versions.
type VersionsResult struct {
	Versions   []ComponentVersion
	NextCursor string
}

// Versions returns a list of a component's published versions.
func (s *Service) Versions(ctx context.Context, componentID string, in *VersionsInput) (*VersionsResult, error) {
	if in == nil {
		in = &VersionsInput{}
	}
	params := api.GetComponentVersionsParams{ComponentId: componentID}
	if len(in.Include) > 0 {
		params.Include = api.NewOptString(joinVersionIncludes(in.Include))
	}

	res, err := s.api.GetComponentVersions(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetComponentVersions:
		out := &VersionsResult{}
		for _, v := range r.Data {
			out.Versions = append(out.Versions, componentVersionFromAPI(v))
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetComponentVersionsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetComponentVersionsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CreateVersion --------------------------------------------------------

// CreateVersionInput holds the fields for publishing a component version.
type CreateVersionInput struct {
	// Label is the version's label. Must begin and end with an alphanumeric
	// character; may contain letters, digits, dots, underscores, plus signs,
	// and hyphens, up to 60 characters.
	Label string
	// Source is the source to publish the version from. Defaults to the
	// component's current draft.
	Source SourceType
}

// CreateVersionResult is returned after publishing a component version.
type CreateVersionResult struct {
	ID string
}

// CreateVersion publishes a new version of a component from the current
// draft. Archived components must be reactivated before publishing.
func (s *Service) CreateVersion(ctx context.Context, componentID string, in *CreateVersionInput) (*CreateVersionResult, error) {
	if in == nil {
		in = &CreateVersionInput{}
	}
	labelJSON, err := json.Marshal(in.Label)
	if err != nil {
		return nil, err
	}
	req := &api.CreateComponentVersion{Label: labelJSON}
	if in.Source != "" {
		req.Source = api.NewOptCreateComponentVersionSource(api.CreateComponentVersionSource{
			Type: api.NewOptSourceType(api.SourceType(in.Source)),
		})
	}
	params := api.CreateComponentVersionParams{ComponentId: componentID}

	res, err := s.api.CreateComponentVersion(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateComponentVersionResponse:
		return &CreateVersionResult{ID: r.ID.Or("")}, nil
	case *api.CreateComponentVersionBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateComponentVersionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.CreateComponentVersionConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Version ------------------------------------------------------------

// VersionInput holds the options for Version.
type VersionInput struct {
	// Include lists additional fields to include in the response.
	Include []VersionInclude
}

// Version returns a single published version of a component.
func (s *Service) Version(ctx context.Context, componentID, versionID string, in *VersionInput) (*ComponentVersion, error) {
	if in == nil {
		in = &VersionInput{}
	}
	params := api.GetComponentVersionParams{ComponentId: componentID, VersionId: versionID}
	if len(in.Include) > 0 {
		params.Include = api.NewOptString(joinVersionIncludes(in.Include))
	}

	res, err := s.api.GetComponentVersion(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ComponentVersionData:
		v := componentVersionFromAPI(*r)
		return &v, nil
	case *api.GetComponentVersionBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetComponentVersionNotFound:
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
