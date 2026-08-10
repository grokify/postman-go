package comments_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grokify/postman-go/comments"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns a
// Comments service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*comments.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return comments.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestResolveThread(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/comments-resolutions/123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.ResolveThread(context.Background(), "123"); err != nil {
		t.Fatalf("ResolveThread: %v", err)
	}
}

func TestResolveThreadErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"Unauthorized", http.StatusUnauthorized},
		{"NotFound", http.StatusNotFound},
		{"InternalServerError", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"status": ` + strconv.Itoa(tt.status) + `, "title": "boom", "type": "about:blank", "detail": "d"}`))
			})
			defer srv.Close()

			err := svc.ResolveThread(context.Background(), "123")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var apiErr *postmanerr.APIError
			if !asAPIError(err, &apiErr) {
				t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Title != "boom" || apiErr.Detail != "d" {
				t.Errorf("api error mismatch: %+v", apiErr)
			}
		})
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
