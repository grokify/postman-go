// Package search provides a high-level client for Postman's Search API.
//
// Search lets you find Postman resources (workspaces, collections, requests,
// environments, flows, and specs) by query text and structured filters.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	result, _ := client.Search().Query(ctx, &search.QueryInput{
//		Q:           "auth",
//		ElementType: search.ElementTypeCollections,
//	})
package search

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// ElementType is the type of Postman resource to search for.
type ElementType string

// Element type values.
const (
	ElementTypeRequests     ElementType = "requests"
	ElementTypeCollections  ElementType = "collections"
	ElementTypeWorkspaces   ElementType = "workspaces"
	ElementTypeEnvironments ElementType = "environments"
	ElementTypeFlows        ElementType = "flows"
	ElementTypeSpecs        ElementType = "specs"
)

// Ownership is the ownership scope for search results.
type Ownership string

// Ownership values.
const (
	// OwnershipOrganization returns resources owned by the user's team. This is the default.
	OwnershipOrganization Ownership = "organization"
	// OwnershipExternal returns resources not owned by the user's team.
	OwnershipExternal Ownership = "external"
	// OwnershipAll returns all resources regardless of ownership.
	OwnershipAll Ownership = "all"
)

// Visibility is a resource visibility value, used with FilterFieldVisibility.
type Visibility string

// Visibility values.
const (
	VisibilityInternal Visibility = "internal"
	VisibilityPublic   Visibility = "public"
	VisibilityPartner  Visibility = "partner"
)

// FilterField identifies which resource attribute a Filter condition targets.
type FilterField string

// Filter field values. Fields marked (bool) accept only BoolEq/BoolNe on
// Filter; all others accept Eq/Ne/In/Nin.
const (
	FilterFieldPrivateNetwork      FilterField = "privateNetwork"      // bool
	FilterFieldPublisherIsVerified FilterField = "publisherIsVerified" // bool
	FilterFieldVisibility          FilterField = "visibility"          // Eq/Ne only, value is a Visibility
	FilterFieldWorkspaceID         FilterField = "workspaceId"
	FilterFieldCollectionID        FilterField = "collectionId"
	FilterFieldTags                FilterField = "tags"
	FilterFieldMethod              FilterField = "method"
	FilterFieldRequestID           FilterField = "requestId"
	FilterFieldSpecificationID     FilterField = "specificationId"
	FilterFieldFlowID              FilterField = "flowId"
	FilterFieldEnvironmentID       FilterField = "environmentId"
	FilterFieldCreatedBy           FilterField = "createdBy"
	FilterFieldOrganizationID      FilterField = "organizationId"
	FilterFieldTeamID              FilterField = "teamId"
	FilterFieldIsGitConnected      FilterField = "isGitConnected" // bool
	FilterFieldType                FilterField = "type"
)

// Filter is a single search filter condition. Set Field to select the
// attribute to filter on, then set the comparison(s) that apply:
//
//   - FilterFieldPrivateNetwork, FilterFieldPublisherIsVerified, and
//     FilterFieldIsGitConnected are boolean-valued: use BoolEq/BoolNe.
//   - FilterFieldVisibility accepts only Eq/Ne, set to a Visibility value.
//   - All other fields are string-valued: use Eq/Ne/In/Nin.
type Filter struct {
	Field FilterField

	Eq  string
	Ne  string
	In  []string
	Nin []string

	BoolEq *bool
	BoolNe *bool
}

// Service is the high-level Search client. Obtain one via postman.Client.Search.
type Service struct {
	api *api.Client
}

// New creates a Search service over the given generated API client. Most
// callers should use postman.Client.Search instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Query --------------------------------------------------------------

// QueryInput holds the query text, filters, and pagination options for Query.
type QueryInput struct {
	// Q is the search query text. Case-insensitive.
	Q string
	// ElementType is the type of Postman resource to search for. Required.
	ElementType ElementType
	// Ownership is the ownership scope for results. Defaults to OwnershipOrganization.
	Ownership Ownership
	// Filters narrows results; all filters must match (logical AND).
	Filters []Filter

	// Limit is the maximum number of results to return per page.
	Limit int
	// Cursor is the pagination cursor (use QueryResult.NextCursor to page).
	Cursor string
}

