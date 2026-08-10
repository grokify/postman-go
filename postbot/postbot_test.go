package postbot_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postbot"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns a
// Postbot service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*postbot.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return postbot.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestGenerateTool(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/postbot/generations/tool" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		var body api.GenerateTool
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.RequestId != "req1" || body.CollectionId != "col1" {
			t.Errorf("body ids mismatch: %+v", body)
		}
		if body.Config.Language != api.ConfigLanguagePython {
			t.Errorf("language = %q, want python", body.Config.Language)
		}
		if body.Config.AgentFramework != api.AgentFrameworkOpenai {
			t.Errorf("agentFramework = %q, want openai", body.Config.AgentFramework)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"text": "def tool(): pass"}}`))
	})
	defer srv.Close()

	got, err := svc.GenerateTool(context.Background(), &postbot.GenerateToolInput{
		RequestID:      "req1",
		CollectionID:   "col1",
		Language:       postbot.LanguagePython,
		AgentFramework: postbot.AgentFrameworkOpenAI,
	})
	if err != nil {
		t.Fatalf("GenerateTool: %v", err)
	}
	if got.Text != "def tool(): pass" {
		t.Errorf("Text = %q, want %q", got.Text, "def tool(): pass")
	}
}

func TestGenerateToolBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "type": "about:blank", "detail": "invalid config"}`))
	})
	defer srv.Close()

	_, err := svc.GenerateTool(context.Background(), &postbot.GenerateToolInput{
		RequestID:    "req1",
		CollectionID: "col1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Title != "Bad Request" || apiErr.Detail != "invalid config" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGenerateToolTooManyRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status": 429, "title": "Too Many Requests", "detail": "rate limit exceeded"}`))
	})
	defer srv.Close()

	_, err := svc.GenerateTool(context.Background(), &postbot.GenerateToolInput{
		RequestID:    "req1",
		CollectionID: "col1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 429 || apiErr.Title != "Too Many Requests" || apiErr.Detail != "rate limit exceeded" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// TestGenerateToolInternalServerErrorFallback documents a known limitation:
// Postman's generated 500 schema is a union its TS SDK cannot resolve
// statically, so it collapses to an empty object. No body detail can be
// recovered here, only the HTTP status code.
func TestGenerateToolInternalServerErrorFallback(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status": 500, "title": "Internal Server Error"}`))
	})
	defer srv.Close()

	_, err := svc.GenerateTool(context.Background(), &postbot.GenerateToolInput{
		RequestID:    "req1",
		CollectionID: "col1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 || apiErr.Detail != "" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// asAPIError is a tiny errors.As helper kept local to avoid an extra import in
// each test.
func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
