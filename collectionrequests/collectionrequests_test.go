package collectionrequests_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/collectionrequests"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns a
// collection responses service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*collectionrequests.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return collectionrequests.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/requests/r1/comments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": 1, "threadId": 1, "status": "active", "createdBy": 42, "createdAt": "2026-01-01T00:00:00Z", "body": "hi"}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "c1", "r1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("len(Comments) = %d, want 1", len(got.Comments))
	}
	c := got.Comments[0]
	if c.ID != 1 || c.CreatedBy != 42 || c.Body != "hi" || c.Status != "active" {
		t.Errorf("comment mismatch: %+v", c)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/requests/r1/comments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CommentCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Body != "hello" {
			t.Errorf("body.Body = %q, want hello", body.Body)
		}
		if body.ThreadId.Or(0) != 5 {
			t.Errorf("body.ThreadId = %v, want 5", body.ThreadId)
		}
		tags, ok := body.Tags.Get()
		if !ok {
			t.Fatal("body.Tags not set")
		}
		un, ok := tags.UserName.Get()
		if !ok || un.ID != "u1" || un.Type != "user" {
			t.Errorf("tags.UserName = %+v, ok=%v", un, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": 2, "threadId": 5, "createdBy": 42, "body": "hello"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), "c1", "r1", &collectionrequests.CreateInput{
		Body:     "hello",
		ThreadID: 5,
		Tag:      &collectionrequests.Tag{Type: "user", ID: "u1"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != 2 || got.ThreadID != 5 || got.CreatedBy != 42 || got.Body != "hello" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/requests/r1/comments/2" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CommentUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Body != "updated" {
			t.Errorf("body.Body = %q, want updated", body.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": 2, "body": "updated"}}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "c1", "r1", 2, &collectionrequests.UpdateInput{Body: "updated"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != 2 || got.Body != "updated" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/requests/r1/comments/2" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.Delete(context.Background(), "c1", "r1", 2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "type": "about:blank", "instance": "/collections/c1/requests/r1/comments"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "c1", "r1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Title != "Not Found" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestCreateUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), "c1", "r1", &collectionrequests.CreateInput{Body: "hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 401 {
		t.Fatalf("api error mismatch: %v", err)
	}
}

func TestUpdateForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "c1", "r1", 2, &collectionrequests.UpdateInput{Body: "hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Fatalf("api error mismatch: %v", err)
	}
}

func TestDeleteInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status": 500, "title": "Internal Server Error"}`))
	})
	defer srv.Close()

	err := svc.Delete(context.Background(), "c1", "r1", 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 500 {
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
