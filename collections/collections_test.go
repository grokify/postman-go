package collections_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/collections"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns a
// Collections service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*collections.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return collections.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
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

// --- List ------------------------------------------------------------------

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		if got := r.URL.Query().Get("offset"); got != "5" {
			t.Errorf("offset = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"collections": [
				{"id": "c1", "name": "C1", "owner": "9", "createdAt": "t1", "updatedAt": "t2", "uid": "9-c1", "isPublic": true,
				 "fork": {"label": "lbl", "createdAt": "tf", "from": "parent1"}}
			],
			"meta": {"total": 1, "offset": 5, "limit": 10}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &collections.ListInput{Workspace: "w1", Limit: 10, Offset: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Total != 1 || got.Offset != 5 || got.Limit != 10 {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Collections) != 1 {
		t.Fatalf("len(Collections) = %d, want 1", len(got.Collections))
	}
	c := got.Collections[0]
	if c.ID != "c1" || c.Name != "C1" || c.Owner != "9" || !c.IsPublic || c.UID != "9-c1" {
		t.Errorf("collection mismatch: %+v", c)
	}
	if c.Fork == nil || c.Fork.Label != "lbl" || c.Fork.From != "parent1" {
		t.Errorf("fork mismatch: %+v", c.Fork)
	}
}

func TestListNilInput(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("workspace") != "" {
			t.Errorf("workspace should be empty, got %q", r.URL.Query().Get("workspace"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collections": []}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Collections) != 0 {
		t.Errorf("Collections = %+v, want empty", got.Collections)
	}
}

func TestListUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

// --- Create ------------------------------------------------------------------

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		var body api.CreateCollection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		c, ok := body.Collection.Get()
		if !ok || c.Info.Name != "My Collection" {
			t.Errorf("collection mismatch: %+v", c)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"id": "c1", "name": "My Collection", "uid": "9-c1"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), "w1", []byte(`{
		"info": {"name": "My Collection", "schema": "https://schema.postman.com/json/collection/v2.1.0/collection.json"},
		"item": []
	}`))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "c1" || got.Name != "My Collection" || got.UID != "9-c1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateInvalidJSON(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called")
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), "w1", []byte(`not json`))
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestCreateBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "invalid name"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), "w1", []byte(`{
		"info": {"name": "x", "schema": "https://schema.postman.com/json/collection/v2.1.0/collection.json"},
		"item": []
	}`))
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "invalid name" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// --- ForkedByUser ------------------------------------------------------------

func TestForkedByUser(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/collection-forks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("cursor"); got != "abc" {
			t.Errorf("cursor = %q, want abc", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		if got := r.URL.Query().Get("direction"); got != "desc" {
			t.Errorf("direction = %q, want desc", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"forkId": "f1", "forkName": "Fork1", "sourceId": "s1", "createdAt": "t1"}],
			"meta": {"total": 1, "inaccessibleFork": 0, "nextCursor": "next1"}
		}`))
	})
	defer srv.Close()

	got, err := svc.ForkedByUser(context.Background(), &collections.ForkedByUserInput{
		Cursor: "abc", Limit: 5, Direction: collections.SortDirectionDesc,
	})
	if err != nil {
		t.Fatalf("ForkedByUser: %v", err)
	}
	if got.Total != 1 || got.NextCursor != "next1" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Forks) != 1 || got.Forks[0].ForkID != "f1" || got.Forks[0].SourceID != "s1" {
		t.Errorf("forks mismatch: %+v", got.Forks)
	}
}

func TestForkedByUserBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.ForkedByUser(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// --- Fork --------------------------------------------------------------------

func TestFork(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/fork/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		var body api.CreateCollectionFork
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Label != "my fork" {
			t.Errorf("label = %q, want my fork", body.Label)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"id": "c2", "name": "Fork", "uid": "9-c2", "fork": {"label": "my fork", "createdAt": "t1", "from": "c1"}}}`))
	})
	defer srv.Close()

	got, err := svc.Fork(context.Background(), "c1", &collections.ForkInput{Workspace: "w1", Label: "my fork"})
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if got.ID != "c2" || got.UID != "9-c2" {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.Fork == nil || got.Fork.Label != "my fork" || got.Fork.From != "c1" {
		t.Errorf("fork mismatch: %+v", got.Fork)
	}
}

func TestForkNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "collection not found"}`))
	})
	defer srv.Close()

	_, err := svc.Fork(context.Background(), "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Detail != "collection not found" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// --- MergeFork -----------------------------------------------------------

func TestMergeFork(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/merge" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.MergeCollectionFork
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Source != "fork1" || body.Destination != "parent1" {
			t.Errorf("body mismatch: %+v", body)
		}
		strategy, ok := body.Strategy.Get()
		if !ok || strategy != api.MergeCollectionForkStrategyDeleteSource {
			t.Errorf("strategy = %+v", body.Strategy)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"id": "parent1", "uid": "9-parent1"}}`))
	})
	defer srv.Close()

	got, err := svc.MergeFork(context.Background(), &collections.MergeForkInput{
		Source: "fork1", Destination: "parent1", Strategy: collections.MergeStrategyDeleteSource,
	})
	if err != nil {
		t.Fatalf("MergeFork: %v", err)
	}
	if got.ID != "parent1" || got.UID != "9-parent1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestMergeForkForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.MergeFork(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

// --- MergeForkAsync / MergeForkAsyncStatus --------------------------------

func TestMergeForkAsync(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collection-merges" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.MergePullCollectionChanges
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Source != "src1" || body.Destination != "dst1" || body.Strategy != api.MergePullCollectionChangesStrategyDefault {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task": {"id": "t1", "status": "in-progress"}}`))
	})
	defer srv.Close()

	got, err := svc.MergeForkAsync(context.Background(), &collections.MergeForkAsyncInput{Source: "src1", Destination: "dst1"})
	if err != nil {
		t.Fatalf("MergeForkAsync: %v", err)
	}
	if got.ID != "t1" || got.Status != "in-progress" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestMergeForkAsyncBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.MergeForkAsync(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestMergeForkAsyncStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collection-merges-tasks/t1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "t1", "status": "failed", "details": {"error": {"message": "conflict"}}}`))
	})
	defer srv.Close()

	got, err := svc.MergeForkAsyncStatus(context.Background(), "t1")
	if err != nil {
		t.Fatalf("MergeForkAsyncStatus: %v", err)
	}
	if got.ID != "t1" || got.Status != collections.AsyncTaskFailed || got.ErrorMessage != "conflict" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestMergeForkAsyncStatusForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.MergeForkAsyncStatus(context.Background(), "t1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

// --- Get ---------------------------------------------------------------

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("access_key"); got != "key1" {
			t.Errorf("access_key = %q, want key1", got)
		}
		if got := r.URL.Query().Get("model"); got != "minimal" {
			t.Errorf("model = %q, want minimal", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"info": {"name": "My Collection"}, "item": []}}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "c1", &collections.GetInput{AccessKey: "key1", Model: collections.CollectionModelMinimal})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var decoded struct {
		Info struct {
			Name string `json:"name"`
		} `json:"info"`
	}
	if err := json.Unmarshal(got.Collection, &decoded); err != nil {
		t.Fatalf("unmarshal Collection: %v", err)
	}
	if decoded.Info.Name != "My Collection" {
		t.Errorf("Info.Name = %q, want My Collection", decoded.Info.Name)
	}
}

func TestGetBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "bad model"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "bad model" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// --- Replace -------------------------------------------------------------

func TestReplace(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.ReplaceCollectionData
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		ws, ok := body.Collection.Get()
		if !ok || ws.Info.Name != "Replaced" {
			t.Errorf("collection mismatch: %+v", ws)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, err := svc.Replace(context.Background(), "c1", []byte(`{
		"info": {"name": "Replaced", "schema": "https://schema.postman.com/json/collection/v2.1.0/collection.json"},
		"item": []
	}`))
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
}

func TestReplaceInvalidJSON(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called")
	})
	defer srv.Close()

	_, err := svc.Replace(context.Background(), "c1", []byte(`not json`))
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestReplaceConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict", "detail": "id mismatch"}`))
	})
	defer srv.Close()

	_, err := svc.Replace(context.Background(), "c1", []byte(`{
		"info": {"name": "x", "schema": "https://schema.postman.com/json/collection/v2.1.0/collection.json"},
		"item": []
	}`))
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 409 || apiErr.Detail != "id mismatch" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestReplaceBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Replace(context.Background(), "c1", []byte(`{
		"info": {"name": "x", "schema": "https://schema.postman.com/json/collection/v2.1.0/collection.json"},
		"item": []
	}`))
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// --- Update ----------------------------------------------------------------

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/collections/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Collection struct {
				Info struct {
					Name string `json:"name"`
				} `json:"info"`
			} `json:"collection"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Collection.Info.Name != "Updated Name" {
			t.Errorf("name = %q, want Updated Name", body.Collection.Info.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"id": "c1", "name": "Updated Name", "description": "d1"}}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "c1", []byte(`{"info": {"name": "Updated Name"}}`))
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != "c1" || got.Name != "Updated Name" || got.Description != "d1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "c1", []byte(`{}`))
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- Delete ------------------------------------------------------------

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"id": "c1", "uid": "9-c1"}}`))
	})
	defer srv.Close()

	got, err := svc.Delete(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.ID != "c1" || got.UID != "9-c1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "no such collection"}`))
	})
	defer srv.Close()

	_, err := svc.Delete(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Detail != "no such collection" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// --- Comments ------------------------------------------------------------

func TestComments(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/comments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": 1, "threadId": 1, "status": "active", "createdBy": 9, "createdAt": "t1", "updatedAt": "t2", "body": "hi"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.Comments(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Body != "hi" || got[0].CreatedBy != 9 {
		t.Errorf("comments mismatch: %+v", got)
	}
}

func TestCommentsUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.Comments(context.Background(), "c1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

// --- CreateComment ---------------------------------------------------------

func TestCreateComment(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/comments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CommentCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Body != "a new comment" {
			t.Errorf("body = %q", body.Body)
		}
		if got, ok := body.ThreadId.Get(); !ok || got != 5 {
			t.Errorf("threadId = %+v", body.ThreadId)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": 2, "threadId": 5, "createdBy": 9, "createdAt": "t1", "updatedAt": "t2", "body": "a new comment"}}`))
	})
	defer srv.Close()

	got, err := svc.CreateComment(context.Background(), "c1", "a new comment", 5)
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if got.ID != 2 || got.ThreadID != 5 || got.Body != "a new comment" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateCommentForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.CreateComment(context.Background(), "c1", "x", 0)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

// --- UpdateComment -----------------------------------------------------------

func TestUpdateComment(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/comments/2" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CommentUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Body != "updated" {
			t.Errorf("body = %q, want updated", body.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": 2, "threadId": 1, "createdBy": 9, "createdAt": "t1", "updatedAt": "t2", "body": "updated"}}`))
	})
	defer srv.Close()

	got, err := svc.UpdateComment(context.Background(), "c1", 2, "updated")
	if err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if got.Body != "updated" || got.ID != 2 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateCommentNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateComment(context.Background(), "c1", 2, "x")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- DeleteComment -----------------------------------------------------------

func TestDeleteComment(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/comments/2" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.DeleteComment(context.Background(), "c1", 2); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
}

func TestDeleteCommentInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status": 500, "title": "Internal Server Error"}`))
	})
	defer srv.Close()

	err := svc.DeleteComment(context.Background(), "c1", 2)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

// --- Duplicate / DuplicateStatus -----------------------------------------

