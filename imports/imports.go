// Package imports provides a high-level client for Postman's Import API.
//
// It imports an OpenAPI definition into Postman as a new collection.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	result, _ := client.Imports().FromOpenAPI(ctx, &imports.FromOpenAPIInput{
//		WorkspaceID: "ws-id",
//		Type:        imports.InputTypeString,
//		Input:       openAPIYAMLOrJSONString,
//	})
package imports

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// InputType selects the shape of FromOpenAPIInput.Input.
type InputType string

// Input type values.
const (
	// InputTypeJSON indicates Input is a JSON-marshalable OpenAPI definition.
	InputTypeJSON InputType = "json"
	// InputTypeString indicates Input is a stringified OpenAPI definition.
	InputTypeString InputType = "string"
)

// RequestNameSource determines how the generated collection's requests are
// named.
type RequestNameSource string

// Request name source values.
const (
	// RequestNameSourceFallback names a request after the first of summary,
	// operationId, description, or url found in the schema.
	RequestNameSourceFallback RequestNameSource = "Fallback"
	// RequestNameSourceURL names a request after its URL.
	RequestNameSourceURL RequestNameSource = "URL"
)

// IndentCharacter sets the indentation character used in the generated
// collection.
type IndentCharacter string

// Indent character values.
const (
	IndentCharacterTab   IndentCharacter = "Tab"
	IndentCharacterSpace IndentCharacter = "Space"
)

// FolderStrategy controls whether folders in the generated collection are
// created from the specification's paths or tags.
type FolderStrategy string

// Folder strategy values.
const (
	FolderStrategyPaths FolderStrategy = "Paths"
	FolderStrategyTags  FolderStrategy = "Tags"
)

// Service is the high-level Import client. Obtain one via
// postman.Client.Imports.
type Service struct {
	api *api.Client
}

// New creates an Import service over the given generated API client. Most
// callers should use postman.Client.Imports instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- FromOpenAPI --------------------------------------------------------------

// CollectionOptions holds the advanced OpenAPI-to-collection creation
// options. These field names are case-sensitive on the API side. See
// Postman's OpenAPI to Postman Collection Converter OPTIONS documentation
// for details.
type CollectionOptions struct {
	RequestNameSource           RequestNameSource `json:"requestNameSource,omitempty"`
	IndentCharacter             IndentCharacter   `json:"indentCharacter,omitempty"`
	ParametersResolution        string            `json:"parametersResolution,omitempty"`
	FolderStrategy              FolderStrategy    `json:"folderStrategy,omitempty"`
	IncludeAuthInfoInExample    bool              `json:"includeAuthInfoInExample,omitempty"`
	EnableOptionalParameters    bool              `json:"enableOptionalParameters,omitempty"`
	KeepImplicitHeaders         bool              `json:"keepImplicitHeaders,omitempty"`
	IncludeDeprecated           bool              `json:"includeDeprecated,omitempty"`
	AlwaysInheritAuthentication bool              `json:"alwaysInheritAuthentication,omitempty"`
	NestedFolderHierarchy       bool              `json:"nestedFolderHierarchy,omitempty"`
}

// FromOpenAPIInput holds the parameters for FromOpenAPI.
type FromOpenAPIInput struct {
	// WorkspaceID is the workspace to import the collection into. If empty,
	// Postman imports into the oldest personal Internal workspace you own.
	WorkspaceID string
	// Type selects whether Input is a JSON value or a stringified definition.
	Type InputType
	// Input is the OpenAPI definition: a JSON-marshalable value when Type is
	// InputTypeJSON, or the stringified definition when Type is
	// InputTypeString.
	Input any
	// Options holds advanced collection-creation options. Optional.
	Options *CollectionOptions
}

// FromOpenAPIResult is the set of collections created from an OpenAPI
// import.
type FromOpenAPIResult struct {
	Collections []ImportedCollection
}

// ImportedCollection identifies a single collection created by an import.
type ImportedCollection struct {
	ID   string
	Name string
	UID  string
}

// FromOpenAPI imports an OpenAPI definition into Postman as a new
// collection.
//
// This endpoint has a rate limit of 10 requests per 10 seconds. The Postman
// web app does not support the "file" input method type. For an example of
// importing a file, see Postman's public example collection.
func (s *Service) FromOpenAPI(ctx context.Context, in *FromOpenAPIInput) (*FromOpenAPIResult, error) {
	if in == nil {
		in = &FromOpenAPIInput{}
	}

	body := struct {
		Type    InputType          `json:"type"`
		Input   any                `json:"input"`
		Options *CollectionOptions `json:"options,omitempty"`
	}{
		Type:    in.Type,
		Input:   in.Input,
		Options: in.Options,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	params := api.ImportOpenApiDefinitionParams{Workspace: in.WorkspaceID}

	res, err := s.api.ImportOpenApiDefinition(ctx, api.ImportOpenApiDefinition(raw), params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ImportOpenApiDefinitionOkResponse:
		out := &FromOpenAPIResult{}
		for _, c := range r.Collections {
			out.Collections = append(out.Collections, ImportedCollection{
				ID:   c.ID.Or(""),
				Name: c.Name.Or(""),
				UID:  c.UID.Or(""),
			})
		}
		return out, nil
	case *api.ImportOpenApiDefinitionBadRequestResponse:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
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