// QueryResult is the paginated result of a resource search.
type QueryResult struct {
	// Results is the page of matching Postman resources.
	Results []Result
	// NextCursor points to the next page; empty when there are no more results.
	NextCursor string
	// Q is the search query text that produced this result.
	Q string
	// Total is the number of matching records.
	Total int
}

// Result describes a single Postman resource matched by a search query.
type Result struct {
	ID                     string
	Name                   string
	Method                 string
	Type                   string
	Description            string
	Summary                string
	URL                    string
	Tags                   []string
	SpecificationID        string
	SpecificationType      string
	SpecificationName      string
	IsPrivateNetworkEntity bool
	CreatedBy              string
	// Team is the team associated with the resource. Nil for the "user" publisher type.
	Team           *Team
	IsGitConnected bool
	// Collection is set only for requests.
	Collection *Collection
	Workspace  *Workspace
	// Organization is the organization that published the resource. Nil for the "user" publisher type.
	Organization *Organization
	Links        *Links
}

// Team identifies a team associated with a search result.
type Team struct {
	ID   string
	Name string
}

// Collection identifies the collection containing a request search result.
type Collection struct {
	ID   string
	Name string
}

// Workspace identifies the workspace containing a search result.
type Workspace struct {
	ID   string
	Name string
}

// Organization identifies the organization that published a search result.
type Organization struct {
	ID         string
	Name       string
	IsVerified bool
}

// Links holds hypermedia links for a search result.
type Links struct {
	// Web is the URL of the resource in the Postman web app.
	Web string
	// Self is the URL of the resource in the Postman API.
	Self string
}

