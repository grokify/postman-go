// Package secretscanner provides a high-level client for Postman's Secret
// Scanner API.
//
// The Secret Scanner detects secrets (API keys, tokens, and other credentials)
// that have been exposed in a team's Postman workspaces. Access requires a
// Postman Enterprise plan with the Advanced Security Administration add-on.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	types, _ := client.SecretScanner().SecretTypes(ctx)
//	result, _ := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{Resolved: false})
package secretscanner

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Resolution is the resolution status of a detected secret.
type Resolution string

// Resolution status values.
const (
	// ResolutionActive indicates the secret is active and unresolved.
	ResolutionActive Resolution = "ACTIVE"
	// ResolutionFalsePositive indicates the discovered secret is not an actual secret.
	ResolutionFalsePositive Resolution = "FALSE_POSITIVE"
	// ResolutionRevoked indicates the secret was valid but the key was rotated.
	ResolutionRevoked Resolution = "REVOKED"
	// ResolutionAcceptedRisk indicates the user accepts the risk of the exposed secret.
	ResolutionAcceptedRisk Resolution = "ACCEPTED_RISK"
)

// ResourceType is the type of Postman resource in which a secret was detected.
type ResourceType string

// Resource type values.
const (
	ResourceTypeCollection               ResourceType = "collection"
	ResourceTypeEnvironment              ResourceType = "environment"
	ResourceTypeExtensibleCollection     ResourceType = "extensible-collection"
	ResourceTypeGlobals                  ResourceType = "globals"
	ResourceTypeExample                  ResourceType = "example"
	ResourceTypeRequest                  ResourceType = "request"
	ResourceTypeFolder                   ResourceType = "folder"
	ResourceTypeExtensibleCollectionMeta ResourceType = "extensible-collection-meta"
	ResourceTypeExtensibleRequest        ResourceType = "extensible-request"
	ResourceTypeExtensibleFolder         ResourceType = "extensible-folder"
	ResourceTypeExtensibleExample        ResourceType = "extensible-example"
	ResourceTypeExtensibleMessage        ResourceType = "extensible-message"
)

// SecretTypeOrigin is the origin of a supported secret type.
type SecretTypeOrigin string

// Secret type origin values.
const (
	// SecretTypeOriginDefault is a secret type supported by default in Postman.
	SecretTypeOriginDefault SecretTypeOrigin = "DEFAULT"
	// SecretTypeOriginTeamRegex is a custom regex added by a team Admin.
	SecretTypeOriginTeamRegex SecretTypeOrigin = "TEAM_REGEX" //nolint:gosec // G101: Enum/constant identifier matches the credential-name heuristic, but the value is a public tag, not a secret
)

// Service is the high-level Secret Scanner client. Obtain one via
// postman.Client.SecretScanner.
type Service struct {
	api *api.Client
}

// New creates a Secret Scanner service over the given generated API client.
// Most callers should use postman.Client.SecretScanner instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Query ------------------------------------------------------------------

// QueryInput holds the filters and pagination options for Query.
//
// All fields are optional. Resources and WorkspaceIDs are mutually exclusive:
// set at most one of them.
type QueryInput struct {
	// Resolved, when true, returns only secrets with a resolved status.
	Resolved bool
	// SecretTypes limits results to the given secret type IDs (see SecretTypes).
	SecretTypes []string
	// Statuses limits results to the given resolution statuses.
	Statuses []Resolution
	// Resources limits results to the given resources. Mutually exclusive with WorkspaceIDs.
	Resources []QueryResource
	// WorkspaceIDs limits results to the given workspaces. Mutually exclusive with Resources.
	WorkspaceIDs []string
	// WorkspaceVisibilities limits results to workspaces with the given visibility
	// settings (currently "team" and "public").
	WorkspaceVisibilities []string

	// Limit is the maximum number of rows to return.
	Limit int
	// Cursor is the pagination cursor (use QueryResult.NextCursor to page).
	Cursor string
	// IncludeTotal, when true, includes the total match count in QueryResult.Total.
	IncludeTotal bool
	// Since returns only results created at or after this RFC 3339 time.
	Since string
	// Until returns only results created at or before this RFC 3339 time.
	Until string
}

// QueryResource scopes a query to specific resources of a given type.
type QueryResource struct {
	Type ResourceType
	IDs  []string
}

// QueryResult is the paginated result of a detected-secrets query.
type QueryResult struct {
	// Secrets is the page of detected secrets.
	Secrets []DetectedSecret
	// Limit is the maximum number of records in this page.
	Limit int
	// NextCursor points to the next page; empty when there are no more results.
	NextCursor string
	// Total is the number of matching records; only set when IncludeTotal was requested.
	Total int
}