func TestDuplicate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/duplicates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.DuplicateCollection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Workspace != "w2" {
			t.Errorf("workspace = %q, want w2", body.Workspace)
		}
		if got, ok := body.Suffix.Get(); !ok || got != " (copy)" {
			t.Errorf("suffix = %+v", body.Suffix)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task": {"id": "t1", "status": "processing"}}`))
	})
	defer srv.Close()

	got, err := svc.Duplicate(context.Background(), "c1", &collections.DuplicateInput{Workspace: "w2", Suffix: " (copy)"})
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if got.ID != "t1" || got.Status != collections.DuplicateTaskProcessing {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDuplicateBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Duplicate(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestDuplicateStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collection-duplicate-tasks/t1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task": {"id": "t1", "status": "failed", "reason": "quota exceeded"}}`))
	})
	defer srv.Close()

	got, err := svc.DuplicateStatus(context.Background(), "t1")
	if err != nil {
		t.Fatalf("DuplicateStatus: %v", err)
	}
	if got.Status != collections.DuplicateTaskFailed || got.Reason != "quota exceeded" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDuplicateStatusNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.DuplicateStatus(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- Forks -----------------------------------------------------------------

func TestForks(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/forks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("direction"); got != "asc" {
			t.Errorf("direction = %q, want asc", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"forkId": "f1", "forkName": "Fork1", "createdAt": "t1", "createdBy": "9"}],
			"meta": {"total": 1, "nextCursor": "next1"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Forks(context.Background(), "c1", &collections.ForksInput{Direction: collections.SortDirectionAsc})
	if err != nil {
		t.Fatalf("Forks: %v", err)
	}
	if got.Total != 1 || got.NextCursor != "next1" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Forks) != 1 || got.Forks[0].ID != "f1" || got.Forks[0].CreatedBy != "9" {
		t.Errorf("forks mismatch: %+v", got.Forks)
	}
}

func TestForksNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Forks(context.Background(), "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- PublishDocs / UnpublishDocs -----------------------------------------

func TestPublishDocs(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/public-documentations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.PublishDocumentation
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got, ok := body.EnvironmentUid.Get(); !ok || got != "e1" {
			t.Errorf("environmentUid = %+v", body.EnvironmentUid)
		}
		if got, ok := body.DocumentationLayout.Get(); !ok || got != api.DocumentationLayoutClassicSingleColumn {
			t.Errorf("documentationLayout = %+v", body.DocumentationLayout)
		}
		if got, ok := body.CustomColor.Highlight.Get(); !ok || got != "#fff" {
			t.Errorf("customColor.highlight = %+v", body.CustomColor.Highlight)
		}
		if len(body.Customization.MetaTags) != 1 || body.Customization.MetaTags[0].Name != "og:title" {
			t.Errorf("metaTags mismatch: %+v", body.Customization.MetaTags)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"published": true, "documentationLayout": "classic-single-column",
			"customization": {"metaTags": [{"name": "og:title", "value": "My Collection"}]},
			"publishDate": "2026-01-01T00:00:00Z", "publisherId": "9", "environmentUid": "e1",
			"customColor": {"highlight": "#fff", "rightSidebar": "#000", "topBar": "#111"},
			"publicUrl": "https://doc", "id": "pd1", "collectionId": "c1"
		}`))
	})
	defer srv.Close()

	got, err := svc.PublishDocs(context.Background(), "c1", &collections.PublishDocsInput{
		EnvironmentID: "e1",
		Layout:        collections.DocLayoutClassicSingleColumn,
		CustomColor:   collections.DocColorSettings{Highlight: "#fff", RightSidebar: "#000", TopBar: "#111"},
		MetaTags:      []collections.DocMetaTag{{Name: "og:title", Value: "My Collection"}},
	})
	if err != nil {
		t.Fatalf("PublishDocs: %v", err)
	}
	if !got.Published || got.Layout != "classic-single-column" || got.PublicURL != "https://doc" || got.ID != "pd1" {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.CustomColor.Highlight != "#fff" {
		t.Errorf("CustomColor mismatch: %+v", got.CustomColor)
	}
	if len(got.MetaTags) != 1 || got.MetaTags[0].Name != "og:title" {
		t.Errorf("MetaTags mismatch: %+v", got.MetaTags)
	}
}

func TestPublishDocsBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.PublishDocs(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestUnpublishDocs(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/public-documentations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.UnpublishDocs(context.Background(), "c1"); err != nil {
		t.Fatalf("UnpublishDocs: %v", err)
	}
}

func TestUnpublishDocsNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	err := svc.UnpublishDocs(context.Background(), "c1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// --- Pull / PullRequests / CreatePullRequest ------------------------------

func TestPull(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/pulls" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"destinationId": "c1", "sourceId": "p1"}}`))
	})
	defer srv.Close()

	got, err := svc.Pull(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if got.DestinationID != "c1" || got.SourceID != "p1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestPullNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Pull(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestPullRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/pull-requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": "pr1", "title": "T1", "description": "D1", "status": "open", "sourceId": "s1",
			 "destinationId": "d1", "href": "https://x", "comment": "c", "createdAt": "t1", "createdBy": "9",
			 "updatedAt": "t2", "updatedBy": "10"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.PullRequests(context.Background(), "c1")
	if err != nil {
		t.Fatalf("PullRequests: %v", err)
	}
	if len(got) != 1 || got[0].ID != "pr1" || got[0].Status != collections.PullRequestStatusOpen || got[0].UpdatedBy != "10" {
		t.Errorf("pull requests mismatch: %+v", got)
	}
}

func TestPullRequestsForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.PullRequests(context.Background(), "c1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestCreatePullRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/pull-requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreatePullRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Title != "My PR" || body.DestinationId != "parent1" || len(body.Reviewers) != 1 || body.Reviewers[0] != "9" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "pr1", "title": "My PR", "status": "open", "sourceId": "c1", "destinationId": "parent1", "createdAt": "t1", "createdBy": "9", "updatedAt": "t1"}`))
	})
	defer srv.Close()

	got, err := svc.CreatePullRequest(context.Background(), "c1", &collections.CreatePullRequestInput{
		Title: "My PR", Reviewers: []string{"9"}, DestinationID: "parent1",
	})
	if err != nil {
		t.Fatalf("CreatePullRequest: %v", err)
	}
	if got.ID != "pr1" || got.Title != "My PR" || got.Status != collections.PullRequestStatusOpen {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreatePullRequestBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.CreatePullRequest(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// --- Roles / UpdateRoles -------------------------------------------------

func TestRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"group": [{"id": 1, "role": "VIEWER"}],
			"team": [{"id": 2, "role": "EDITOR"}],
			"user": [{"id": 3, "role": "VIEWER"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.Roles(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Roles: %v", err)
	}
	if len(got.Groups) != 1 || got.Groups[0].ID != 1 || got.Groups[0].Role != collections.RoleViewer {
		t.Errorf("groups mismatch: %+v", got.Groups)
	}
	if len(got.Teams) != 1 || got.Teams[0].ID != 2 || got.Teams[0].Role != collections.RoleEditor {
		t.Errorf("teams mismatch: %+v", got.Teams)
	}
	if len(got.Users) != 1 || got.Users[0].ID != 3 || got.Users[0].Role != collections.RoleViewer {
		t.Errorf("users mismatch: %+v", got.Users)
	}
}

func TestRolesNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Roles(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestUpdateRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/collections/c1/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateCollectionRoles
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.Roles) != 1 || body.Roles[0].Op != api.RolesOpUpdate || body.Roles[0].Path != api.UpdateCollectionRolesRolesPathSlashUser {
			t.Errorf("roles mismatch: %+v", body.Roles)
		}
		if len(body.Roles[0].Value) != 1 || body.Roles[0].Value[0].ID != 5 || body.Roles[0].Value[0].Role != api.ValueRoleEDITOR {
			t.Errorf("value mismatch: %+v", body.Roles[0].Value)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := svc.UpdateRoles(context.Background(), "c1", []collections.RoleUpdate{
		{Path: collections.RolePathUser, Values: []collections.RoleAssignment{{ID: 5, Role: collections.RoleEditor}}},
	})
	if err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}
}

func TestUpdateRolesBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "invalid role"}`))
	})
	defer srv.Close()

	err := svc.UpdateRoles(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "invalid role" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// --- SourceStatus ----------------------------------------------------------

func TestSourceStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/source-status" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collection": {"collectionUid": {"isSourceAhead": true}}}`))
	})
	defer srv.Close()

	got, err := svc.SourceStatus(context.Background(), "c1")
	if err != nil {
		t.Fatalf("SourceStatus: %v", err)
	}
	if !got.IsSourceAhead {
		t.Errorf("IsSourceAhead = false, want true")
	}
}

func TestSourceStatusBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.SourceStatus(context.Background(), "c1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// --- TransformToOpenAPI ----------------------------------------------------

func TestTransformToOpenAPI(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/transformations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "yaml" {
			t.Errorf("format = %q, want yaml", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output": "openapi: 3.0.0"}`))
	})
	defer srv.Close()

	got, err := svc.TransformToOpenAPI(context.Background(), "c1", collections.TransformFormatYAML)
	if err != nil {
		t.Fatalf("TransformToOpenAPI: %v", err)
	}
	if got != "openapi: 3.0.0" {
		t.Errorf("output = %q, want openapi: 3.0.0", got)
	}
}

func TestTransformToOpenAPIUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.TransformToOpenAPI(context.Background(), "c1", "")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

// --- TransferFolders / TransferRequests / TransferResponses ---------------

func transferInput() *collections.TransferInput {
	return &collections.TransferInput{
		IDs:    []string{"i1", "i2"},
		Mode:   collections.TransferModeCopy,
		Target: collections.TransferTarget{ID: "t1", Model: collections.TransferTargetModelFolder},
		Location: collections.TransferLocation{
			ID: "loc1", Model: "folder", Position: collections.TransferPositionAfter,
		},
	}
}

func assertTransferBody(t *testing.T, r *http.Request) {
	t.Helper()
	var body api.TransferCollectionItems
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Ids) != 2 || body.Ids[0] != "i1" || body.Mode != api.ModeCopy {
		t.Errorf("body mismatch: %+v", body)
	}
	if body.Target.ID != "t1" || body.Target.Model != api.TargetModelFolder {
		t.Errorf("target mismatch: %+v", body.Target)
	}
	if got, ok := body.Location.ID.Get(); !ok || got != "loc1" {
		t.Errorf("location.id = %+v", body.Location.ID)
	}
	if got, ok := body.Location.Model.Get(); !ok || got != "folder" {
		t.Errorf("location.model = %+v", body.Location.Model)
	}
	if body.Location.Position != api.PositionAfter {
		t.Errorf("location.position = %q, want after", body.Location.Position)
	}
}

func TestTransferFolders(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collection-folders-transfers" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertTransferBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ids": ["i1", "i2"]}`))
	})
	defer srv.Close()

	got, err := svc.TransferFolders(context.Background(), transferInput())
	if err != nil {
		t.Fatalf("TransferFolders: %v", err)
	}
	if len(got.IDs) != 2 || got.IDs[0] != "i1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestTransferFoldersBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.TransferFolders(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestTransferRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collection-requests-transfers" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertTransferBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ids": ["i1", "i2"]}`))
	})
	defer srv.Close()

	got, err := svc.TransferRequests(context.Background(), transferInput())
	if err != nil {
		t.Fatalf("TransferRequests: %v", err)
	}
	if len(got.IDs) != 2 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestTransferRequestsBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.TransferRequests(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestTransferResponses(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collection-responses-transfers" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		assertTransferBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ids": ["i1", "i2"]}`))
	})
	defer srv.Close()

	got, err := svc.TransferResponses(context.Background(), transferInput())
	if err != nil {
		t.Fatalf("TransferResponses: %v", err)
	}
	if len(got.IDs) != 2 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestTransferResponsesBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.TransferResponses(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// --- UpdateStatus ------------------------------------------------------

func TestUpdateStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collection-updates-tasks/t1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "t1", "status": "successful"}`))
	})
	defer srv.Close()

	got, err := svc.UpdateStatus(context.Background(), "t1")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if got.ID != "t1" || got.Status != collections.AsyncTaskSuccessful {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateStatusForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.UpdateStatus(context.Background(), "t1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

// --- misc --------------------------------------------------------------

func TestInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}
