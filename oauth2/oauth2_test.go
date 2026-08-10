package oauth2_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/oauth2"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns
// an OAuth 2.0 service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*oauth2.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return oauth2.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestGenerate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.GenerateOauthToken
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.GrantType != "client_credentials" {
			t.Errorf("grantType = %q, want client_credentials", body.GrantType)
		}
		if body.InstallationAuthId != "inst1" || body.Jwt != "jwt1" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken": "tok123", "expiresIn": 3600, "tokenType": "Bearer"}`))
	})
	defer srv.Close()

	got, err := svc.Generate(context.Background(), &oauth2.GenerateInput{
		GrantType:          "client_credentials",
		InstallationAuthID: "inst1",
		JWT:                "jwt1",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.AccessToken != "tok123" || got.ExpiresIn != 3600 || got.TokenType != oauth2.TokenTypeBearer {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGenerateBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request", "detail": "missing jwt"}`))
	})
	defer srv.Close()

	_, err := svc.Generate(context.Background(), &oauth2.GenerateInput{GrantType: "client_credentials"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Detail != "missing jwt" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGenerateNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "installation not found"}`))
	})
	defer srv.Close()

	_, err := svc.Generate(context.Background(), &oauth2.GenerateInput{GrantType: "client_credentials"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestGenerateInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Generate(context.Background(), nil)
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

func TestRevoke(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/token/revoke" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.RevokeOauthToken
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Token != "tok123" {
			t.Errorf("token = %q, want tok123", body.Token)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": "true"}`))
	})
	defer srv.Close()

	got, err := svc.Revoke(context.Background(), "tok123")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got.Success != "true" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestRevokeNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "token not found"}`))
	})
	defer srv.Close()

	_, err := svc.Revoke(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*postmanerr.APIError)
	if !ok {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Detail != "token not found" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}
