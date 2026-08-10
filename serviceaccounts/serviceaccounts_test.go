package serviceaccounts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/serviceaccounts"
)

func newService(t *testing.T, handler http.HandlerFunc) (*serviceaccounts.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return serviceaccounts.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestGenerateToken(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/service-account-tokens" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken": "eyJhbGciOiJIUzI1NiJ9.example"}`))
	})
	defer srv.Close()

	got, err := svc.GenerateToken(context.Background())
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if got.AccessToken != "eyJhbGciOiJIUzI1NiJ9.example" {
		t.Errorf("AccessToken = %q", got.AccessToken)
	}
}

func TestGenerateTokenErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"bad request", http.StatusBadRequest, `{}`},
		{"unauthorized", http.StatusUnauthorized, `{"status": 401, "title": "Unauthorized"}`},
		{"too many requests", http.StatusTooManyRequests, `{"status": 429, "title": "Too Many Requests"}`},
		{"internal error", http.StatusInternalServerError, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})
			defer srv.Close()

			_, err := svc.GenerateToken(context.Background())
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

func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
