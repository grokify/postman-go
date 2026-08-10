package monitors_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/monitors"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns a
// Monitors service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*monitors.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return monitors.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/monitors" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		if got := r.URL.Query().Get("active"); got != "true" {
			t.Errorf("active = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"monitors": [
				{"id": "m1", "name": "Mon 1", "active": true, "uid": "u1", "owner": 7, "collectionUid": "c1"}
			],
			"meta": {"limit": 25, "nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &monitors.ListInput{WorkspaceID: "w1", Active: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Limit != 25 || got.NextCursor != "abc" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Monitors) != 1 || got.Monitors[0].ID != "m1" || got.Monitors[0].CollectionUID != "c1" {
		t.Errorf("monitors mismatch: %+v", got.Monitors)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monitors" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		var body api.CreateMonitor
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		m, ok := body.Monitor.Get()
		if !ok {
			t.Fatal("monitor not set")
		}
		if m.Name != "My Monitor" || m.Collection != "col1" {
			t.Errorf("monitor mismatch: %+v", m)
		}
		if m.Schedule.Cron.Or("") != "0 0 * * *" {
			t.Errorf("cron = %q", m.Schedule.Cron.Or(""))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor": {"id": "m1", "name": "My Monitor", "active": true, "uid": "u1"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &monitors.CreateInput{
		WorkspaceID: "w1",
		Name:        "My Monitor",
		Collection:  "col1",
		Schedule:    monitors.Schedule{Cron: "0 0 * * *", Timezone: "UTC"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "m1" || got.Name != "My Monitor" || !got.Active {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/monitors/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"monitor": {
				"id": "m1", "name": "Mon 1", "uid": "u1", "owner": 7, "active": true,
				"notificationLimit": 5, "collectionUid": "c1", "environmentUid": "e1",
				"options": {"followRedirects": true, "requestDelay": 100, "requestTimeout": 5000, "strictSsl": true},
				"notifications": {"onError": [{"email": "a@example.com"}], "onFailure": [{"email": "b@example.com"}]},
				"distribution": [{"region": "us-east"}],
				"schedule": {"cron": "0 0 * * *", "timezone": "UTC", "nextRun": "2026-01-01T00:00:00Z"},
				"retry": {"attempts": 2},
				"lastRun": {
					"status": "success", "startedAt": "2026-01-01T00:00:00Z", "finishedAt": "2026-01-01T00:01:00Z",
					"stats": {"assertions": {"total": 10, "failed": 1}, "requests": {"total": 5, "failed": 0}, "runCount": 1, "errorCount": 0}
				}
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "m1" || got.NotificationLimit != 5 || got.CollectionUID != "c1" {
		t.Errorf("monitor mismatch: %+v", got)
	}
	if !got.Options.FollowRedirects || got.Options.RequestDelay != 100 || got.Options.RequestTimeout != 5000 {
		t.Errorf("options mismatch: %+v", got.Options)
	}
	if len(got.Notifications.OnError) != 1 || got.Notifications.OnError[0].Email != "a@example.com" {
		t.Errorf("notifications mismatch: %+v", got.Notifications)
	}
	if len(got.Distribution) != 1 || got.Distribution[0] != monitors.RegionUsEast {
		t.Errorf("distribution mismatch: %+v", got.Distribution)
	}
	if got.Schedule.NextRun != "2026-01-01T00:00:00Z" || got.Retry.Attempts != 2 {
		t.Errorf("schedule/retry mismatch: %+v %+v", got.Schedule, got.Retry)
	}
	if got.LastRun.Stats.AssertionsTotal != 10 || got.LastRun.Stats.RequestsTotal != 5 {
		t.Errorf("last run stats mismatch: %+v", got.LastRun.Stats)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/monitors/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateMonitor
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		m, ok := body.Monitor.Get()
		if !ok {
			t.Fatal("monitor not set")
		}
		if m.Active.Or(true) {
			t.Errorf("active = %v, want false", m.Active.Or(true))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor": {"id": "m1", "name": "Renamed", "active": false, "uid": "u1"}}`))
	})
	defer srv.Close()

	inactive := false
	got, err := svc.Update(context.Background(), "m1", &monitors.UpdateInput{Active: &inactive})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Active {
		t.Errorf("Active = true, want false")
	}
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", got.Name)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/monitors/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monitor": {"id": "m1", "uid": "u1"}}`))
	})
	defer srv.Close()

	got, err := svc.Delete(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.ID != "m1" || got.UID != "u1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestRun(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monitors/m1/run" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("async"); got != "true" {
			t.Errorf("async = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Run(context.Background(), "m1", &monitors.RunInput{Async: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunnerInstances(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/runners/r1/instances" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": "i1", "hostname": "host1", "uniqueId": "u1", "runnerId": "r1"}],
			"meta": {"nextCursor": "next1"}
		}`))
	})
	defer srv.Close()

	got, err := svc.RunnerInstances(context.Background(), "r1", &monitors.RunnerInstancesInput{Limit: 10})
	if err != nil {
		t.Fatalf("RunnerInstances: %v", err)
	}
	if len(got.Instances) != 1 || got.Instances[0].ID != "i1" || got.Instances[0].RunnerID != "r1" {
		t.Errorf("instances mismatch: %+v", got.Instances)
	}
	if got.NextCursor != "next1" {
		t.Errorf("NextCursor = %q, want next1", got.NextCursor)
	}
}

func TestRunnerMetrics(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/runners/r1/metrics" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lastPingAt": "2026-01-01T00:00:00Z", "oldestQueuedRunAgeSeconds": 30, "queueDepth": 2}`))
	})
	defer srv.Close()

	got, err := svc.RunnerMetrics(context.Background(), "r1")
	if err != nil {
		t.Fatalf("RunnerMetrics: %v", err)
	}
	if got.QueueDepth != 2 || got.OldestQueuedRunAgeSeconds != 30 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestAPIErrorProblemDetails(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "type": "about:blank", "instance": "/monitors/m1"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Title != "Not Found" || apiErr.Instance != "/monitors/m1" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// TestAPIErrorForbiddenFallback documents a known limitation: Postman's
// generated 403 schema is a union its TS SDK cannot resolve statically, so it
// collapses to an empty object. No body detail can be recovered here, only
// the HTTP status code.
func TestAPIErrorForbiddenFallback(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "detail": "no access"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "m1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Title != "Forbidden" || apiErr.Detail != "" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestListBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
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

// asAPIError is a tiny errors.As helper kept local to avoid an extra import in
// each test.
func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
