// Package oauth2 provides a high-level client for Postman's OAuth 2.0 API.
//
// Use Generate to obtain an access token for a client application via the
// client_credentials grant type, and Revoke to invalidate a previously issued
// token.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	tok, _ := client.OAuth2().Generate(ctx, &oauth2.GenerateInput{
//		GrantType:          "client_credentials",
//		InstallationAuthID: "installation-id",
//		JWT:                "jwt",
//	})
package oauth2

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// TokenType is the type of an issued OAuth 2.0 access token.
type TokenType string

// Token type values.
const (
	TokenTypeBearer TokenType = "Bearer"
)

// Service is the high-level OAuth 2.0 client. Obtain one via
// postman.Client.OAuth2.
type Service struct {
	api *api.Client
}

// New creates an OAuth 2.0 service over the given generated API client. Most
// callers should use postman.Client.OAuth2 instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Generate ----------------------------------------------------------

// GenerateInput holds the fields for generating an OAuth 2.0 access token.
type GenerateInput struct {
	// GrantType is the OAuth 2.0 grant type, e.g. "client_credentials".
	GrantType string
	// InstallationAuthID identifies the app installation to authenticate as.
	InstallationAuthID string
	// JWT is the signed JSON Web Token proving the caller's identity.
	JWT string
}

// GenerateResult is the issued access token.
type GenerateResult struct {
	AccessToken string
	ExpiresIn   int
	TokenType   TokenType
}

// Generate generates an OAuth 2.0 access token for a client application using
// the client_credentials grant type. Use this with backend services or bots
// to authenticate and authorize API requests without user interaction.
func (s *Service) Generate(ctx context.Context, in *GenerateInput) (*GenerateResult, error) {
	if in == nil {
		in = &GenerateInput{}
	}
	req := &api.GenerateOauthToken{
		GrantType:          in.GrantType,
		InstallationAuthId: in.InstallationAuthID,
		Jwt:                in.JWT,
	}

	res, err := s.api.GenerateOauthToken(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GenerateOauthTokenResponse:
		return &GenerateResult{
			AccessToken: r.AccessToken.Or(""),
			ExpiresIn:   r.ExpiresIn.Or(0),
			TokenType:   TokenType(r.TokenType.Or("")),
		}, nil
	case *api.GenerateOauthTokenBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GenerateOauthTokenNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Revoke ------------------------------------------------------------

// RevokeResult reports the outcome of revoking an OAuth 2.0 access token.
type RevokeResult struct {
	Success string
}

// Revoke revokes an active OAuth 2.0 access token, preventing further use of
// it for authentication. Revocation is immediate and can't be undone. This
// request does not use any authorization.
func (s *Service) Revoke(ctx context.Context, token string) (*RevokeResult, error) {
	req := &api.RevokeOauthToken{Token: token}

	res, err := s.api.RevokeOauthToken(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.RevokeOauthTokenResponse:
		return &RevokeResult{Success: r.Success.Or("")}, nil
	case *api.OauthTokenError:
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
