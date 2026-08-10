// Package apisecurity provides a high-level client for Postman's API
// Security service.
//
// It analyzes an API definition against your team's configured API
// governance rulesets (including Postman's OWASP security rules, if
// enabled) and reports any violations found.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	result, _ := client.APISecurity().Validate(ctx, &apisecurity.ValidateInput{
//		Type:     apisecurity.SchemaTypeOpenAPI3,
//		Language: apisecurity.SchemaLanguageJSON,
//		Schema:   openapiJSON,
//	})
package apisecurity

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// SchemaType is the format of an API definition submitted for validation.
type SchemaType string

// Schema type values.
const (
	// SchemaTypeOpenAPI3 indicates an OpenAPI 3.x definition.
	SchemaTypeOpenAPI3 SchemaType = "openapi3"
	// SchemaTypeOpenAPI2 indicates an OpenAPI 2.0 (Swagger) definition.
	SchemaTypeOpenAPI2 SchemaType = "openapi2"
)

// SchemaLanguage is the serialization format of an API definition submitted
// for validation.
type SchemaLanguage string

// Schema language values.
const (
	SchemaLanguageJSON SchemaLanguage = "json"
	SchemaLanguageYaml SchemaLanguage = "yaml"
)

// Service is the high-level API Security client. Obtain one via
// postman.Client.APISecurity.
type Service struct {
	api *api.Client
}

// New creates an API Security service over the given generated API client.
// Most callers should use postman.Client.APISecurity instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Validate -----------------------------------------------------------

// ValidateInput holds the API definition to analyze.
//
// All fields are optional; an empty input validates an empty definition.
type ValidateInput struct {
	// Type is the definition format (openapi3 or openapi2).
	Type SchemaType
	// Language is the definition's serialization format (json or yaml).
	Language SchemaLanguage
	// Schema is the stringified API definition. The maximum allowed size is 10 MB.
	Schema string
}

// ValidateResult is the result of an API security validation.
type ValidateResult struct {
	// Warnings holds the raw JSON of each issue discovered in the analysis.
	// Each object includes the violation's severity and category, the
	// location of the issue, data paths, and (when applicable) a
	// possibleFixUrl linking to documentation to resolve the warning. The
	// schema for these objects is not fixed, so callers must unmarshal them
	// as needed.
	Warnings []json.RawMessage
}

// Validate analyzes an API definition and returns any issues found based on
// your team's configured API governance rulesets. You must import and enable
// Postman's OWASP security rules for this to return security rule
// violations. It can be integrated into CI/CD to automate schema validation.
func (s *Service) Validate(ctx context.Context, in *ValidateInput) (*ValidateResult, error) {
	if in == nil {
		in = &ValidateInput{}
	}

	req := &api.SchemaValidationRequestBody{}
	if in.Type != "" || in.Language != "" || in.Schema != "" {
		req.Schema = api.NewOptSchemaValidationRequestBodySchema(api.SchemaValidationRequestBodySchema{
			Type:     api.SchemaType(in.Type),
			Language: api.SchemaLanguage(in.Language),
			Schema:   in.Schema,
		})
	}

	res, err := s.api.SchemaSecurityValidation(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SchemaSecurityValidationOkResponse:
		out := &ValidateResult{}
		for _, w := range r.Warnings {
			out.Warnings = append(out.Warnings, json.RawMessage(w))
		}
		return out, nil
	case *api.SchemaSecurityValidationBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
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

// --- error helpers --------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
