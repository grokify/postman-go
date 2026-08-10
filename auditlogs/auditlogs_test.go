package auditlogs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/auditlogs"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

func newService(t *testing.T, handler http.HandlerFunc) (*auditlogs.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return auditlogs.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/audit/logs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("userId"); got != "9" {
			t.Errorf("userId = %q, want 9", got)
		}
		if got := r.URL.Query().Get("orderBy"); got != "desc" {
			t.Errorf("orderBy = %q, want desc", got)
		}
		// order_by is a documented collision with orderBy dropped by the
		// generator; it must never be sent.
		if r.URL.Query().Has("order_by") {
			t.Errorf("unexpected order_by query param present")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"trails": [
				{
					"id": 1,
					"ip": "1.2.3.4",
					"action": "user.login",
					"timestamp": "2026-01-01T00:00:00Z",
					"data": {
						"actor": {"name": "Alice", "id": 9, "active": true},
						"user": {"name": "Bob", "id": 10},
						"team": {"name": "Team A", "id": 5}
					}
				}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &auditlogs.ListInput{
		UserID:  9,
		OrderBy: auditlogs.OrderDesc,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Trails) != 1 {
		t.Fatalf("len(Trails) = %d, want 1", len(got.Trails))
	}
	ev := got.Trails[0]
	if ev.ID != 1 || ev.IP != "1.2.3.4" || ev.Action != "user.login" {
		t.Errorf("event mismatch: %+v", ev)
	}
	if ev.Data.Actor.Name != "Alice" || ev.Data.Actor.ID != 9 || !ev.Data.Actor.Active {
		t.Errorf("actor mismatch: %+v", ev.Data.Actor)
	}
	if ev.Data.User.Name != "Bob" || ev.Data.User.ID != 10 {
		t.Errorf("user mismatch: %+v", ev.Data.User)
	}
	if ev.Data.Team.Name != "Team A" || ev.Data.Team.ID != 5 {
		t.Errorf("team mismatch: %+v", ev.Data.Team)
	}
}

func TestListError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"name": "Bad", "message": "bad request"}}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
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

func TestActions(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/audit-actions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name": "user.login", "displayName": "User Login"},
			{"name": "workspace.created", "displayName": "Workspace Created"}
		]`))
	})
	defer srv.Close()

	got, err := svc.Actions(context.Background())
	if err != nil {
		t.Fatalf("Actions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Actions) = %d, want 2", len(got))
	}
	if got[0].Name != "user.login" || got[0].DisplayName != "User Login" {
		t.Errorf("action[0] mismatch: %+v", got[0])
	}
}

func TestActionsForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"name": "Forbidden", "message": "no access"}}`))
	})
	defer srv.Close()

	_, err := svc.Actions(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
