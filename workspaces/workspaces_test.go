package workspaces_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/workspaces"
)

// newService spins up a test HTTP server with the given handler and returns a
// Workspaces service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*workspaces.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return workspaces.New(apiClient), srv
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

func TestManagePartnerInvites(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/invitations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"action":"invite_partner"}` {
			t.Errorf("body = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	defer srv.Close()

	got, err := svc.ManagePartnerInvites(context.Background(), &workspaces.ManagePartnerInvitesInput{
		Body: json.RawMessage(`{"action":"invite_partner"}`),
	})
	if err != nil {
		t.Fatalf("ManagePartnerInvites: %v", err)
	}
	if string(got.Body) != `{"status":"ok"}` {
		t.Errorf("Body = %s", got.Body)
	}
}

func TestManagePartnerInvitesError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"title":"Unauthorized","type":"about:blank","instance":"/invitations"}`))
	})
	defer srv.Close()

	_, err := svc.ManagePartnerInvites(context.Background(), &workspaces.ManagePartnerInvitesInput{Body: json.RawMessage(`{}`)})
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 || apiErr.Title != "Unauthorized" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("type"); got != "team" {
			t.Errorf("type = %q, want team", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"workspaces": [
				{"id": "w1", "name": "Workspace 1", "type": "team", "visibility": "team", "createdBy": "1", "about": "about", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &workspaces.ListInput{Type: workspaces.WorkspaceTypeTeam, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Workspaces) != 1 || got.Workspaces[0].ID != "w1" || got.Workspaces[0].Type != workspaces.WorkspaceTypeTeam {
		t.Errorf("Workspaces mismatch: %+v", got.Workspaces)
	}
}

func TestListError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/workspaces" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateWorkspace
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		ws, ok := body.Workspace.Get()
		if !ok || ws.Name != "My Workspace" || ws.Type != api.CreateWorkspaceWorkspaceTypeTeam {
			t.Errorf("workspace mismatch: %+v", ws)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"id": "w1", "name": "My Workspace"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &workspaces.CreateInput{Name: "My Workspace", Type: workspaces.WorkspaceTypeTeam})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "w1" || got.Name != "My Workspace" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status":403,"title":"Forbidden","type":"about:blank"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &workspaces.CreateInput{Name: "x", Type: workspaces.WorkspaceTypeTeam})
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestAllRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces-roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"roles": {"user": [{"id": "1", "displayName": "Viewer"}], "usergroup": [], "partner": []}}`))
	})
	defer srv.Close()

	got, err := svc.AllRoles(context.Background())
	if err != nil {
		t.Fatalf("AllRoles: %v", err)
	}
	if len(got.User) != 1 || got.User[0].DisplayName != "Viewer" {
		t.Errorf("User mismatch: %+v", got.User)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "scim" {
			t.Errorf("include = %q, want scim", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"workspace": {
				"id": "w1", "name": "Workspace 1", "type": "team", "visibility": "team",
				"collections": [{"id": "c1", "name": "Collection 1", "uid": "u-c1"}],
				"mocks": [{"id": "m1", "name": "Mock 1", "uid": "u-m1", "deactivated": true}],
				"scim": {"createdBy": "scim-1", "updatedBy": "scim-2"}
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "w1", &workspaces.GetInput{Include: workspaces.IncludeSCIM})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "w1" || got.Type != workspaces.WorkspaceTypeTeam {
		t.Errorf("result mismatch: %+v", got)
	}
	if len(got.Collections) != 1 || got.Collections[0].UID != "u-c1" {
		t.Errorf("Collections mismatch: %+v", got.Collections)
	}
	if len(got.Mocks) != 1 || !got.Mocks[0].Deactivated {
		t.Errorf("Mocks mismatch: %+v", got.Mocks)
	}
	if got.ScimCreatedBy != "scim-1" || got.ScimUpdatedBy != "scim-2" {
		t.Errorf("scim mismatch: %+v", got)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"title":"Not Found","type":"about:blank","instance":"/workspaces/w1"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/workspaces/w1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateWorkspaceRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		ws, ok := body.Workspace.Get()
		if !ok || ws.Name.Or("") != "New Name" {
			t.Errorf("workspace mismatch: %+v", ws)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"id": "w1", "name": "New Name"}}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "w1", &workspaces.UpdateInput{Name: "New Name"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"title":"Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/workspaces/w1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"id": "w1"}}`))
	})
	defer srv.Close()

	got, err := svc.Delete(context.Background(), "w1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.ID != "w1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestActivityFeed(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/activities" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("userId"); got != "7" {
			t.Errorf("userId = %q, want 7", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"workspaceId": "w1", "id": 1, "action": "create", "elementType": "collection", "elementId": "c1", "user": {"id": 9, "username": "alice", "isPartner": false, "name": "Alice"}}
			],
			"meta": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.ActivityFeed(context.Background(), "w1", &workspaces.ActivityFeedInput{UserID: "7"})
	if err != nil {
		t.Fatalf("ActivityFeed: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Entries) != 1 || got.Entries[0].Action != "create" || got.Entries[0].User.Username != "alice" {
		t.Errorf("Entries mismatch: %+v", got.Entries)
	}
}

func TestActivityFeedError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.ActivityFeed(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestTransferElement(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/workspaces/w1/element-transfers" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.TransferWorkspaceElement
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Type != api.TransferWorkspaceElementTypeCollection || body.To != "w2" {
			t.Errorf("body mismatch: %+v", body)
		}
		var id string
		if err := json.Unmarshal(body.ID, &id); err != nil || id != "c1" {
			t.Errorf("id mismatch: %s (%v)", body.ID, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"elementTransfers": {"type": "collection", "from": "w1", "id": "c1", "to": "w2"}}}`))
	})
	defer srv.Close()

	got, err := svc.TransferElement(context.Background(), "w1", &workspaces.TransferElementInput{
		ElementID: "c1", ElementType: workspaces.TransferElementTypeCollection, To: "w2",
	})
	if err != nil {
		t.Fatalf("TransferElement: %v", err)
	}
	if got.To != "w2" || got.From != "w1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestTransferElementError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"title":"Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.TransferElement(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestGlobalVariables(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/global-variables" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values": [{"key": "k1", "type": "secret", "value": "v1", "enabled": true, "description": "desc"}]}`))
	})
	defer srv.Close()

	got, err := svc.GlobalVariables(context.Background(), "w1")
	if err != nil {
		t.Fatalf("GlobalVariables: %v", err)
	}
	if len(got.Values) != 1 || got.Values[0].Key != "k1" || got.Values[0].Type != workspaces.GlobalVariableTypeSecret || got.Values[0].Description != "desc" {
		t.Errorf("Values mismatch: %+v", got.Values)
	}
}

