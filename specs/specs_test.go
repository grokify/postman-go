package specs_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/specs"
)

// newService spins up a test HTTP server with the given handler and returns a
// Specs service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*specs.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return specs.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
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

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"meta": {"nextCursor": "abc"},
			"specs": [{"id": "s1", "name": "My Spec", "type": "OPENAPI:3.0", "createdBy": 1}]
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &specs.ListInput{WorkspaceID: "w1", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Specs) != 1 || got.Specs[0].ID != "s1" || got.Specs[0].Type != specs.SpecTypeOpenAPI30 {
		t.Errorf("Specs mismatch: %+v", got.Specs)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/specs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		var body struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Files []struct {
				Path    string `json:"path"`
				Content string `json:"content"`
				Type    string `json:"type,omitempty"`
			} `json:"files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "My Spec" || body.Type != "OPENAPI:3.0" {
			t.Errorf("body mismatch: %+v", body)
		}
		if len(body.Files) != 1 || body.Files[0].Path != "index.yaml" || body.Files[0].Type != "" {
			t.Errorf("files mismatch: %+v", body.Files)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "s1", "name": "My Spec", "type": "OPENAPI:3.0", "createdBy": 1}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &specs.CreateInput{
		WorkspaceID: "w1",
		Name:        "My Spec",
		Type:        specs.SpecTypeOpenAPI30,
		Files: []specs.SpecFileInput{
			{Path: "index.yaml", Content: "openapi: 3.0.0"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "s1" || got.Name != "My Spec" || got.Type != specs.SpecTypeOpenAPI30 || got.CreatedBy != 1 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "s1", "name": "My Spec", "type": "OPENAPI:3.0", "fileFormat": "yaml"}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "s1" || got.FileFormat != specs.FileFormatYAML {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateProperties(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/specs/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateSpecProperties
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "New Name" {
			t.Errorf("name = %q, want New Name", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "s1", "name": "New Name", "type": "OPENAPI:3.0"}`))
	})
	defer srv.Close()

	got, err := svc.UpdateProperties(context.Background(), "s1", &specs.UpdatePropertiesInput{Name: "New Name"})
	if err != nil {
		t.Fatalf("UpdateProperties: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("Name = %q, want New Name", got.Name)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/specs/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer srv.Close()

	if err := svc.Delete(context.Background(), "s1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDefinition(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/definitions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi": "3.0.0"}`))
	})
	defer srv.Close()

	if err := svc.Definition(context.Background(), "s1"); err != nil {
		t.Fatalf("Definition: %v", err)
	}
}

func TestFiles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/files" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"files": [{"id": "f1", "path": "index.yaml", "type": "ROOT"}],
			"meta": {"nextCursor": null}
		}`))
	})
	defer srv.Close()

	got, err := svc.Files(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(got.Files) != 1 || got.Files[0].ID != "f1" || got.Files[0].Type != specs.FileTypeRoot {
		t.Errorf("files mismatch: %+v", got.Files)
	}
}

func TestCreateFile(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/specs/s1/files" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateSpecFile
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Path != "components/schemas.json" {
			t.Errorf("path = %q", body.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "f2", "path": "components/schemas.json", "type": "DEFAULT"}`))
	})
	defer srv.Close()

	got, err := svc.CreateFile(context.Background(), "s1", &specs.CreateFileInput{
		Path:    "components/schemas.json",
		Content: "{}",
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if got.ID != "f2" || got.Type != specs.FileTypeDefault {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestFile(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/files/index.yaml" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "f1", "path": "index.yaml", "content": "openapi: 3.0.0", "type": "ROOT"}`))
	})
	defer srv.Close()

	got, err := svc.File(context.Background(), "s1", "index.yaml")
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if got.Content != "openapi: 3.0.0" {
		t.Errorf("Content = %q", got.Content)
	}
}

