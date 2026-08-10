package components_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/components"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns
// a Components service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*components.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return components.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
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

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/components" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("type"); got != "OAS3" {
			t.Errorf("type = %q, want OAS3", got)
		}
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Errorf("status = %q, want active", got)
		}
		if got := r.URL.Query().Get("include"); got != "hasVersions,latestVersion" {
			t.Errorf("include = %q, want hasVersions,latestVersion", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "c1", "name": "Pet", "type": "OAS3", "status": "active",
					"hasVersions": true, "latestVersion": "v1"
				},
				{
					"id": "c2", "name": "Order", "type": "OAS3", "status": "active",
					"latestVersion": {"id": "v2", "label": "1.0.0", "format": "JSON"}
				}
			],
			"meta": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &components.ListInput{
		Type:    components.ComponentTypeOAS3,
		Status:  components.ComponentStatusActive,
		Include: []components.ComponentInclude{components.ComponentIncludeHasVersions, components.ComponentIncludeLatestVersion},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Components) != 2 {
		t.Fatalf("len(Components) = %d, want 2", len(got.Components))
	}
	c1 := got.Components[0]
	if c1.ID != "c1" || !c1.HasVersions || c1.LatestVersion == nil || c1.LatestVersion.ID != "v1" {
		t.Errorf("Components[0] = %+v", c1)
	}
	c2 := got.Components[1]
	if c2.LatestVersion == nil || c2.LatestVersion.Label != "1.0.0" || c2.LatestVersion.Format != "JSON" {
		t.Errorf("Components[1].LatestVersion = %+v", c2.LatestVersion)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/components" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
			Format  string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "PetSchema" || body.Type != "OAS3" || body.Content != "{}" || body.Format != "JSON" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "c1"}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &components.CreateInput{
		Name:    "PetSchema",
		Type:    components.ComponentTypeOAS3,
		Content: "{}",
		Format:  components.ContentFormatJSON,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "c1" {
		t.Errorf("ID = %q, want c1", got.ID)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/components/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("expand"); got != "latestVersion" {
			t.Errorf("expand = %q, want latestVersion", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "c1", "name": "Pet", "type": "OAS3", "status": "active",
				"createdBy": "1", "updatedBy": "1"
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "c1", &components.GetInput{
		Expand: []components.ComponentExpand{components.ComponentExpandLatestVersion},
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "c1" || got.Name != "Pet" || got.Type != components.ComponentTypeOAS3 || got.Status != components.ComponentStatusActive {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/components/c1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		// Known limitation: the generated client always sends "{}" as the
		// body for this operation, since ogen collapsed the request union
		// schema to an empty object.
		if string(body) != "{}" {
			t.Errorf("body = %q, want {}", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "c1", "name": "Pet", "status": "archived"}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != "c1" || got.Status != components.ComponentStatusArchived {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDraft(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/components/c1/drafts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": "{\"openapi\":\"3.0.0\"}", "format": "JSON"}`))
	})
	defer srv.Close()

	got, err := svc.Draft(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if got.Format != "JSON" || got.Content == "" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateDraft(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/components/c1/drafts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateComponentDraft
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Content.Or("") != "updated" || body.Format.Or("") != api.ComponentContentFormatYAML {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "c1"}`))
	})
	defer srv.Close()

	got, err := svc.UpdateDraft(context.Background(), "c1", &components.UpdateDraftInput{
		Content: "updated",
		Format:  components.ContentFormatYAML,
	})
	if err != nil {
		t.Fatalf("UpdateDraft: %v", err)
	}
	if got.ID != "c1" {
		t.Errorf("ID = %q, want c1", got.ID)
	}
}

func TestVersions(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/components/c1/versions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "content" {
			t.Errorf("include = %q, want content", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": "v1", "label": "1.0.0", "format": "JSON", "content": "{}"}],
			"meta": {"nextCursor": "xyz"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Versions(context.Background(), "c1", &components.VersionsInput{
		Include: []components.VersionInclude{components.VersionIncludeContent},
	})
	if err != nil {
		t.Fatalf("Versions: %v", err)
	}
	if got.NextCursor != "xyz" {
		t.Errorf("NextCursor = %q, want xyz", got.NextCursor)
	}
	if len(got.Versions) != 1 || got.Versions[0].ID != "v1" || got.Versions[0].Content != "{}" {
		t.Errorf("versions mismatch: %+v", got.Versions)
	}
}

func TestCreateVersion(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/components/c1/versions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Label  string `json:"label"`
			Source struct {
				Type string `json:"type"`
			} `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Label != "1.0.0" || body.Source.Type != "draft" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "v1"}`))
	})
	defer srv.Close()

	got, err := svc.CreateVersion(context.Background(), "c1", &components.CreateVersionInput{
		Label:  "1.0.0",
		Source: components.SourceTypeDraft,
	})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if got.ID != "v1" {
		t.Errorf("ID = %q, want v1", got.ID)
	}
}

func TestVersion(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/components/c1/versions/v1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "v1", "label": "1.0.0", "url": "https://x", "format": "JSON", "publishedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.Version(context.Background(), "c1", "v1", nil)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if got.ID != "v1" || got.Label != "1.0.0" || got.URL != "https://x" {
		t.Errorf("result mismatch: %+v", got)
	}
}

// TestErrorsCommon401 exercises the 401 error path (which collapses to an
// empty body per postmanerr.Empty) across every operation.
func TestErrorsCommon401(t *testing.T) {
	respond := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	}

	cases := []struct {
		name string
		call func(svc *components.Service) error
	}{
		{"List", func(svc *components.Service) error {
			_, err := svc.List(context.Background(), nil)
			return err
		}},
		{"Create", func(svc *components.Service) error {
			_, err := svc.Create(context.Background(), &components.CreateInput{Name: "n", Type: components.ComponentTypeOAS3, Content: "{}"})
			return err
		}},
		{"Get", func(svc *components.Service) error {
			_, err := svc.Get(context.Background(), "c1", nil)
			return err
		}},
		{"Update", func(svc *components.Service) error {
			_, err := svc.Update(context.Background(), "c1")
			return err
		}},
		{"Draft", func(svc *components.Service) error {
			_, err := svc.Draft(context.Background(), "c1")
			return err
		}},
		{"UpdateDraft", func(svc *components.Service) error {
			_, err := svc.UpdateDraft(context.Background(), "c1", nil)
			return err
		}},
		{"Versions", func(svc *components.Service) error {
			_, err := svc.Versions(context.Background(), "c1", nil)
			return err
		}},
		{"CreateVersion", func(svc *components.Service) error {
			_, err := svc.CreateVersion(context.Background(), "c1", nil)
			return err
		}},
		{"Version", func(svc *components.Service) error {
			_, err := svc.Version(context.Background(), "c1", "v1", nil)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, srv := newService(t, respond)
			defer srv.Close()

			err := tc.call(svc)
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
		})
	}
}

func TestCreateErrorConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict", "detail": "name already exists"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &components.CreateInput{Name: "dup", Type: components.ComponentTypeOAS3, Content: "{}"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Detail != "name already exists" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGetErrorNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "no such component"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Detail != "no such component" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// TestUpdateDraftErrorForbidden documents a known limitation: for
// UpdateComponentDraft, the generated client binds HTTP 403 to the shared
// Common403Error schema, which the TypeScript SDK can't resolve to concrete
// fields (it collapses to an empty object). No body detail can be recovered
// beyond the HTTP status code.
func TestUpdateDraftErrorForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "detail": "component is archived"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateDraft(context.Background(), "c1", &components.UpdateDraftInput{Content: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Detail != "" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}
