package apisecurity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/apisecurity"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns
// an API Security service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*apisecurity.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return apisecurity.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestValidate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/security/api-validation" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		var body api.SchemaValidationRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		schema, ok := body.Schema.Get()
		if !ok {
			t.Fatal("schema not set")
		}
		if schema.Type != api.SchemaTypeOpenapi3 {
			t.Errorf("type = %q, want openapi3", schema.Type)
		}
		if schema.Language != api.SchemaLanguageJSON {
			t.Errorf("language = %q, want json", schema.Language)
		}
		if schema.Schema != `{"openapi":"3.0.0"}` {
			t.Errorf("schema = %q", schema.Schema)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"warnings": [
				{"severity": "error", "category": "security", "possibleFixUrl": "https://example.com"}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.Validate(context.Background(), &apisecurity.ValidateInput{
		Type:     apisecurity.SchemaTypeOpenAPI3,
		Language: apisecurity.SchemaLanguageJSON,
		Schema:   `{"openapi":"3.0.0"}`,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(got.Warnings))
	}
	var w struct {
		Severity       string `json:"severity"`
		Category       string `json:"category"`
		PossibleFixURL string `json:"possibleFixUrl"`
	}
	if err := json.Unmarshal(got.Warnings[0], &w); err != nil {
		t.Fatalf("unmarshal warning: %v", err)
	}
	if w.Severity != "error" || w.Category != "security" || w.PossibleFixURL != "https://example.com" {
		t.Errorf("warning mismatch: %+v", w)
	}
}

func TestValidateNilInput(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"warnings": []}`))
	})
	defer srv.Close()

	got, err := svc.Validate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(got.Warnings) != 0 {
		t.Errorf("len(Warnings) = %d, want 0", len(got.Warnings))
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantTitle  string
		wantDetail string
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error": {"name": "invalidSchemaError"}}`,
			wantTitle:  http.StatusText(http.StatusBadRequest),
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{}`,
			wantTitle:  http.StatusText(http.StatusUnauthorized),
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       `{}`,
			wantTitle:  http.StatusText(http.StatusForbidden),
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			body:       `{}`,
			wantTitle:  http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})
			defer srv.Close()

			_, err := svc.Validate(context.Background(), &apisecurity.ValidateInput{Schema: "x"})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			apiErr, ok := err.(*postmanerr.APIError)
			if !ok {
				t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if tt.wantTitle != "" && apiErr.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", apiErr.Title, tt.wantTitle)
			}
		})
	}
}
