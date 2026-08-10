package mocks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/mocks"
	"github.com/grokify/postman-go/postmanerr"
)

func newService(t *testing.T, handler http.HandlerFunc) (*mocks.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return mocks.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}

func TestList(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/mocks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mocks": [
			{"id": "m1", "name": "Mock 1", "isPublic": true, "config": {"matchBody": true, "delay": {"type": "fixed", "duration": 100}}}
		]}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &mocks.ListInput{Workspace: "w1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "m1" || !got[0].IsPublic {
		t.Fatalf("List result mismatch: %+v", got)
	}
	if got[0].Config == nil || !got[0].Config.MatchBody || got[0].Config.Delay == nil || got[0].Config.Delay.Duration != 100 {
		t.Errorf("Config mismatch: %+v", got[0].Config)
	}
}

func TestListError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}

func TestCreate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mocks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspace"); got != "w1" {
			t.Errorf("workspace = %q, want w1", got)
		}
		var body api.CreateMock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		m, ok := body.Mock.Get()
		if !ok || m.Collection != "c1" {
			t.Errorf("mock.collection = %+v, want c1", body.Mock)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1", "uid": "u1-m1", "collection": "c1", "name": "My Mock"}}`))
	})
	defer srv.Close()

	got, err := svc.Create(context.Background(), &mocks.CreateInput{Workspace: "w1", Collection: "c1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ID != "m1" || got.UID != "u1-m1" || got.Collection != "c1" || got.Name != "My Mock" {
		t.Errorf("Create result mismatch: %+v", got)
	}
}

func TestCreateError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Create(context.Background(), &mocks.CreateInput{Workspace: "w1", Collection: "c1"})
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestGet(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/mocks/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1", "name": "My Mock", "deactivated": true, "config": {"matchQueryParams": true}}}`))
	})
	defer srv.Close()

	got, err := svc.Get(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "m1" || !got.Deactivated {
		t.Errorf("Get result mismatch: %+v", got)
	}
	if got.Config == nil || !got.Config.MatchQueryParams {
		t.Errorf("Config mismatch: %+v", got.Config)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"name": "NotFoundError", "message": "not found"}`))
	})
	defer srv.Close()

	_, err := svc.Get(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/mocks/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateMock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		m, ok := body.Mock.Get()
		if !ok || m.Name.Or("") != "renamed" {
			t.Errorf("mock.name = %+v, want renamed", body.Mock)
		}
		cfg, ok := m.Config.Get()
		if !ok || cfg.ServerResponseId.Or("") != "sr1" {
			t.Errorf("mock.config.serverResponseId = %+v, want sr1", m.Config)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1", "name": "renamed"}}`))
	})
	defer srv.Close()

	got, err := svc.Update(context.Background(), "m1", &mocks.UpdateInput{Name: "renamed", ServerResponseID: "sr1"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "renamed" {
		t.Errorf("Update result mismatch: %+v", got)
	}
}

func TestUpdateBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Update(context.Background(), "m1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/mocks/m1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1", "uid": "u1-m1"}}`))
	})
	defer srv.Close()

	got, err := svc.Delete(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.ID != "m1" || got.UID != "u1-m1" {
		t.Errorf("Delete result mismatch: %+v", got)
	}
}

func TestDeleteError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type": "about:blank", "title": "boom", "status": 500}`))
	})
	defer srv.Close()

	_, err := svc.Delete(context.Background(), "m1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 500 || apiErr.Title != "boom" {
		t.Fatalf("expected 500 APIError, got %v", err)
	}
}

func TestCallLogs(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/mocks/m1/call-logs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("sort"); got != "servedAt" {
			t.Errorf("sort = %q, want servedAt", got)
		}
		if got := r.URL.Query().Get("direction"); got != "desc" {
			t.Errorf("direction = %q, want desc", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"callLogs": [
				{"id": "cl1", "servedAt": "2026-01-01T00:00:00Z",
				 "request": {"method": "GET", "path": "/foo", "headers": {"key": "Accept", "value": "*/*"}, "body": {"mode": "raw", "data": ""}},
				 "response": {"type": "success", "statusCode": 200, "headers": {"key": "Content-Type", "value": "application/json"}, "body": {"data": "{}"}}}
			],
			"meta": {"nextCursor": "next1"}
		}`))
	})
	defer srv.Close()

	got, err := svc.CallLogs(context.Background(), "m1", &mocks.CallLogsInput{
		Sort:      mocks.CallLogSortServedAt,
		Direction: mocks.SortDirectionDesc,
	})
	if err != nil {
		t.Fatalf("CallLogs: %v", err)
	}
	if got.NextCursor != "next1" || len(got.CallLogs) != 1 {
		t.Fatalf("CallLogs result mismatch: %+v", got)
	}
	cl := got.CallLogs[0]
	if cl.Request == nil || cl.Request.Method != "GET" || cl.Response == nil || cl.Response.StatusCode != 200 {
		t.Errorf("CallLog mismatch: %+v", cl)
	}
}

func TestCallLogsBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.CallLogs(context.Background(), "m1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestPublish(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mocks/m1/publish" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1"}}`))
	})
	defer srv.Close()

	got, err := svc.Publish(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got.ID != "m1" {
		t.Errorf("Publish result mismatch: %+v", got)
	}
}

func TestPublishNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Publish(context.Background(), "m1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestUnpublish(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/mocks/m1/unpublish" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock": {"id": "m1"}}`))
	})
	defer srv.Close()

	got, err := svc.Unpublish(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Unpublish: %v", err)
	}
	if got.ID != "m1" {
		t.Errorf("Unpublish result mismatch: %+v", got)
	}
}

func TestUnpublishBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Unpublish(context.Background(), "m1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestServerResponses(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/mocks/m1/server-responses" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": "sr1", "name": "Server Error", "statusCode": 503}]`))
	})
	defer srv.Close()

	got, err := svc.ServerResponses(context.Background(), "m1")
	if err != nil {
		t.Fatalf("ServerResponses: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sr1" || got[0].StatusCode != 503 {
		t.Errorf("ServerResponses result mismatch: %+v", got)
	}
}

func TestServerResponsesNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "not found"}`))
	})
	defer srv.Close()

	_, err := svc.ServerResponses(context.Background(), "missing")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestCreateServerResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mocks/m1/server-responses" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateMockServerResponse
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		sr, ok := body.ServerResponse.Get()
		if !ok || sr.Name != "Server Error" || sr.StatusCode != 503 {
			t.Errorf("serverResponse = %+v", body.ServerResponse)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "sr1", "name": "Server Error", "statusCode": 503, "language": "json", "mock": "m1"}`))
	})
	defer srv.Close()

	got, err := svc.CreateServerResponse(context.Background(), "m1", &mocks.CreateServerResponseInput{
		Name:       "Server Error",
		StatusCode: 503,
		Language:   mocks.ServerResponseLanguageJSON,
	})
	if err != nil {
		t.Fatalf("CreateServerResponse: %v", err)
	}
	if got.ID != "sr1" || got.Mock != "m1" || got.Language != mocks.ServerResponseLanguageJSON {
		t.Errorf("CreateServerResponse result mismatch: %+v", got)
	}
}

func TestCreateServerResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.CreateServerResponse(context.Background(), "m1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestServerResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/mocks/m1/server-responses/sr1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "sr1", "name": "Server Error", "statusCode": 503}`))
	})
	defer srv.Close()

	got, err := svc.ServerResponse(context.Background(), "m1", "sr1")
	if err != nil {
		t.Fatalf("ServerResponse: %v", err)
	}
	if got.ID != "sr1" {
		t.Errorf("ServerResponse result mismatch: %+v", got)
	}
}

func TestServerResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.ServerResponse(context.Background(), "m1", "sr1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 APIError, got %v", err)
	}
}

func TestUpdateServerResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/mocks/m1/server-responses/sr1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateMockServerResponse
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		sr, ok := body.ServerResponse.Get()
		if !ok || sr.StatusCode.Or(0) != 500 {
			t.Errorf("serverResponse.statusCode = %+v, want 500", body.ServerResponse)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "sr1", "statusCode": 500}`))
	})
	defer srv.Close()

	got, err := svc.UpdateServerResponse(context.Background(), "m1", "sr1", &mocks.UpdateServerResponseInput{StatusCode: 500})
	if err != nil {
		t.Fatalf("UpdateServerResponse: %v", err)
	}
	if got.StatusCode != 500 {
		t.Errorf("UpdateServerResponse result mismatch: %+v", got)
	}
}

func TestUpdateServerResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.UpdateServerResponse(context.Background(), "m1", "sr1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestDeleteServerResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/mocks/m1/server-responses/sr1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "sr1", "name": "Server Error", "statusCode": 503}`))
	})
	defer srv.Close()

	got, err := svc.DeleteServerResponse(context.Background(), "m1", "sr1")
	if err != nil {
		t.Fatalf("DeleteServerResponse: %v", err)
	}
	if got.ID != "sr1" || got.StatusCode != 503 {
		t.Errorf("DeleteServerResponse result mismatch: %+v", got)
	}
}

func TestDeleteServerResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.DeleteServerResponse(context.Background(), "m1", "sr1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}
