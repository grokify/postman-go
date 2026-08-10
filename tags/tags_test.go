package tags_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/tags"
)

// newService spins up a test HTTP server with the given handler and returns a
// Tags service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*tags.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return tags.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestCollection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tags": [{"slug": "foo"}, {"slug": "bar"}]}`))
	})
	defer srv.Close()

	got, err := svc.Collection(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Errorf("tags mismatch: %+v", got)
	}
}

func TestUpdateCollection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateTags
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		var decoded []map[string]string
		if err := json.Unmarshal(body.Tags, &decoded); err != nil {
			t.Fatalf("decode tags: %v", err)
		}
		if len(decoded) != 1 || decoded[0]["slug"] != "my-tag" {
			t.Errorf("tags body mismatch: %+v", decoded)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tags": [{"slug": "my-tag"}]}`))
	})
	defer srv.Close()

	got, err := svc.UpdateCollection(context.Background(), "c1", []string{"my-tag"})
	if err != nil {
		t.Fatalf("UpdateCollection: %v", err)
	}
	if len(got) != 1 || got[0] != "my-tag" {
		t.Errorf("tags mismatch: %+v", got)
	}
}

func TestWorkspace(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tags": [{"slug": "team-x"}]}`))
	})
	defer srv.Close()

	got, err := svc.Workspace(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Workspace: %v", err)
	}
	if len(got) != 1 || got[0] != "team-x" {
		t.Errorf("tags mismatch: %+v", got)
	}
}

func TestUpdateWorkspace(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/workspaces/w1/tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tags": [{"slug": "team-x"}, {"slug": "team-y"}]}`))
	})
	defer srv.Close()

	got, err := svc.UpdateWorkspace(context.Background(), "w1", []string{"team-x", "team-y"})
	if err != nil {
		t.Fatalf("UpdateWorkspace: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("tags mismatch: %+v", got)
	}
}

func TestEntities(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tags/my-tag/entities" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("entityType"); got != "workspace" {
			t.Errorf("entityType = %q, want workspace", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {"entities": [{"entityId": "w1", "entityType": "workspace"}]},
			"meta": {"count": 1, "nextCursor": "next1"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Entities(context.Background(), "my-tag", &tags.EntitiesInput{
		EntityType: tags.EntityTypeWorkspace,
	})
	if err != nil {
		t.Fatalf("Entities: %v", err)
	}
	if got.Count != 1 || got.NextCursor != "next1" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Entities) != 1 || got.Entities[0].ID != "w1" || got.Entities[0].Type != tags.EntityTypeWorkspace {
		t.Errorf("entities mismatch: %+v", got.Entities)
	}
}

func TestAPIErrorProblemDetails(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "type": "about:blank", "instance": "/workspaces/w1/tags"}`))
	})
	defer srv.Close()

	_, err := svc.Workspace(context.Background(), "w1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Title != "Not Found" || apiErr.Instance != "/workspaces/w1/tags" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestAPIErrorUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.Collection(context.Background(), "c1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

func TestUpdateCollectionBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "too many tags"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateCollection(context.Background(), "c1", []string{"a", "b", "c", "d", "e", "f"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "too many tags" {
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
