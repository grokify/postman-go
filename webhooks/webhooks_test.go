package webhooks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/webhooks"
)

func newService(t *testing.T, handler http.HandlerFunc) (*webhooks.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return webhooks.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/webhooks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "ws1" {
			t.Errorf("workspace = %q, want ws1", got)
		}
		var body api.CreateWebhook
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		wh, ok := body.Webhook.Get()
		if !ok {
			t.Fatal("webhook not set in request body")
		}
		if wh.Name != "my hook" || wh.Collection != "col1" {
			t.Errorf("webhook body mismatch: %+v", wh)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"webhook": {
				"id": "wh1",
				"name": "my hook",
				"collection": "col1",
				"webhookUrl": "https://postman.com/webhooks/wh1",
				"uid": "user-wh1"
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &webhooks.CreateInput{
		Workspace:  "ws1",
		Name:       "my hook",
		Collection: "col1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "wh1" || got.Name != "my hook" || got.Collection != "col1" {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.WebhookURL != "https://postman.com/webhooks/wh1" || got.UID != "user-wh1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"name": "ValidationError", "message": "bad input"}}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &webhooks.CreateInput{Workspace: "ws1", Name: "n", Collection: "c"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestCreateUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &webhooks.CreateInput{Workspace: "ws1", Name: "n", Collection: "c"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
