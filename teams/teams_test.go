package teams_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/teams"
)

// newService spins up a test HTTP server with the given handler and returns a
// Teams service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*teams.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return teams.New(apiClient), srv
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
		if r.Method != http.MethodGet || r.URL.Path != "/teams" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": 1, "name": "Acme", "handle": "acme", "createdBy": 42, "enabled": true, "memberCount": 3}
			],
			"metadata": {"nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &teams.ListInput{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want abc", got.NextCursor)
	}
	if len(got.Teams) != 1 {
		t.Fatalf("len(Teams) = %d, want 1", len(got.Teams))
	}
	tm := got.Teams[0]
	if tm.ID != 1 || tm.Name != "Acme" || tm.MemberCount != 3 || tm.CreatedBy != "42" {
		t.Errorf("Teams[0] = %+v", tm)
	}
}

func TestListError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized", "instance": "/teams"}`))
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
	if apiErr.StatusCode != 401 || apiErr.Title != "Unauthorized" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/teams" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateTeam
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Name.Or("") != "New Team" {
			t.Errorf("name = %q, want New Team", body.Name.Or(""))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"team": {"id": 2, "name": "New Team", "createdBy": 7}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &teams.CreateInput{Name: "New Team"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != 2 || got.Name != "New Team" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestCreateForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "detail": "no access"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &teams.CreateInput{Name: "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Detail != "no access" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/teams/12" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "members" {
			t.Errorf("include = %q, want members", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"team": {"id": 12, "name": "Acme", "handle": "acme"}}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "12", &teams.GetInput{Include: teams.IncludeMembers})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 12 || got.Handle != "acme" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGetInternalServerError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status": 500, "title": "Internal Server Error"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "12", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestAccessRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/teams/12/access-requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": 5, "role": "TEAM_DEVELOPER", "requestType": "REQUEST_TO_JOIN", "entityType": "user", "entityId": 99, "createdBy": 1}
			],
			"metadata": {"nextCursor": "xyz"}
		}`))
	})
	defer srv.Close()

	got, err := svc.AccessRequests(context.Background(), "12", nil)
	if err != nil {
		t.Fatalf("AccessRequests: %v", err)
	}
	if got.NextCursor != "xyz" || len(got.Requests) != 1 {
		t.Fatalf("result mismatch: %+v", got)
	}
	ar := got.Requests[0]
	if ar.ID != 5 || ar.Role != teams.RoleTeamDeveloper || ar.EntityID != "99" {
		t.Errorf("Requests[0] = %+v", ar)
	}
}

func TestAccessRequestsForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.AccessRequests(context.Background(), "12", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestAccess(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/teams/12/access-requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateAccessRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.RequestType != api.RequestTypeREQUESTTOJOIN {
			t.Errorf("requestType = %q, want REQUEST_TO_JOIN", body.RequestType)
		}
		if len(body.EntityList) != 1 || body.EntityList[0].EntityType != api.TeamEntityInfoEntityTypeUser {
			t.Errorf("entityList mismatch: %+v", body.EntityList)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": [{"entityType": "user", "entityId": 99, "status": "pending"}]}`))
	})
	defer srv.Close()

	got, err := svc.RequestAccess(context.Background(), "12", &teams.RequestAccessInput{
		Entities:    []teams.EntityRef{{Type: teams.EntityTypeUser, ID: "99"}},
		RequestType: teams.RequestTypeJoin,
	})
	if err != nil {
		t.Fatalf("RequestAccess: %v", err)
	}
	if len(got.Records) != 1 || got.Records[0].Status != "pending" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDecideAccessRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/teams/12/access-requests/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.ApproveDenyAccessRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Action != api.ApproveDenyAccessRequestActionApprove {
			t.Errorf("action = %q, want approve", body.Action)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": {"entityType": "user", "entityId": 99, "status": "approved"}}`))
	})
	defer srv.Close()

	got, err := svc.DecideAccessRequest(context.Background(), "12", "5", teams.AccessActionApprove)
	if err != nil {
		t.Fatalf("DecideAccessRequest: %v", err)
	}
	if got.Status != "approved" || got.EntityID != "99" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDecideAccessRequestNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.DecideAccessRequest(context.Background(), "12", "5", teams.AccessActionDeny)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManageMemberRoles(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/teams/12/bulk-members" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.ManageTeamMemberRoles
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		add, ok := body.Add.Get()
		if !ok {
			t.Fatal("expected add to be set")
		}
		users, ok := add.Users.Get()
		if !ok || len(users.UserId) != 1 || users.UserId[0] != api.TeamRolesTEAMDEVELOPER {
			t.Errorf("add.users mismatch: %+v", add.Users)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": [{"entityType": "user", "entityId": 99, "role": "TEAM_DEVELOPER", "status": "added"}]}`))
	})
	defer srv.Close()

	got, err := svc.ManageMemberRoles(context.Background(), "12", &teams.ManageMemberRolesInput{
		Add: &teams.MemberRoleChanges{Users: []teams.Role{teams.RoleTeamDeveloper}},
	})
	if err != nil {
		t.Fatalf("ManageMemberRoles: %v", err)
	}
	if len(got.Results) != 1 || got.Results[0].Status != "added" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestRemoveMembers(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/teams/12/bulk-members" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.RemoveTeamMembers
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if len(body.Entities) != 1 || body.Entities[0].EntityType != api.TeamEntityInfoEntityTypeUser {
			t.Errorf("entities mismatch: %+v", body.Entities)
		}
		// The generated client (see internal/api/oas_response_decoders_gen.go)
		// decodes this operation's success response as status 200, not the
		// documented 204 No Content.
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := svc.RemoveMembers(context.Background(), "12", []teams.EntityRef{{Type: teams.EntityTypeUser, ID: "99"}})
	if err != nil {
		t.Fatalf("RemoveMembers: %v", err)
	}
}

func TestRemoveMembersForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	err := svc.RemoveMembers(context.Background(), "12", []teams.EntityRef{{Type: teams.EntityTypeUser, ID: "99"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSettings(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/teams/12/settings" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"settings": {"rfaForAddMember": "enabled", "rfaForAddCollaborator": "disabled"}}`))
	})
	defer srv.Close()

	got, err := svc.Settings(context.Background(), "12")
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got.RfaForAddMember != "enabled" || got.RfaForAddCollaborator != "disabled" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateSettings(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/teams/12/settings" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateTeamSettings
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		settings, ok := body.Settings.Get()
		if !ok {
			t.Fatal("expected settings to be set")
		}
		if settings.RfaForAddMember.Or("") != api.RfaForAddMemberEnabled {
			t.Errorf("rfaForAddMember = %q, want enabled", settings.RfaForAddMember.Or(""))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"settings": {"rfaForAddMember": "enabled", "rfaForAddCollaborator": "enabled"}}`))
	})
	defer srv.Close()

	got, err := svc.UpdateSettings(context.Background(), "12", &teams.UpdateSettingsInput{RfaForAddMember: teams.RfaEnabled})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if got.RfaForAddMember != "enabled" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestSettingsUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status": 401, "title": "Unauthorized"}`))
	})
	defer srv.Close()

	_, err := svc.Settings(context.Background(), "12")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 401 {
		t.Fatalf("unexpected error: %v", err)
	}
}