// DetectedSecret describes a single secret found by the Secret Scanner.
type DetectedSecret struct {
	SecretID            string
	SecretType          string
	SecretHash          string
	ObfuscatedSecret    string
	Resolution          Resolution
	WorkspaceID         string
	WorkspaceVisibility string
	ResourceType        string
	ResourceID          string
	DetectedAt          string
	Occurrences         int
}

// Query returns secrets detected by the Secret Scanner, grouped by workspace or
// resource. A nil or empty input returns all results.
func (s *Service) Query(ctx context.Context, in *QueryInput) (*QueryResult, error) {
	if in == nil {
		in = &QueryInput{}
	}

	req := &api.DetectedSecretsQueryRequest{
		SecretTypes:  in.SecretTypes,
		WorkspaceIds: in.WorkspaceIDs,
	}
	if in.Resolved {
		req.Resolved = api.NewOptBool(true)
	}
	for _, st := range in.Statuses {
		req.Statuses = append(req.Statuses, api.Statuses(st))
	}
	for _, v := range in.WorkspaceVisibilities {
		req.WorkspaceVisibilities = append(req.WorkspaceVisibilities, api.WorkspaceVisibilities(v))
	}
	for _, r := range in.Resources {
		res := api.DetectedSecretsQueryRequestResources{Ids: r.IDs}
		if r.Type != "" {
			res.Type = api.NewOptResourcesType(api.ResourcesType(r.Type))
		}
		req.Resources = append(req.Resources, res)
	}

	params := api.DetectedSecretsQueriesParams{}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.IncludeTotal {
		params.Include = api.NewOptString("meta.total")
	}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}

	res, err := s.api.DetectedSecretsQueries(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SearchDetectedSecretsRequest:
		out := &QueryResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.Limit = meta.Limit.Or(0)
			out.NextCursor = meta.NextCursor.Or("")
			out.Total = meta.Total.Or(0)
		}
		for _, d := range r.Data {
			out.Secrets = append(out.Secrets, detectedSecretFromAPI(d))
		}
		return out, nil
	case *api.DetectedSecretsQuery400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.DetectedSecretsQueriesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.SecretScanner403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.DetectedSecretsQueriesInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func detectedSecretFromAPI(d api.SearchDetectedSecretsRequestData) DetectedSecret {
	return DetectedSecret{
		SecretID:            d.SecretId.Or(""),
		SecretType:          d.SecretType.Or(""),
		SecretHash:          d.SecretHash.Or(""),
		ObfuscatedSecret:    d.ObfuscatedSecret.Or(""),
		Resolution:          Resolution(d.Resolution.Or("")),
		WorkspaceID:         d.WorkspaceId.Or(""),
		WorkspaceVisibility: string(d.WorkspaceVisibility.Or("")),
		ResourceType:        d.ResourceType.Or(""),
		ResourceID:          d.ResourceId.Or(""),
		DetectedAt:          d.DetectedAt.Or(""),
		Occurrences:         d.Occurrences.Or(0),
	}
}

// --- UpdateResolution -------------------------------------------------------

// UpdateResolutionInput holds the fields for updating a secret's resolution.
type UpdateResolutionInput struct {
	// Resolution is the new resolution status. ACTIVE cannot be set.
	Resolution Resolution
	// WorkspaceID is the ID of the workspace that contains the secret.
	WorkspaceID string
}

// UpdateResolutionResult is returned after updating a secret's resolution.
type UpdateResolutionResult struct {
	SecretHash  string
	WorkspaceID string
	Resolution  Resolution
	History     []ResolutionHistoryEntry
}

// ResolutionHistoryEntry is a single change to a secret's resolution status.
type ResolutionHistoryEntry struct {
	Actor      int
	CreatedAt  string
	Resolution Resolution
}

