package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/search"
)

// newService spins up a test HTTP server with the given handler and returns a
// Search service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*search.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return search.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestQuery(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/search" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		if got := r.URL.Query().Get("cursor"); got != "abc" {
			t.Errorf("cursor = %q, want abc", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		var body api.SearchPostmanResources
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ElementType != api.SearchPostmanResourcesElementTypeCollections {
			t.Errorf("elementType = %q, want collections", body.ElementType)
		}
		if body.Q.Or("") != "auth" {
			t.Errorf("q = %q, want auth", body.Q.Or(""))
		}
		if body.Ownership.Or("") != api.OwnershipExternal {
			t.Errorf("ownership = %q, want external", body.Ownership.Or(""))
		}
		filters, ok := body.Filters.Get()
		if !ok || len(filters.And) != 2 {
			t.Fatalf("filters = %+v, want 2 entries", filters)
		}
		if wsID, ok := filters.And[0].WorkspaceId.Get(); !ok || wsID.Eq.Or("") != "w1" {
			t.Errorf("filters[0].workspaceId.eq = %+v, want w1", filters.And[0].WorkspaceId)
		}
		if gc, ok := filters.And[1].IsGitConnected.Get(); !ok || !gc.Eq.Or(false) {
			t.Errorf("filters[1].isGitConnected.eq = %+v, want true", filters.And[1].IsGitConnected)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta": {"nextCursor": "next1", "q": "auth", "total": 1},
			"data": [
				{
					"id": "c1", "name": "Auth Collection", "type": "collection",
					"description": "desc", "summary": "sum", "tags": ["auth", "public"],
					"isPrivateNetworkEntity": false, "createdBy": "u1",
					"team": {"id": "t1", "name": "Team1"},
					"isGitConnected": true,
					"workspace": {"id": "ws1", "name": "WS1"},
					"organization": {"id": "o1", "name": "Org1", "isVerified": true},
					"links": {"web": {"href": "https://web"}, "self": {"href": "https://self"}}
				}
			]
		}`))
	})
	defer srv.Close()

	trueVal := true
	got, err := svc.Query(context.Background(), &search.QueryInput{
		Q:           "auth",
		ElementType: search.ElementTypeCollections,
		Ownership:   search.OwnershipExternal,
		Limit:       10,
		Cursor:      "abc",
		Filters: []search.Filter{
			{Field: search.FilterFieldWorkspaceID, Eq: "w1"},
			{Field: search.FilterFieldIsGitConnected, BoolEq: &trueVal},
		},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if got.NextCursor != "next1" || got.Q != "auth" || got.Total != 1 {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(got.Results))
	}
	r := got.Results[0]
	if r.ID != "c1" || r.Name != "Auth Collection" || len(r.Tags) != 2 {
		t.Errorf("result mismatch: %+v", r)
	}
	if r.Team == nil || r.Team.ID != "t1" {
		t.Errorf("team mismatch: %+v", r.Team)
	}
	if r.Workspace == nil || r.Workspace.ID != "ws1" {
		t.Errorf("workspace mismatch: %+v", r.Workspace)
	}
	if r.Organization == nil || !r.Organization.IsVerified {
		t.Errorf("organization mismatch: %+v", r.Organization)
	}
	if r.Links == nil || r.Links.Web != "https://web" || r.Links.Self != "https://self" {
		t.Errorf("links mismatch: %+v", r.Links)
	}
	if !r.IsGitConnected {
		t.Errorf("isGitConnected = false, want true")
	}
}

func TestQueryNilInput(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	})
	defer srv.Close()

	got, err := svc.Query(context.Background(), nil)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got.Results) != 0 {
		t.Errorf("Results = %+v, want empty", got.Results)
	}
}

func TestQueryBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "elementType is required", "type": "about:blank"}`))
	})
	defer srv.Close()

	_, err := svc.Query(context.Background(), &search.QueryInput{ElementType: search.ElementTypeRequests})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Title != "Bad Request" || apiErr.Detail != "elementType is required" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestQueryInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Query(context.Background(), &search.QueryInput{ElementType: search.ElementTypeRequests})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
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