// Query searches Postman for resources such as workspaces, collections,
// requests, and other resource types. Results can be filtered by ownership,
// tags, and other criteria via Filters.
//
// If called without an API key, the response only returns publicly-available
// resources.
func (s *Service) Query(ctx context.Context, in *QueryInput) (*QueryResult, error) {
	if in == nil {
		in = &QueryInput{}
	}

	req := &api.SearchPostmanResources{
		ElementType: api.SearchPostmanResourcesElementType(in.ElementType),
	}
	if in.Q != "" {
		req.Q = api.NewOptString(in.Q)
	}
	if in.Ownership != "" {
		req.Ownership = api.NewOptOwnership(api.Ownership(in.Ownership))
	}
	if len(in.Filters) > 0 {
		var and []api.SearchFilters
		for _, f := range in.Filters {
			and = append(and, filterToAPI(f))
		}
		req.Filters = api.NewOptFilters(api.Filters{And: and})
	}

	params := api.SearchPostmanResourcesParams{}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}

	res, err := s.api.SearchPostmanResources(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SearchPostmanResourcesResponse:
		out := &QueryResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
			out.Q = meta.Q.Or("")
			out.Total = meta.Total.Or(0)
		}
		for _, d := range r.Data {
			out.Results = append(out.Results, resultFromAPI(d))
		}
		return out, nil
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func filterToAPI(f Filter) api.SearchFilters {
	var out api.SearchFilters
	switch f.Field {
	case FilterFieldPrivateNetwork:
		out.PrivateNetwork = api.NewOptSearchFilterPrivateApiNetwork(api.SearchFilterPrivateApiNetwork{
			Eq: optBool(f.BoolEq),
			Ne: optBool(f.BoolNe),
		})
	case FilterFieldPublisherIsVerified:
		out.PublisherIsVerified = api.NewOptSearchFilterPublisherIsVerified(api.SearchFilterPublisherIsVerified{
			Eq: optBool(f.BoolEq),
			Ne: optBool(f.BoolNe),
		})
	case FilterFieldVisibility:
		v := api.SearchFilterVisibility{}
		if f.Eq != "" {
			v.Eq = api.NewOptEq(api.Eq(f.Eq))
		}
		if f.Ne != "" {
			v.Ne = api.NewOptNe(api.Ne(f.Ne))
		}
		out.Visibility = api.NewOptSearchFilterVisibility(v)
	case FilterFieldWorkspaceID:
		out.WorkspaceId = api.NewOptSearchFilterWorkspaceId(api.SearchFilterWorkspaceId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldCollectionID:
		out.CollectionId = api.NewOptSearchFilterCollectionId(api.SearchFilterCollectionId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldTags:
		out.Tags = api.NewOptSearchFilterTags(api.SearchFilterTags{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldMethod:
		out.Method = api.NewOptSearchFilterRequestHttpMethod(api.SearchFilterRequestHttpMethod{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldRequestID:
		out.RequestId = api.NewOptSearchFilterRequestId(api.SearchFilterRequestId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldSpecificationID:
		out.SpecificationId = api.NewOptSearchFilterSpecId(api.SearchFilterSpecId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldFlowID:
		out.FlowId = api.NewOptSearchFilterFlowId(api.SearchFilterFlowId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldEnvironmentID:
		out.EnvironmentId = api.NewOptSearchFilterEnvironmentId(api.SearchFilterEnvironmentId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldCreatedBy:
		out.CreatedBy = api.NewOptSearchFilterCreatedBy(api.SearchFilterCreatedBy{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldOrganizationID:
		out.OrganizationId = api.NewOptSearchFilterOrgId(api.SearchFilterOrgId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldTeamID:
		out.TeamId = api.NewOptSearchFilterTeamId(api.SearchFilterTeamId{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	case FilterFieldIsGitConnected:
		out.IsGitConnected = api.NewOptSearchFilterGitConnected(api.SearchFilterGitConnected{
			Eq: optBool(f.BoolEq),
			Ne: optBool(f.BoolNe),
		})
	case FilterFieldType:
		out.Type = api.NewOptSearchFilterRequestResourceType(api.SearchFilterRequestResourceType{
			Eq: optString(f.Eq), Ne: optString(f.Ne), In: f.In, Nin: f.Nin,
		})
	}
	return out
}

func optString(v string) api.OptString {
	if v == "" {
		return api.OptString{}
	}
	return api.NewOptString(v)
}

func optBool(v *bool) api.OptBool {
	if v == nil {
		return api.OptBool{}
	}
	return api.NewOptBool(*v)
}

func resultFromAPI(d api.SearchPostmanResourcesResponseData) Result {
	out := Result{
		ID:                     d.ID.Or(""),
		Name:                   d.Name.Or(""),
		Method:                 d.Method.Or(""),
		Type:                   d.Type.Or(""),
		Description:            d.Description.Or(""),
		Summary:                d.Summary.Or(""),
		URL:                    d.URL.Or(""),
		Tags:                   d.Tags,
		SpecificationID:        d.SpecificationId.Or(""),
		SpecificationType:      d.SpecificationType.Or(""),
		SpecificationName:      d.SpecificationName.Or(""),
		IsPrivateNetworkEntity: d.IsPrivateNetworkEntity.Or(false),
		CreatedBy:              d.CreatedBy.Or(""),
		IsGitConnected:         d.IsGitConnected.Or(false),
	}
	if v, ok := d.Team.Get(); ok {
		out.Team = &Team{ID: v.ID.Or(""), Name: v.Name.Or("")}
	}
	if v, ok := d.Collection.Get(); ok {
		out.Collection = &Collection{ID: v.ID.Or(""), Name: v.Name.Or("")}
	}
	if v, ok := d.Workspace.Get(); ok {
		out.Workspace = &Workspace{ID: v.ID.Or(""), Name: v.Name.Or("")}
	}
	if v, ok := d.Organization.Get(); ok {
		out.Organization = &Organization{
			ID:         v.ID.Or(""),
			Name:       v.Name.Or(""),
			IsVerified: v.IsVerified.Or(false),
		}
	}
	if v, ok := d.Links.Get(); ok {
		links := &Links{}
		if web, ok := v.Web.Get(); ok {
			links.Web = web.Href.Or("")
		}
		if self, ok := v.Self.Get(); ok {
			links.Self = self.Href.Or("")
		}
		out.Links = links
	}
	return out
}

// --- error helpers --------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
