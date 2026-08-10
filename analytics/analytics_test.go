package analytics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grokify/postman-go/analytics"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

func newService(t *testing.T, handler http.HandlerFunc) (*analytics.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return analytics.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestData(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/analytics" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if got := q.Get("resource"); got != "workspace" {
			t.Errorf("resource = %q, want workspace", got)
		}
		if got := q.Get("metrics"); got != "active_workspaces" {
			t.Errorf("metrics = %q, want active_workspaces", got)
		}
		if got := q.Get("view"); got != "summary" {
			t.Errorf("view = %q, want summary", got)
		}
		if got := q.Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"count": 3, "trend": "up"}}`))
	})
	defer srv.Close()

	got, err := svc.Data(context.Background(), &analytics.DataInput{
		Resource: analytics.ResourceWorkspace,
		Metrics:  analytics.MetricActiveWorkspaces,
		View:     analytics.ViewSummary,
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("Data: %v", err)
	}
	if string(got.Data) != `{"count": 3, "trend": "up"}` {
		t.Errorf("Data = %s", got.Data)
	}
}

func TestDataError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"forbidden", http.StatusForbidden},
		{"internal error", http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"status": ` + strconv.Itoa(tt.statusCode) + `, "title": "err"}`))
			})
			defer srv.Close()

			_, err := svc.Data(context.Background(), &analytics.DataInput{
				Resource: analytics.ResourceWorkspace,
				Metrics:  analytics.MetricActiveWorkspaces,
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var apiErr *postmanerr.APIError
			if !asAPIError(err, &apiErr) {
				t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/analytics-metadata" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("resources"); got != "workspace" {
			t.Errorf("resources = %q, want workspace", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"description": "catalog",
				"resources": [{"name": "workspace"}, {"name": "team"}]
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Metadata(context.Background(), &analytics.MetadataInput{Resources: "workspace"})
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if got.Description != "catalog" {
		t.Errorf("Description = %q, want catalog", got.Description)
	}
	if len(got.Resources) != 2 {
		t.Fatalf("len(Resources) = %d, want 2", len(got.Resources))
	}
	if string(got.Resources[0]) != `{"name": "workspace"}` {
		t.Errorf("Resources[0] = %s", got.Resources[0])
	}
}

func TestMetadataUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.Metadata(context.Background(), nil)
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
