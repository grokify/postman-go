package collectionitems_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/collectionitems"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

func newService(t *testing.T, handler http.HandlerFunc) (*collectionitems.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return collectionitems.New(apiClient), srv
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

// --- Folder -----------------------------------------------------------------

func TestCreateFolder(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/folders" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateFolder
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name.Or("") != "New Folder" {
			t.Errorf("name = %+v, want New Folder", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "f1", "name": "New Folder", "owner": "o1"}, "modelId": "c1", "revision": 2}`))
	})
	defer srv.Close()

	got, err := svc.CreateFolder(context.Background(), "c1", &collectionitems.CreateFolderInput{Name: "New Folder"})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if got.Folder.ID != "f1" || got.ModelID != "c1" || got.Revision != 2 {
		t.Errorf("CreateFolder result mismatch: %+v", got)
	}
}

func TestCreateFolderError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"name": "ValidationError", "message": "bad request"}`))
	})
	defer srv.Close()

	_, err := svc.CreateFolder(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestGetFolder(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/folders/f1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("populate"); got != "true" {
			t.Errorf("populate = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"modelId": "c1", "data": {"id": "f1", "name": "Folder 1", "lastRevision": 3}}`))
	})
	defer srv.Close()

	got, err := svc.GetFolder(context.Background(), "c1", "f1", &collectionitems.GetOptions{Populate: true})
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if got.Folder.ID != "f1" || got.Folder.LastRevision != 3 {
		t.Errorf("GetFolder result mismatch: %+v", got)
	}
}

func TestGetFolderNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.GetFolder(context.Background(), "c1", "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestUpdateFolder(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/folders/f1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateFolder
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name.Or("") != "Renamed" {
			t.Errorf("name = %+v, want Renamed", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "f1", "name": "Renamed"}}`))
	})
	defer srv.Close()

	got, err := svc.UpdateFolder(context.Background(), "c1", "f1", &collectionitems.UpdateFolderInput{Name: "Renamed"})
	if err != nil {
		t.Fatalf("UpdateFolder: %v", err)
	}
	if got.Folder.Name != "Renamed" {
		t.Errorf("UpdateFolder result mismatch: %+v", got)
	}
}

func TestUpdateFolderNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"name": "NotFoundError", "message": "not found"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateFolder(context.Background(), "c1", "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestDeleteFolder(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/folders/f1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "f1", "owner": "o1"}}`))
	})
	defer srv.Close()

	got, err := svc.DeleteFolder(context.Background(), "c1", "f1")
	if err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	if got.ID != "f1" || got.Owner != "o1" {
		t.Errorf("DeleteFolder result mismatch: %+v", got)
	}
}

func TestDeleteFolderUnauthorized(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.DeleteFolder(context.Background(), "c1", "f1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}

// --- Request ------------------------------------------------------------

func TestCreateRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("folderId"); got != "f1" {
			t.Errorf("folderId = %q, want f1", got)
		}
		var body api.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Method.Or("") != "GET" {
			t.Errorf("method = %+v, want GET", body.Method)
		}
		if len(body.HeaderData) != 1 || body.HeaderData[0].Key.Or("") != "Accept" {
			t.Errorf("headerData = %+v", body.HeaderData)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "r1", "name": "Req 1", "folder": "f1"}, "modelId": "c1", "revision": 1}`))
	})
	defer srv.Close()

	got, err := svc.CreateRequest(context.Background(), "c1", &collectionitems.CreateRequestInput{
		FolderID: "f1",
		Name:     "Req 1",
		Method:   "GET",
		Headers:  []collectionitems.Header{{Key: "Accept", Value: "*/*"}},
	})
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if got.Request.ID != "r1" || got.Request.ParentFolder != "f1" {
		t.Errorf("CreateRequest result mismatch: %+v", got)
	}
}

func TestCreateRequestError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.CreateRequest(context.Background(), "c1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}

func TestGetRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/requests/r1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"modelId": "c1", "data": {"id": "r1", "name": "Req 1"}}`))
	})
	defer srv.Close()

	got, err := svc.GetRequest(context.Background(), "c1", "r1", nil)
	if err != nil {
		t.Fatalf("GetRequest: %v", err)
	}
	if got.Request.ID != "r1" {
		t.Errorf("GetRequest result mismatch: %+v", got)
	}
}

func TestGetRequestNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.GetRequest(context.Background(), "c1", "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestUpdateRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/requests/r1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.URL.Or("") != "https://example.com" {
			t.Errorf("url = %+v, want https://example.com", body.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "r1", "name": "Req 1"}}`))
	})
	defer srv.Close()

	got, err := svc.UpdateRequest(context.Background(), "c1", "r1", &collectionitems.UpdateRequestInput{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("UpdateRequest: %v", err)
	}
	if got.Request.ID != "r1" {
		t.Errorf("UpdateRequest result mismatch: %+v", got)
	}
}

func TestUpdateRequestBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.UpdateRequest(context.Background(), "c1", "r1", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

func TestDeleteRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/requests/r1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "r1", "owner": "o1"}}`))
	})
	defer srv.Close()

	got, err := svc.DeleteRequest(context.Background(), "c1", "r1")
	if err != nil {
		t.Fatalf("DeleteRequest: %v", err)
	}
	if got.ID != "r1" {
		t.Errorf("DeleteRequest result mismatch: %+v", got)
	}
}

func TestDeleteRequestError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.DeleteRequest(context.Background(), "c1", "r1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 APIError, got %v", err)
	}
}

// --- Response -----------------------------------------------------------

func TestCreateResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/c1/responses" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("request"); got != "r1" {
			t.Errorf("request = %q, want r1", got)
		}
		var body api.CreateCollectionResponseRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		rc, ok := body.ResponseCode.Get()
		if !ok || rc.Code.Or(0) != 200 {
			t.Errorf("responseCode = %+v", body.ResponseCode)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "resp1", "request": "r1"}, "modelId": "c1", "revision": 1}`))
	})
	defer srv.Close()

	in := &collectionitems.CreateResponseInput{RequestID: "r1"}
	in.Name = "OK"
	in.ResponseCode = &collectionitems.ResponseCode{Code: 200, Name: "OK"}
	got, err := svc.CreateResponse(context.Background(), "c1", in)
	if err != nil {
		t.Fatalf("CreateResponse: %v", err)
	}
	if got.Response.ID != "resp1" || got.Response.Request != "r1" {
		t.Errorf("CreateResponse result mismatch: %+v", got)
	}
}

func TestCreateResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.CreateResponse(context.Background(), "c1", &collectionitems.CreateResponseInput{RequestID: "r1"})
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 APIError, got %v", err)
	}
}

//nolint:dupl // Standalone test for a distinct endpoint; consolidating would reduce per-test clarity for a negligible reduction in duplication
func TestGetResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/collections/c1/responses/resp1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "true" {
			t.Errorf("ids = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "resp1", "request": "r1", "name": "OK"}}`))
	})
	defer srv.Close()

	got, err := svc.GetResponse(context.Background(), "c1", "resp1", &collectionitems.GetOptions{IDs: true})
	if err != nil {
		t.Fatalf("GetResponse: %v", err)
	}
	if got.Response.ID != "resp1" || got.Response.Request != "r1" {
		t.Errorf("GetResponse result mismatch: %+v", got)
	}
}

func TestGetResponseNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.GetResponse(context.Background(), "c1", "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestUpdateResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/c1/responses/resp1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateCollectionResponse1
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Name.Or("") != "Renamed" {
			t.Errorf("name = %+v, want Renamed", body.Name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "resp1", "name": "Renamed"}}`))
	})
	defer srv.Close()

	in := &collectionitems.UpdateResponseInput{}
	in.Name = "Renamed"
	got, err := svc.UpdateResponse(context.Background(), "c1", "resp1", in)
	if err != nil {
		t.Fatalf("UpdateResponse: %v", err)
	}
	if got.Response.Name != "Renamed" {
		t.Errorf("UpdateResponse result mismatch: %+v", got)
	}
}

func TestUpdateResponseNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.UpdateResponse(context.Background(), "c1", "missing", nil)
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 APIError, got %v", err)
	}
}

func TestDeleteResponse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/collections/c1/responses/resp1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"id": "resp1", "owner": "o1"}}`))
	})
	defer srv.Close()

	got, err := svc.DeleteResponse(context.Background(), "c1", "resp1")
	if err != nil {
		t.Fatalf("DeleteResponse: %v", err)
	}
	if got.ID != "resp1" {
		t.Errorf("DeleteResponse result mismatch: %+v", got)
	}
}

func TestDeleteResponseError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.DeleteResponse(context.Background(), "c1", "resp1")
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}