func TestUpdateFile(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/specs/s1/files/index.yaml" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateSpecFile
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got, ok := body.Content.Get(); !ok || got != "openapi: 3.0.1" {
			t.Errorf("content = %+v", body.Content)
		}
		if body.Name.Set {
			t.Errorf("name should not be set: %+v", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "f1", "path": "index.yaml", "type": "ROOT"}`))
	})
	defer srv.Close()

	got, err := svc.UpdateFile(context.Background(), "s1", "index.yaml", &specs.UpdateFileInput{
		Content: "openapi: 3.0.1",
	})
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}
	if got.ID != "f1" {
		t.Errorf("ID = %q", got.ID)
	}
}

func TestDeleteFile(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/specs/s1/files/index.yaml" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer srv.Close()

	if err := svc.DeleteFile(context.Background(), "s1", "index.yaml"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
}

func TestCollections(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/generations/collection" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"collections": [{"id": "c1", "name": "My Collection", "state": "in-sync"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.Collections(context.Background(), "s1", &specs.CollectionsInput{Limit: 5})
	if err != nil {
		t.Fatalf("Collections: %v", err)
	}
	if len(got.Collections) != 1 || got.Collections[0].State != specs.SyncStateInSync {
		t.Errorf("collections mismatch: %+v", got.Collections)
	}
}

func TestGenerateCollection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/specs/s1/generations/collection" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.GenerateCollection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "Generated" {
			t.Errorf("name = %q", body.Name)
		}
		if got := body.Options.FolderStrategy.Or(""); got != api.FolderStrategyTags {
			t.Errorf("folderStrategy = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId": "t1", "url": "https://x/tasks/t1"}`))
	})
	defer srv.Close()

	got, err := svc.GenerateCollection(context.Background(), "s1", &specs.GenerateCollectionInput{
		Name:           "Generated",
		FolderStrategy: specs.FolderStrategyTags,
	})
	if err != nil {
		t.Fatalf("GenerateCollection: %v", err)
	}
	if got.TaskID != "t1" {
		t.Errorf("TaskID = %q", got.TaskID)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestSyncWithCollection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/specs/s1/synchronizations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("collectionUid"); got != "c1" {
			t.Errorf("collectionUid = %q, want c1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId": "t1"}`))
	})
	defer srv.Close()

	got, err := svc.SyncWithCollection(context.Background(), "s1", "c1")
	if err != nil {
		t.Fatalf("SyncWithCollection: %v", err)
	}
	if got.TaskID != "t1" {
		t.Errorf("TaskID = %q", got.TaskID)
	}
}

func TestUpdateSyncOptions(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/specs/s1/collections/c1/sync-options" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.ApiSpecSyncOptions
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		opts, ok := body.SyncOptions.Get()
		if !ok || !opts.SyncExamples.Or(false) {
			t.Errorf("syncOptions = %+v", body.SyncOptions)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"syncOptions": {"syncExamples": true, "deleteOrphanedRequests": false}}`))
	})
	defer srv.Close()

	got, err := svc.UpdateSyncOptions(context.Background(), "s1", "c1", &specs.SyncOptionsInput{SyncExamples: true})
	if err != nil {
		t.Fatalf("UpdateSyncOptions: %v", err)
	}
	if !got.SyncExamples || got.DeleteOrphanedRequests {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCollectionSpecs(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/generations/spec" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"specs": [{"id": "s1", "name": "My Spec", "state": "out-of-sync"}]}`))
	})
	defer srv.Close()

	got, err := svc.CollectionSpecs(context.Background(), "c1")
	if err != nil {
		t.Fatalf("CollectionSpecs: %v", err)
	}
	if len(got.Specs) != 1 || got.Specs[0].State != specs.SyncStateOutOfSync {
		t.Errorf("specs mismatch: %+v", got.Specs)
	}
}

