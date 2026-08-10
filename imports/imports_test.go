package imports_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grokify/postman-go/imports"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns
// an Import service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*imports.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return imports.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestFromOpenAPI(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/import/openapi" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		var body struct {
			Type    string `json:"type"`
			Input   string `json:"input"`
			Options struct {
				FolderStrategy string `json:"folderStrategy"`
			} `json:"options"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Type != "string" {
			t.Errorf("type = %q, want string", body.Type)
		}
		if body.Input != "openapi: 3.0.0" {
			t.Errorf("input = %q, want openapi: 3.0.0", body.Input)
		}
		if body.Options.FolderStrategy != "Tags" {
			t.Errorf("options.folderStrategy = %q, want Tags", body.Options.FolderStrategy)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"collections": [
				{"id": "c1", "name": "My API", "uid": "user-c1"}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.FromOpenAPI(context.Background(), &imports.FromOpenAPIInput{
		WorkspaceID: "w1",
		Type:        imports.InputTypeString,
		Input:       "openapi: 3.0.0",
		Options: &imports.CollectionOptions{
			FolderStrategy: imports.FolderStrategyTags,
		},
	})
	if err != nil {
		t.Fatalf("FromOpenAPI: %v", err)
	}
	if len(got.Collections) != 1 {
		t.Fatalf("len(Collections) = %d, want 1", len(got.Collections))
	}
	c := got.Collections[0]
	if c.ID != "c1" || c.Name != "My API" || c.UID != "user-c1" {
		t.Errorf("collection mismatch: %+v", c)
	}
}

func TestFromOpenAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus int
	}{
		{"BadRequest", http.StatusBadRequest, http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized, http.StatusUnauthorized},
		{"InternalServerError", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"status": ` + strconv.Itoa(tt.statusCode) + `}`))
			})
			defer srv.Close()

			_, err := svc.FromOpenAPI(context.Background(), &imports.FromOpenAPIInput{
				WorkspaceID: "w1",
				Type:        imports.InputTypeJSON,
				Input:       map[string]any{"openapi": "3.0.0"},
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var apiErr *postmanerr.APIError
			if !asAPIError(err, &apiErr) {
				t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
			}
			if apiErr.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantStatus)
			}
		})
	}
}

// asAPIError is a tiny errors.As helper kept local to avoid an extra import
// in each test.
func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
