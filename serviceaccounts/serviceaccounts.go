// Package serviceaccounts provides a high-level client for Postman's
// Service Accounts API.
//
// It exchanges a service account API key for a short-lived access token
// used to authenticate downstream service-to-service requests.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(serviceAccountKey))
//	token, _ := client.ServiceAccounts().GenerateToken(ctx)
package serviceaccounts

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level Service Accounts client. Obtain one via
// postman.Client.ServiceAccounts.
type Service struct {
	api *api.Client
}

// New creates a Service Accounts service over the given generated API
// client. Most callers should use postman.Client.ServiceAccounts instead of
// calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- GenerateToken ------------------------------------------------------

// TokenResult is the access token returned by GenerateToken.
type TokenResult struct {
	// AccessToken is a JWT that encodes the service account's identity and
	// permissions. It is valid for 15 minutes.
	AccessToken string
}

// GenerateToken exchanges the service account API key used to authenticate
// this client for a short-lived access token.
//
// The API key configured on the client must belong to a service account;
// API keys belonging to regular users aren't supported. This endpoint has a
// rate limit of 10 requests per 10 second window per user.
func (s *Service) GenerateToken(ctx context.Context) (*TokenResult, error) {
	res, err := s.api.GenerateServiceAccountToken(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GenerateServiceAccountTokenResponse:
		return &TokenResult{AccessToken: r.AccessToken.Or("")}, nil
	case *api.Common400Error:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
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