func TestGenerateFromCollection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/generations/spec" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.GenerateSpecFromCollection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "Generated Spec" {
			t.Errorf("name = %q", body.Name)
		}
		if got := body.Type.Or(""); got != api.GenerateSpecFromCollectionTypeOPENAPI30 {
			t.Errorf("type = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId": "t2"}`))
	})
	defer srv.Close()

	got, err := svc.GenerateFromCollection(context.Background(), "c1", &specs.GenerateFromCollectionInput{
		Name: "Generated Spec",
		Type: specs.SpecTypeOpenAPI30,
	})
	if err != nil {
		t.Fatalf("GenerateFromCollection: %v", err)
	}
	if got.TaskID != "t2" {
		t.Errorf("TaskID = %q", got.TaskID)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestSyncCollectionWithSpec(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/synchronizations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("specId"); got != "s1" {
			t.Errorf("specId = %q, want s1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId": "t3"}`))
	})
	defer srv.Close()

	got, err := svc.SyncCollectionWithSpec(context.Background(), "c1", "s1")
	if err != nil {
		t.Fatalf("SyncCollectionWithSpec: %v", err)
	}
	if got.TaskID != "t3" {
		t.Errorf("TaskID = %q", got.TaskID)
	}
}

func TestTaskStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/spec1/tasks/t1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "completed", "id": "spec1"}`))
	})
	defer srv.Close()

	got, err := svc.TaskStatus(context.Background(), specs.TaskElementTypeSpecs, "spec1", "t1")
	if err != nil {
		t.Fatalf("TaskStatus: %v", err)
	}
	var decoded struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(got.Raw, &decoded); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if decoded.Status != "completed" {
		t.Errorf("status = %q, want completed", decoded.Status)
	}
}

func TestVersionTag(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/version-tags/tag1/files" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": "e1", "name": "index.yaml", "type": "FILE", "fileType": "ROOT", "content": "openapi: 3.0.0"}]
		}`))
	})
	defer srv.Close()

	got, err := svc.VersionTag(context.Background(), "s1", "tag1")
	if err != nil {
		t.Fatalf("VersionTag: %v", err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Type != specs.VersionTagEntryTypeFile || got.Entries[0].FileType != specs.FileTypeRoot {
		t.Errorf("entries mismatch: %+v", got.Entries)
	}
}

func TestVersionTags(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/specs/s1/version-tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"id": "tag1", "name": "v1"}],
			"meta": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.VersionTags(context.Background(), "s1", nil)
	if err != nil {
		t.Fatalf("VersionTags: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0].ID != "tag1" || got.NextCursor != "abc" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateVersionTag(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/specs/s1/version-tags" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateSpecVersionTag
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name != "v1" {
			t.Errorf("name = %q, want v1", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "tag1", "name": "v1"}}`))
	})
	defer srv.Close()

	got, err := svc.CreateVersionTag(context.Background(), "s1", &specs.CreateVersionTagInput{Name: "v1"})
	if err != nil {
		t.Fatalf("CreateVersionTag: %v", err)
	}
	if got.ID != "tag1" || got.Name != "v1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

// --- error-path tests ---------------------------------------------------

func TestAPIErrorProblemDetails(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized", "type": "about:blank", "instance": "/specs/s1"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 || apiErr.Title != "Unauthorized" || apiErr.Instance != "/specs/s1" {
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

	_, err := svc.Get(context.Background(), "s1")
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

func TestAPIErrorNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "spec not found"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Detail != "spec not found" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGenerateCollectionLocked(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked)
		_, _ = w.Write([]byte(`{"status": 423, "title": "Locked", "detail": "generation in progress"}`))
	})
	defer srv.Close()

	_, err := svc.GenerateCollection(context.Background(), "s1", &specs.GenerateCollectionInput{Name: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 423 {
		t.Errorf("StatusCode = %d, want 423", apiErr.StatusCode)
	}
}

func TestCreateVersionTagConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict", "detail": "version tag already exists"}`))
	})
	defer srv.Close()

	_, err := svc.CreateVersionTag(context.Background(), "s1", &specs.CreateVersionTagInput{Name: "v1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 409 {
		t.Errorf("StatusCode = %d, want 409", apiErr.StatusCode)
	}
}

func TestInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), &specs.ListInput{WorkspaceID: "w1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}
