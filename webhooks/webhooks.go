// Package webhooks provides a high-level client for Postman's Webhooks API.
//
// It creates webhooks that trigger a collection run with a custom payload.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	created, _ := client.Webhooks().Create(ctx, &webhooks.CreateInput{
//		Workspace:  "workspace-id",
//		Name:       "my webhook",
//		Collection: "collection-id",
//	})
package webhooks

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Service is the high-level Webhooks client. Obtain one via
// postman.Client.Webhooks.
type Service struct {
	api *api.Client
}

// New creates a Webhooks service over the given generated API client. Most
// callers should use postman.Client.Webhooks instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Create -------------------------------------------------------------

// CreateInput holds the fields for creating a webhook.
type CreateInput struct {
	// Workspace is the workspace's ID. If empty, the webhook is created in
	// the oldest personal Internal workspace the caller owns.
	Workspace string
	// Name is the webhook's name.
	Name string
	// Collection is the ID of the collection the webhook triggers.
	Collection string
	// Environment is the ID of the environment to run the collection with.
	Environment string
}

// CreateResult is the webhook created by Create.
type CreateResult struct {
	ID         string
	Name       string
	Collection string
	// WebhookURL is the URL used to trigger the webhook.
	WebhookURL string
	UID        string
}

// Create creates a webhook that triggers a collection with a custom
// payload. Use CreateResult.WebhookURL to trigger it.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*CreateResult, error) {
	if in == nil {
		in = &CreateInput{}
	}
	webhook := api.CreateWebhookWebhook{
		Collection: in.Collection,
		Name:       in.Name,
	}
	if in.Environment != "" {
		webhook.Environment = api.NewOptString(in.Environment)
	}
	req := &api.CreateWebhook{
		Webhook: api.NewOptCreateWebhookWebhook(webhook),
	}
	params := api.CreateWebhookParams{Workspace: in.Workspace}

	res, err := s.api.CreateWebhook(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.WebhookCreated:
		out := &CreateResult{}
		if wh, ok := r.Webhook.Get(); ok {
			out.ID = wh.ID.Or("")
			out.Name = wh.Name.Or("")
			out.Collection = wh.Collection.Or("")
			out.WebhookURL = wh.WebhookUrl.Or("")
			out.UID = wh.UID.Or("")
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

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
