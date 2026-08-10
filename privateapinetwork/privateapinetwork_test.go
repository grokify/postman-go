package privateapinetwork_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/privateapinetwork"
)

// newService spins up a test HTTP server with the given handler and returns a
// Private API Network service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*privateapinetwork.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return privateapinetwork.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/network/private" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "foo" {
			t.Errorf("name = %q, want foo", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"elements": [
				{"id": "e1", "name": "Workspace 1", "type": "workspace", "parentFolderId": 3}
			],
			"folders": [],
			"meta": {"limit": 10, "offset": 0, "totalCount": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &privateapinetwork.ListInput{Name: "foo"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", got.TotalCount)
	}
	if len(got.Elements) != 1 || got.Elements[0].ID != "e1" || got.Elements[0].ParentFolderID != 3 {
		t.Errorf("elements mismatch: %+v", got.Elements)
	}
}

func TestAdd(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/network/private" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.AddWorkspace
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Workspace.ID != "w1" {
			t.Errorf("workspace id = %q, want w1", body.Workspace.ID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "e1", "name": "Workspace 1", "type": "workspace"}`))
	})
	defer srv.Close()

	got, err := svc.Add(context.Background(), &privateapinetwork.AddInput{WorkspaceID: "w1"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got.ID != "e1" || got.Type != "workspace" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestRemove(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/network/private/workspace/w1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"id": "w1"}}`))
	})
	defer srv.Close()

	got, err := svc.Remove(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got.WorkspaceID != "w1" {
		t.Errorf("WorkspaceID = %q, want w1", got.WorkspaceID)
	}
}

func TestUpdateFolder(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/network/private/workspace/w1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdatePanElementOrFolderRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		ws, ok := body.Workspace.Get()
		if !ok || ws.ParentFolderId.Or(0) != 5 {
			t.Errorf("workspace mismatch: %+v", body.Workspace)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := svc.UpdateFolder(context.Background(), "w1", &privateapinetwork.UpdateFolderInput{ParentFolderID: 5})
	if err != nil {
		t.Fatalf("UpdateFolder: %v", err)
	}
}

func TestListAddRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/network/private/network-entity/request/all" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "pending" {
			t.Errorf("status = %q, want pending", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"requests": [
				{
					"id": 42, "createdAt": "2026-01-01T00:00:00Z", "createdBy": 7,
					"status": "pending",
					"element": {"id": "w1", "type": "workspace", "name": "Workspace 1"}
				}
			],
			"meta": {"offset": 0, "totalCount": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.ListAddRequests(context.Background(), &privateapinetwork.ListAddRequestsInput{
		Status: privateapinetwork.RequestStatusPending,
	})
	if err != nil {
		t.Fatalf("ListAddRequests: %v", err)
	}
	if len(got.Requests) != 1 || got.Requests[0].ID != 42 || got.Requests[0].Element.Name != "Workspace 1" {
		t.Errorf("requests mismatch: %+v", got.Requests)
	}
	if got.Requests[0].Status != privateapinetwork.RequestStatusPending {
		t.Errorf("status = %q, want pending", got.Requests[0].Status)
	}
}

func TestRespondAddRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/network/private/network-entity/request/99" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.RespondPanElementAddRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Status != api.RespondPanElementAddRequestBodyStatusApproved {
			t.Errorf("status = %q, want approved", body.Status)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 99, "status": "approved"}`))
	})
	defer srv.Close()

	got, err := svc.RespondAddRequest(context.Background(), "99", &privateapinetwork.RespondInput{
		Status: privateapinetwork.DecisionApproved,
	})
	if err != nil {
		t.Fatalf("RespondAddRequest: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(got.Raw, &decoded); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if decoded["status"] != "approved" {
		t.Errorf("decoded status = %v, want approved", decoded["status"])
	}
}

func TestAPIErrorProblemDetails(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "type": "about:blank", "instance": "/network/private/workspace/w1"}`))
	})
	defer srv.Close()

	_, err := svc.Add(context.Background(), &privateapinetwork.AddInput{WorkspaceID: "w1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 || apiErr.Title != "Not Found" {
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

	_, err := svc.List(context.Background(), nil)
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

func TestRemoveBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Remove(context.Background(), "w1")
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
