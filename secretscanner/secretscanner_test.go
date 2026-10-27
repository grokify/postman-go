package secretscanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/secretscanner"
)

// newService spins up a test HTTP server with the given handler and returns a
// Secret Scanner service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*secretscanner.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return secretscanner.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestSecretTypes(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/secret-types" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "aws_access_key", "name": "AWS Access Key", "type": "DEFAULT"},
				{"id": "custom_1", "name": "Custom", "type": "TEAM_REGEX"}
			],
			"meta": {"total": 2}
		}`))
	})
	defer srv.Close()

	got, err := svc.SecretTypes(context.Background())
	if err != nil {
		t.Fatalf("SecretTypes: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2", got.Total)
	}
	if len(got.Types) != 2 {
		t.Fatalf("len(Types) = %d, want 2", len(got.Types))
	}
	if got.Types[0].ID != "aws_access_key" || got.Types[0].Origin != secretscanner.SecretTypeOriginDefault {
		t.Errorf("Types[0] = %+v", got.Types[0])
	}
	if got.Types[1].Origin != secretscanner.SecretTypeOriginTeamRegex {
		t.Errorf("Types[1].Origin = %q", got.Types[1].Origin)
	}
}

func TestQuery(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/detected-secrets-queries" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "meta.total" {
			t.Errorf("include = %q, want meta.total", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		var body api.DetectedSecretsQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if !body.Resolved.Or(false) {
			t.Errorf("resolved = false, want true")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta": {"limit": 10, "nextCursor": "abc", "total": 1},
			"data": [
				{"secretId": "s1", "secretType": "aws", "resolution": "ACTIVE", "occurrences": 3, "workspaceId": "w1"}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.Query(context.Background(), &secretscanner.QueryInput{
		Resolved:     true,
		Limit:        10,
		IncludeTotal: true,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if got.Total != 1 || got.NextCursor != "abc" || got.Limit != 10 {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Secrets) != 1 {
		t.Fatalf("len(Secrets) = %d, want 1", len(got.Secrets))
	}
	s := got.Secrets[0]
	if s.SecretID != "s1" || s.Resolution != secretscanner.ResolutionActive || s.Occurrences != 3 {
		t.Errorf("secret mismatch: %+v", s)
	}
}

// TestQueryDefaultLimit verifies the SDK always sends an explicit limit, even
// when the caller specifies none. Omitting it makes Postman echo its default
// page size back as a string ("limit":"30"), which the typed decoder rejects;
// sending an explicit limit forces an integer echo.
func TestQueryDefaultLimit(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100 (default)", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta": {"limit": 100, "nextCursor": null}, "data": []}`))
	})
	defer srv.Close()

	if _, err := svc.Query(context.Background(), &secretscanner.QueryInput{}); err != nil {
		t.Fatalf("Query: %v", err)
	}
}

func TestUpdateResolution(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/detected-secrets/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateSecretResolutionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Resolution != api.UpdateSecretResolutionRequestResolutionREVOKED {
			t.Errorf("resolution = %q, want REVOKED", body.Resolution)
		}
		if body.WorkspaceId != "w1" {
			t.Errorf("workspaceId = %q, want w1", body.WorkspaceId)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"secretHash": "h1", "workspaceId": "w1", "resolution": "REVOKED",
			"history": [{"actor": 42, "createdAt": "2026-01-01T00:00:00Z", "resolution": "REVOKED"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.UpdateResolution(context.Background(), "s1", &secretscanner.UpdateResolutionInput{
		Resolution:  secretscanner.ResolutionRevoked,
		WorkspaceID: "w1",
	})
	if err != nil {
		t.Fatalf("UpdateResolution: %v", err)
	}
	if got.Resolution != secretscanner.ResolutionRevoked || got.SecretHash != "h1" {
		t.Errorf("result mismatch: %+v", got)
	}
	if len(got.History) != 1 || got.History[0].Actor != 42 {
		t.Errorf("history mismatch: %+v", got.History)
	}
}

func TestLocations(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/detected-secrets/s1/locations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"location": "loc1", "url": "https://x", "resourceId": "r1", "occurrences": 2, "isResourceDeleted": true}
			],
			"meta": {"secretHash": "h1", "total": 1, "activityFeed": [{"resolvedBy": 7, "status": "ACTIVE"}]}
		}`))
	})
	defer srv.Close()

	got, err := svc.Locations(context.Background(), "s1", &secretscanner.LocationsInput{WorkspaceID: "w1"})
	if err != nil {
		t.Fatalf("Locations: %v", err)
	}
	if got.SecretHash != "h1" || got.Total != 1 {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Locations) != 1 || got.Locations[0].Location != "loc1" || !got.Locations[0].IsResourceDeleted {
		t.Errorf("location mismatch: %+v", got.Locations)
	}
	if len(got.ActivityFeed) != 1 || got.ActivityFeed[0].ResolvedBy != 7 {
		t.Errorf("activity feed mismatch: %+v", got.ActivityFeed)
	}
}

func TestAPIErrorProblemDetails(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized", "type": "about:blank", "instance": "/secret-types"}`))
	})
	defer srv.Close()

	_, err := svc.SecretTypes(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 || apiErr.Title != "Unauthorized" || apiErr.Instance != "/secret-types" {
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

	_, err := svc.SecretTypes(context.Background())
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

// asAPIError is a tiny errors.As helper kept local to avoid an extra import in
// each test.
func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}

// TestLocationsTolerateNulls reproduces the live Secret Scanner locations
// response for an unresolved secret: meta.cursor is null, and the activity-feed
// entry's resolvedAt/resolvedBy are null. These fields were originally typed
// non-nullable and the decoder rejected the nulls; the spec now marks them
// nullable. This guards against a regression on the next spec regeneration.
func TestLocationsTolerateNulls(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"resourceId": "r1", "resourceType": "request", "location": "Description",
				 "occurrences": 1, "leakedBy": 42, "isResourceDeleted": false,
				 "parentResourceId": "p1", "url": "https://go.postman.co/x"}
			],
			"meta": {
				"secretHash": "h1", "secretType": "GitHub Personal Access Token",
				"obfuscatedSecret": "ghp_1******", "limit": 100, "cursor": null, "nextCursor": null,
				"activityFeed": [{"status": "ACTIVE", "resolvedAt": null, "resolvedBy": null}]
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Locations(context.Background(), "s1", &secretscanner.LocationsInput{WorkspaceID: "w1"})
	if err != nil {
		t.Fatalf("Locations decode with null cursor/resolvedAt/resolvedBy failed: %v", err)
	}
	if len(got.Locations) != 1 || got.Locations[0].LeakedBy != 42 || got.Locations[0].URL != "https://go.postman.co/x" {
		t.Errorf("location mismatch: %+v", got.Locations)
	}
	if len(got.ActivityFeed) != 1 || got.ActivityFeed[0].Status != secretscanner.ResolutionActive {
		t.Errorf("activity feed mismatch: %+v", got.ActivityFeed)
	}
	// Null resolvedAt/resolvedBy decode to zero values.
	if got.ActivityFeed[0].ResolvedAt != "" || got.ActivityFeed[0].ResolvedBy != 0 {
		t.Errorf("expected zero-valued resolvedAt/resolvedBy for null, got %+v", got.ActivityFeed[0])
	}
}