// UpdateResolution updates the resolution status of a detected secret.
func (s *Service) UpdateResolution(ctx context.Context, secretID string, in *UpdateResolutionInput) (*UpdateResolutionResult, error) {
	if in == nil {
		in = &UpdateResolutionInput{}
	}
	req := &api.UpdateSecretResolutionRequest{
		Resolution:  api.UpdateSecretResolutionRequestResolution(in.Resolution),
		WorkspaceId: in.WorkspaceID,
	}
	params := api.UpdateDetectedSecretResolutionsParams{SecretId: secretID}

	res, err := s.api.UpdateDetectedSecretResolutions(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UpdateDetectedSecretResolutionsOkResponse:
		out := &UpdateResolutionResult{
			SecretHash:  r.SecretHash.Or(""),
			WorkspaceID: r.WorkspaceId.Or(""),
			Resolution:  Resolution(r.Resolution.Or("")),
		}
		for _, h := range r.History {
			out.History = append(out.History, ResolutionHistoryEntry{
				Actor:      h.Actor.Or(0),
				CreatedAt:  h.CreatedAt.Or(""),
				Resolution: Resolution(h.Resolution.Or("")),
			})
		}
		return out, nil
	case *api.UpdateDetectedSecretResolutionsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateDetectedSecretResolutionsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.SecretScanner403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateDetectedSecretResolutionsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Locations --------------------------------------------------------------

// LocationsInput holds the options for locating a detected secret.
type LocationsInput struct {
	// WorkspaceID is required: the workspace to search within.
	WorkspaceID string
	// Limit is the maximum number of rows to return.
	Limit int
	// Cursor is the pagination cursor (use LocationsResult.NextCursor to page).
	Cursor string
	// Since returns only results created at or after this RFC 3339 time.
	Since string
	// Until returns only results created at or before this RFC 3339 time.
	Until string
	// ResourceType limits results to the given resource type.
	ResourceType ResourceType
}

// LocationsResult is the paginated set of locations where a secret was found.
type LocationsResult struct {
	Locations []SecretLocation

	// Meta fields.
	SecretHash       string
	SecretType       string
	ObfuscatedSecret string
	Limit            int
	Cursor           string
	NextCursor       string
	Total            int
	ActivityFeed     []ActivityFeedEntry
}

// SecretLocation is a single place where a secret was found.
type SecretLocation struct {
	Location          string
	URL               string
	ResourceID        string
	ResourceType      string
	ParentResourceID  string
	IsResourceDeleted bool
	LeakedBy          int
	Occurrences       int
	DetectedAt        string
}

// ActivityFeedEntry is a change to a secret's resolution status.
type ActivityFeedEntry struct {
	ResolvedAt string
	ResolvedBy int
	Status     Resolution
}

// Locations returns the locations of a detected secret within a workspace.
func (s *Service) Locations(ctx context.Context, secretID string, in *LocationsInput) (*LocationsResult, error) {
	if in == nil {
		in = &LocationsInput{}
	}
	params := api.GetDetectedSecretsLocationsParams{
		SecretId:    secretID,
		WorkspaceId: in.WorkspaceID,
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Since != "" {
		params.Since = api.NewOptString(in.Since)
	}
	if in.Until != "" {
		params.Until = api.NewOptString(in.Until)
	}
	if in.ResourceType != "" {
		params.ResourceType = api.NewOptResourceType(api.ResourceType(in.ResourceType))
	}

	res, err := s.api.GetDetectedSecretsLocations(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetDetectedSecretsLocationsOkResponse:
		out := &LocationsResult{}
		for _, l := range r.Data {
			out.Locations = append(out.Locations, SecretLocation{
				Location:          l.Location.Or(""),
				URL:               l.URL.Or(""),
				ResourceID:        l.ResourceId.Or(""),
				ResourceType:      l.ResourceType.Or(""),
				ParentResourceID:  l.ParentResourceId.Or(""),
				IsResourceDeleted: l.IsResourceDeleted.Or(false),
				LeakedBy:          l.LeakedBy.Or(0),
				Occurrences:       l.Occurrences.Or(0),
				DetectedAt:        l.DetectedAt.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.SecretHash = meta.SecretHash.Or("")
			out.SecretType = meta.SecretType.Or("")
			out.ObfuscatedSecret = meta.ObfuscatedSecret.Or("")
			out.Limit = meta.Limit.Or(0)
			out.Cursor = meta.Cursor.Or("")
			out.NextCursor = meta.NextCursor.Or("")
			out.Total = meta.Total.Or(0)
			for _, a := range meta.ActivityFeed {
				out.ActivityFeed = append(out.ActivityFeed, ActivityFeedEntry{
					ResolvedAt: a.ResolvedAt.Or(""),
					ResolvedBy: a.ResolvedBy.Or(0),
					Status:     Resolution(a.Status.Or("")),
				})
			}
		}
		return out, nil
	case *api.GetDetectedSecretsLocationsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetDetectedSecretsLocationsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.SecretScanner403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetDetectedSecretsLocationsInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- SecretTypes ------------------------------------------------------------

// SecretType describes a secret type supported by the Secret Scanner.
type SecretType struct {
	ID     string
	Name   string
	Origin SecretTypeOrigin
}

// SecretTypesResult is the set of supported secret types.
type SecretTypesResult struct {
	Types []SecretType
	Total int
}

// SecretTypes returns the metadata of the secret types supported by the Secret
// Scanner. Use a type's ID in QueryInput.SecretTypes.
func (s *Service) SecretTypes(ctx context.Context) (*SecretTypesResult, error) {
	res, err := s.api.GetSecretTypes(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSecretTypesOkResponse:
		out := &SecretTypesResult{}
		for _, t := range r.Data {
			out.Types = append(out.Types, SecretType{
				ID:     t.ID.Or(""),
				Name:   t.Name.Or(""),
				Origin: SecretTypeOrigin(t.Type.Or("")),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Total = meta.Total.Or(0)
		}
		return out, nil
	case *api.GetSecretTypesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.SecretScanner403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSecretTypesInternalServerError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
