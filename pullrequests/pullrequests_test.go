package pullrequests_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/pullrequests"
)

// newService spins up a test HTTP server with the given handler and returns a
// Pull Requests service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*pullrequests.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return pullrequests.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/pull-requests/6" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "6", "title": "My PR", "description": "desc", "status": "open",
			"source": {"id": "s1", "name": "fork", "exists": true},
			"destination": {"id": "d1", "name": "main", "exists": true},
			"merge": {"status": "inactive"},
			"reviewers": [{"id": "u1", "status": "pending"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "6")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "6" || got.Title != "My PR" || got.Status != "open" {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.Source.ID != "s1" || !got.Source.Exists {
		t.Errorf("source mismatch: %+v", got.Source)
	}
	if got.Destination.ID != "d1" {
		t.Errorf("destination mismatch: %+v", got.Destination)
	}
	if got.Merge.Status != pullrequests.MergeStatusInactive {
		t.Errorf("merge mismatch: %+v", got.Merge)
	}
	if len(got.Reviewers) != 1 || got.Reviewers[0].ID != "u1" {
		t.Errorf("reviewers mismatch: %+v", got.Reviewers)
	}
}

func TestGetError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "detail": "no access"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "6")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Title != "Forbidden" || apiErr.Detail != "no access" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGetInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "6")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/pull-requests/6" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdatePullRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Title != "new title" {
			t.Errorf("title = %q, want %q", body.Title, "new title")
		}
		if len(body.Reviewers) != 1 || body.Reviewers[0] != "u1" {
			t.Errorf("reviewers = %v", body.Reviewers)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "6", "title": "new title", "description": "d", "sourceId": "s1",
			"destinationId": "d1", "forkType": "collection", "status": "open",
			"createdAt": "2026-01-01T00:00:00Z", "createdBy": "u2", "updatedAt": "2026-01-02T00:00:00Z"
		}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "6", &pullrequests.UpdateInput{
		Title:     "new title",
		Reviewers: []string{"u1"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != "6" || got.Title != "new title" || got.Status != pullrequests.StatusOpen {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.SourceID != "s1" || got.DestinationID != "d1" {
		t.Errorf("ids mismatch: %+v", got)
	}
}

func TestUpdateConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict", "detail": "already merged"}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "6", &pullrequests.UpdateInput{Title: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 409 || apiErr.Detail != "already merged" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestUpdateForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "6", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestReview(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pull-requests/6/tasks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.ReviewPullRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Action != api.ReviewPullRequestActionApprove {
			t.Errorf("action = %q, want approve", body.Action)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "6", "status": "approved", "updatedAt": "2026-01-02T00:00:00Z",
			"reviewedBy": {"id": 42, "name": "Jane", "username": "jane"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Review(context.Background(), "6", &pullrequests.ReviewInput{
		Action: pullrequests.ReviewActionApprove,
	})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if got.ID != "6" || got.Status != "approved" {
		t.Errorf("result mismatch: %+v", got)
	}
	if got.ReviewedBy.ID != 42 || got.ReviewedBy.Username != "jane" {
		t.Errorf("reviewedBy mismatch: %+v", got.ReviewedBy)
	}
}

func TestReviewBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "invalid action"}`))
	})
	defer srv.Close()

	_, err := svc.Review(context.Background(), "6", &pullrequests.ReviewInput{Action: pullrequests.ReviewActionMerge})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "invalid action" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}
