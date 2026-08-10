package users_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/users"
)

// newService spins up a test HTTP server with the given handler and returns a
// users service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*users.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return users.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestMe(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/me" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"user": {
				"id": 1, "username": "u1", "email": "u1@example.com", "fullName": "User One",
				"isPublic": true, "teamId": 9, "roles": ["user"]
			},
			"operations": [
				{"name": "flow_count", "usage": 3, "limit": 10, "overage": 0}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if got.User.ID != 1 || got.User.Username != "u1" || got.User.TeamID != 9 || !got.User.IsPublic {
		t.Errorf("user mismatch: %+v", got.User)
	}
	if len(got.User.Roles) != 1 || got.User.Roles[0] != "user" {
		t.Errorf("roles mismatch: %+v", got.User.Roles)
	}
	if len(got.Operations) != 1 || got.Operations[0].Usage != 3 || got.Operations[0].Limit != 10 {
		t.Errorf("operations mismatch: %+v", got.Operations)
	}
}

func TestMeUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail": "bad key"}`))
	})
	defer srv.Close()

	_, err := svc.Me(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 401 {
		t.Fatalf("api error mismatch: %v", err)
	}
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("groupId"); got != "7" {
			t.Errorf("groupId = %q, want 7", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": 1, "name": "User One", "username": "u1", "email": "u1@example.com", "roles": ["user"], "joinedAt": "2026-01-01T00:00:00Z"}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &users.ListInput{GroupID: 7})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Users) != 1 {
		t.Fatalf("len(Users) = %d, want 1", len(got.Users))
	}
	u := got.Users[0]
	if u.ID != 1 || u.Name != "User One" || u.Email != "u1@example.com" {
		t.Errorf("user mismatch: %+v", u)
	}
}

func TestListForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Fatalf("api error mismatch: %v", err)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 5, "name": "User Five", "username": "u5", "email": "u5@example.com", "roles": ["admin"], "joinedAt": "2026-02-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), 5)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 5 || got.Name != "User Five" || got.Username != "u5" || len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Errorf("user mismatch: %+v", got)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "type": "about:blank", "instance": "/users/5"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Title != "Not Found" || apiErr.Instance != "/users/5" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGetBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 400 {
		t.Fatalf("api error mismatch: %v", err)
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