func TestGlobalVariablesError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":500,"title":"Internal Server Error"}`))
	})
	defer srv.Close()

	_, err := svc.GlobalVariables(context.Background(), "w1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestUpdateGlobalVariables(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/workspaces/w1/global-variables" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateGlobalVariables
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if len(body.Values) != 1 || body.Values[0].Key.Or("") != "k1" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values": [{"key": "k1", "type": "default", "value": "v1", "enabled": false}]}`))
	})
	defer srv.Close()

	got, err := svc.UpdateGlobalVariables(context.Background(), "w1", &workspaces.UpdateGlobalVariablesInput{
		Values: []workspaces.GlobalVariable{{Key: "k1", Type: workspaces.GlobalVariableTypeDefault, Value: "v1"}},
	})
	if err != nil {
		t.Fatalf("UpdateGlobalVariables: %v", err)
	}
	if len(got.Values) != 1 || got.Values[0].Key != "k1" {
		t.Errorf("result mismatch: %+v", got.Values)
	}
}

func TestRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "scim" {
			t.Errorf("include = %q, want scim", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"roles": [{"id": "4", "user": ["1", "2"], "displayName": "Viewer"}]}`))
	})
	defer srv.Close()

	got, err := svc.Roles(context.Background(), "w1", &workspaces.RolesInput{IncludeSCIM: true})
	if err != nil {
		t.Fatalf("Roles: %v", err)
	}
	if len(got.Roles) != 1 || got.Roles[0].DisplayName != "Viewer" || len(got.Roles[0].User) != 2 {
		t.Errorf("Roles mismatch: %+v", got.Roles)
	}
}

func TestRolesError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"title":"Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Roles(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestUpdateRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/workspaces/w1/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateWorkspaceRoles
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if len(body.Roles) != 1 || body.Roles[0].Op != "add" || body.Roles[0].Path != api.UpdateWorkspaceRolesRolesPathSlashUser {
			t.Errorf("body mismatch: %+v", body.Roles)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"roles": [{"id": "4", "user": ["1"], "displayName": "Viewer"}]}`))
	})
	defer srv.Close()

	got, err := svc.UpdateRoles(context.Background(), "w1", &workspaces.UpdateRolesInput{
		Operations: []workspaces.RoleOperation{
			{Op: "add", Path: workspaces.RolesPathUser, Value: []workspaces.RoleChange{{ID: "1", Role: "4"}}},
		},
	})
	if err != nil {
		t.Fatalf("UpdateRoles: %v", err)
	}
	if len(got.Roles) != 1 || got.Roles[0].ID != "4" {
		t.Errorf("result mismatch: %+v", got.Roles)
	}
}

func TestUpdateRolesUnprocessable(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":422,"title":"Unprocessable Entity"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateRoles(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
}

