package environments_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/environments"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// newService spins up a test HTTP server with the given handler and returns
// an Environments service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*environments.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return environments.New(apiClient), srv
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
		if r.Method != http.MethodGet || r.URL.Path != "/environments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"environments": [
				{"id": "e1", "name": "Env 1", "uid": "u1-e1", "owner": "1", "isPublic": true}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &environments.ListInput{Workspace: "w1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Environments) != 1 {
		t.Fatalf("len(Environments) = %d, want 1", len(got.Environments))
	}
	e := got.Environments[0]
	if e.ID != "e1" || e.Name != "Env 1" || e.UID != "u1-e1" || !e.IsPublic {
		t.Errorf("Environments[0] = %+v", e)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/environments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		var body struct {
			Environment struct {
				Name   string            `json:"name"`
				Values []json.RawMessage `json:"values"`
			} `json:"environment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Environment.Name != "My Env" {
			t.Errorf("name = %q, want My Env", body.Environment.Name)
		}
		if len(body.Environment.Values) != 1 {
			t.Fatalf("len(values) = %d, want 1", len(body.Environment.Values))
		}
		var v map[string]any
		if err := json.Unmarshal(body.Environment.Values[0], &v); err != nil {
			t.Fatalf("decode value: %v", err)
		}
		if v["key"] != "token" || v["value"] != "abc123" || v["type"] != "secret" {
			t.Errorf("value = %+v", v)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"id": "e1", "name": "My Env", "uid": "u1-e1"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &environments.CreateInput{
		Workspace: "w1",
		Name:      "My Env",
		Values: []environments.Variable{
			{Key: "token", Value: "abc123", Type: environments.VariableTypeSecret, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "e1" || got.Name != "My Env" || got.UID != "u1-e1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/environments/e1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"environment": {
				"id": "e1", "name": "My Env", "owner": "1",
				"createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-02T00:00:00Z",
				"isPublic": false,
				"values": [{"key": "token", "value": "abc123", "type": "secret", "enabled": true}]
			}
		}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "e1" || got.Name != "My Env" || got.Owner != "1" {
		t.Errorf("result mismatch: %+v", got)
	}
	if len(got.Values) != 1 || got.Values[0].Key != "token" || got.Values[0].Type != environments.VariableTypeSecret {
		t.Errorf("values mismatch: %+v", got.Values)
	}
}

func TestReplace(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/environments/e1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Environment struct {
				Name   string            `json:"name"`
				Values []json.RawMessage `json:"values"`
			} `json:"environment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Environment.Name != "Renamed" {
			t.Errorf("name = %q, want Renamed", body.Environment.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"id": "e1", "name": "Renamed", "uid": "u1-e1"}}`))
	})
	defer srv.Close()

	got, err := svc.Replace(context.Background(), "e1", &environments.ReplaceInput{
		Name: "Renamed",
		Values: []environments.Variable{
			{Key: "k", Value: "v", Type: environments.VariableTypeDefault},
		},
	})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", got.Name)
	}
}

func TestPatch(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/environments/e1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var ops []struct {
			Op    string `json:"op"`
			Path  string `json:"path"`
			Value any    `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&ops); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(ops) != 1 || ops[0].Op != "replace" || ops[0].Path != "/name" || ops[0].Value != "New Name" {
			t.Errorf("ops mismatch: %+v", ops)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"id": "e1", "name": "New Name", "uid": "u1-e1"}}`))
	})
	defer srv.Close()

	got, err := svc.Patch(context.Background(), "e1", &environments.PatchInput{
		Ops: []environments.PatchOp{
			{Op: "replace", Path: "/name", Value: "New Name"},
		},
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("Name = %q, want New Name", got.Name)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/environments/e1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"id": "e1", "uid": "u1-e1"}}`))
	})
	defer srv.Close()

	got, err := svc.Delete(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.ID != "e1" || got.UID != "u1-e1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestForks(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/environments/e1/forks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("direction"); got != "desc" {
			t.Errorf("direction = %q, want desc", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{"forkId": "f1", "forkName": "Fork 1", "createdBy": "1"}],
			"meta": {"total": 1, "nextCursor": "abc"}
		}`))
	})
	defer srv.Close()

	got, err := svc.Forks(context.Background(), "e1", &environments.ForksInput{
		Direction: environments.DirectionDesc,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("Forks: %v", err)
	}
	if got.Total != 1 || got.NextCursor != "abc" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Forks) != 1 || got.Forks[0].ForkID != "f1" {
		t.Errorf("forks mismatch: %+v", got.Forks)
	}
}

func TestFork(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/environments/e1/forks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		var body api.ForkEnvironment
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.ForkName != "My Fork" {
			t.Errorf("forkName = %q, want My Fork", body.ForkName)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"uid": "u1-f1", "name": "My Env", "forkName": "My Fork"}}`))
	})
	defer srv.Close()

	got, err := svc.Fork(context.Background(), "e1", &environments.ForkInput{
		WorkspaceID: "w1",
		ForkName:    "My Fork",
	})
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if got.UID != "u1-f1" || got.ForkName != "My Fork" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestMerge(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/environments/e1/merges" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.MergeEnvironmentFork
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Source != "e1-fork" {
			t.Errorf("source = %q, want e1-fork", body.Source)
		}
		if !body.DeleteSource.Or(false) {
			t.Errorf("deleteSource = false, want true")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"uid": "u1-e1"}}`))
	})
	defer srv.Close()

	got, err := svc.Merge(context.Background(), "e1", &environments.MergeInput{
		Source:       "e1-fork",
		DeleteSource: true,
	})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got.UID != "u1-e1" {
		t.Errorf("UID = %q, want u1-e1", got.UID)
	}
}

func TestPull(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/environments/u1-f1/pulls" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.PullEnvironmentForkChanges
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Source != "e1" {
			t.Errorf("source = %q, want e1", body.Source)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"environment": {"uid": "u1-f1"}}`))
	})
	defer srv.Close()

	got, err := svc.Pull(context.Background(), "u1-f1", &environments.PullInput{Source: "e1"})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if got.UID != "u1-f1" {
		t.Errorf("UID = %q, want u1-f1", got.UID)
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
		call func(svc *environments.Service) error
	}{
		{"List", func(svc *environments.Service) error {
			_, err := svc.List(context.Background(), nil)
			return err
		}},
		{"Create", func(svc *environments.Service) error {
			_, err := svc.Create(context.Background(), &environments.CreateInput{Workspace: "w1", Name: "n"})
			return err
		}},
		{"Get", func(svc *environments.Service) error {
			_, err := svc.Get(context.Background(), "e1")
			return err
		}},
		{"Replace", func(svc *environments.Service) error {
			_, err := svc.Replace(context.Background(), "e1", nil)
			return err
		}},
		{"Delete", func(svc *environments.Service) error {
			_, err := svc.Delete(context.Background(), "e1")
			return err
		}},
		{"Forks", func(svc *environments.Service) error {
			_, err := svc.Forks(context.Background(), "e1", nil)
			return err
		}},
		{"Fork", func(svc *environments.Service) error {
			_, err := svc.Fork(context.Background(), "e1", nil)
			return err
		}},
		{"Merge", func(svc *environments.Service) error {
			_, err := svc.Merge(context.Background(), "e1", nil)
			return err
		}},
		{"Pull", func(svc *environments.Service) error {
			_, err := svc.Pull(context.Background(), "e1", nil)
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

func TestCreateErrorLengthRequired(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLengthRequired)
		_, _ = w.Write([]byte(`{"status": 411, "title": "Length Required", "detail": "missing Content-Length"}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &environments.CreateInput{Workspace: "w1", Name: "n"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusLengthRequired || apiErr.Detail != "missing Content-Length" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

// TestGetErrorNotFoundBoundTo400 documents a known quirk in the generated
// client: for GetEnvironment specifically, the Environments404Error schema
// is bound to HTTP status 400, not 404, matching the underlying OpenAPI
// spec produced from the TypeScript SDK.
func TestGetErrorNotFoundBoundTo400(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
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
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestForksErrorNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found", "detail": "no such environment"}`))
	})
	defer srv.Close()

	_, err := svc.Forks(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Detail != "no such environment" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestPatchErrorForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "detail": "cannot patch"}`))
	})
	defer srv.Close()

	_, err := svc.Patch(context.Background(), "e1", &environments.PatchInput{
		Ops: []environments.PatchOp{{Op: "replace", Path: "/name", Value: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Detail != "cannot patch" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}