func TestTransferToTeam(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/workspaces/w1/transfers" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.TransferWorkspaceToTeam
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Source != "t1" || body.Destination != "t2" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workspace": {"transfer": {"id": "w1", "source": "t1", "destination": "t2"}}}`))
	})
	defer srv.Close()

	got, err := svc.TransferToTeam(context.Background(), "w1", &workspaces.TransferToTeamInput{Source: "t1", Destination: "t2"})
	if err != nil {
		t.Fatalf("TransferToTeam: %v", err)
	}
	if got.Source != "t1" || got.Destination != "t2" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestTransferToTeamError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":500,"title":"Internal Server Error"}`))
	})
	defer srv.Close()

	_, err := svc.TransferToTeam(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestUpdates(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/updates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("category"); got != "bug_fix" {
			t.Errorf("category = %q, want bug_fix", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": 1, "topic": "T1", "description": "D1", "workspaceId": "w1", "createdBy": {"id": 9, "name": "Alice", "username": "alice"}, "category": "bug_fix", "isPinned": true, "relatedResources": []}
			],
			"meta": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Updates(context.Background(), "w1", &workspaces.UpdatesInput{Category: "bug_fix"})
	if err != nil {
		t.Fatalf("Updates: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Updates) != 1 || got.Updates[0].Category != workspaces.UpdateCategoryBugFix || got.Updates[0].CreatedBy.Username != "alice" || !got.Updates[0].IsPinned {
		t.Errorf("Updates mismatch: %+v", got.Updates)
	}
}

func TestUpdatesError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"title":"Unauthorized","instance":"/workspaces/w1/updates"}`))
	})
	defer srv.Close()

	_, err := svc.Updates(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
}

func TestCreateUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/workspaces/w1/updates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateWorkspaceUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		var topic string
		if err := json.Unmarshal(body.Topic, &topic); err != nil || topic != "New Feature" {
			t.Errorf("topic mismatch: %s (%v)", body.Topic, err)
		}
		if body.Category != api.WorkspaceUpdateCategoryDataNewFeature {
			t.Errorf("category mismatch: %v", body.Category)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 5, "topic": "New Feature", "workspaceId": "w1", "createdBy": 9, "category": "new_feature"}`))
	})
	defer srv.Close()

	got, err := svc.CreateUpdate(context.Background(), "w1", &workspaces.CreateUpdateInput{
		Topic: "New Feature", Category: workspaces.UpdateCategoryNewFeature,
	})
	if err != nil {
		t.Fatalf("CreateUpdate: %v", err)
	}
	if got.ID != 5 || got.Topic != "New Feature" || got.CreatedBy != 9 {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateUpdateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":422,"title":"Unprocessable Entity"}`))
	})
	defer srv.Close()

	_, err := svc.CreateUpdate(context.Background(), "w1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
}

func TestGetUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/workspaces/w1/updates/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 5, "topic": "T1", "workspaceId": "w1", "createdBy": {"id": 9, "name": "Alice", "username": "alice"}, "category": "announcement"}`))
	})
	defer srv.Close()

	got, err := svc.GetUpdate(context.Background(), "w1", 5)
	if err != nil {
		t.Fatalf("GetUpdate: %v", err)
	}
	if got.ID != 5 || got.CreatedBy.Name != "Alice" || got.Category != workspaces.UpdateCategoryAnnouncement {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGetUpdateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"title":"Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.GetUpdate(context.Background(), "w1", 5)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestPatchUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/workspaces/w1/updates/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateWorkspaceUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if !body.IsPinned.Or(false) {
			t.Errorf("isPinned = false, want true")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 5, "topic": "T1", "workspaceId": "w1", "isPinned": true, "category": "improvement"}`))
	})
	defer srv.Close()

	got, err := svc.PatchUpdate(context.Background(), "w1", 5, &workspaces.PatchUpdateInput{
		Category: workspaces.UpdateCategoryImprovement, IsPinned: true,
	})
	if err != nil {
		t.Fatalf("PatchUpdate: %v", err)
	}
	if !got.IsPinned {
		t.Errorf("IsPinned = false, want true")
	}
}

func TestPatchUpdateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":400,"title":"Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.PatchUpdate(context.Background(), "w1", 5, nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestDeleteUpdate(t *testing.T) {
	// Postman's docs describe this endpoint as returning 204 No Content, but
	// the generated OpenAPI spec (reconstructed from the TypeScript SDK)
	// declares a 200 success response, so that is what the ogen client
	// decodes for.
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/workspaces/w1/updates/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.DeleteUpdate(context.Background(), "w1", 5); err != nil {
		t.Fatalf("DeleteUpdate: %v", err)
	}
}

func TestDeleteUpdateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status":403,"title":"Forbidden"}`))
	})
	defer srv.Close()

	err := svc.DeleteUpdate(context.Background(), "w1", 5)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}
