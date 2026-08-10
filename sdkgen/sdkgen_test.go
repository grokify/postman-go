package sdkgen_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/sdkgen"
)

// newService spins up a test HTTP server with the given handler and returns
// an SDK Generation service pointed at it.
func newService(t *testing.T, handler http.HandlerFunc) (*sdkgen.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return sdkgen.New(apiClient), srv
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
		if r.Method != http.MethodGet || r.URL.Path != "/sdks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "s1", "language": "go", "source": {"type": "collection", "id": "c1"}, "workspaceId": "w1", "buildStatus": "succeeded", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"nextCursor": "abc", "total": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &sdkgen.ListInput{WorkspaceID: "w1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.NextCursor != "abc" || got.Total != 1 {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.SDKs) != 1 || got.SDKs[0].ID != "s1" || got.SDKs[0].BuildStatus != sdkgen.BuildStatusSucceeded {
		t.Errorf("SDKs mismatch: %+v", got.SDKs)
	}
}

func TestListError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status": 429, "title": "Too Many Requests"}`))
	})
	defer srv.Close()

	_, err := svc.List(context.Background(), &sdkgen.ListInput{WorkspaceID: "w1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 429 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerate(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sdks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateSdk
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Language != api.SdkLanguageGo {
			t.Errorf("language = %q, want go", body.Language)
		}
		if body.Source.ID != "c1" {
			t.Errorf("source.id = %q, want c1", body.Source.ID)
		}
		goOpts, ok := body.GoOptions.Get()
		if !ok || goOpts.GoModuleName.Or("") != "github.com/acme/sdk" {
			t.Errorf("goOptions mismatch: %+v", body.GoOptions)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := svc.Generate(context.Background(), &sdkgen.GenerateInput{
		Source:    sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "c1"},
		Language:  sdkgen.LanguageGo,
		GoOptions: &sdkgen.GoOptions{ModuleName: "github.com/acme/sdk"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestGenerateUnprocessableEntity(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status": 422, "title": "Unprocessable Entity"}`))
	})
	defer srv.Close()

	err := svc.Generate(context.Background(), &sdkgen.GenerateInput{
		Source:   sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "c1"},
		Language: sdkgen.LanguageGo,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 422 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatus(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sdks/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "s1", "language": "go", "source": {"type": "collection", "id": "c1"}, "workspaceId": "w1", "buildStatus": "failed", "error": {"code": "E1", "message": "boom"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.Status(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.BuildStatus != sdkgen.BuildStatusFailed || got.Error == nil || got.Error.Code != "E1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestStatusNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Status(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/sdks/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := svc.Delete(context.Background(), "s1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDeleteConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict"}`))
	})
	defer srv.Close()

	err := svc.Delete(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownload(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sdks/s1/downloads" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "s1", "language": "go", "url": "https://example.com/sdk.zip", "expiresAt": "2026-01-01T00:05:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.Download(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got.URL != "https://example.com/sdk.zip" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestDownloadConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict"}`))
	})
	defer srv.Close()

	_, err := svc.Download(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitConnections(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sdk-git-connections" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("workspaceId"); got != "w1" {
			t.Errorf("workspaceId = %q, want w1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "active", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": true, "pullRequests": [], "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"total": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.GitConnections(context.Background(), &sdkgen.GitConnectionsInput{WorkspaceID: "w1"})
	if err != nil {
		t.Fatalf("GitConnections: %v", err)
	}
	if got.Total != 1 || len(got.Connections) != 1 {
		t.Fatalf("result mismatch: %+v", got)
	}
	c := got.Connections[0]
	if c.ID != "g1" || c.Status != sdkgen.GitConnectionStatusActive || !c.AutoUpdatePullRequestsEnabled {
		t.Errorf("Connections[0] = %+v", c)
	}
}

func TestGitConnectionsForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden"}`))
	})
	defer srv.Close()

	_, err := svc.GitConnections(context.Background(), &sdkgen.GitConnectionsInput{WorkspaceID: "w1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 403 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectGit(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sdk-git-connections" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.CreateSdkGitConnection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.RepositoryUrl != "https://github.com/acme/sdk" {
			t.Errorf("repositoryUrl = %q", body.RepositoryUrl)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "active", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": false, "pullRequests": [], "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.ConnectGit(context.Background(), &sdkgen.ConnectGitInput{
		Source:        sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "c1"},
		Language:      sdkgen.LanguageGo,
		RepositoryURL: "https://github.com/acme/sdk",
		TargetBranch:  "main",
	})
	if err != nil {
		t.Fatalf("ConnectGit: %v", err)
	}
	if got.ID != "g1" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestConnectGitConflict(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status": 409, "title": "Conflict"}`))
	})
	defer srv.Close()

	_, err := svc.ConnectGit(context.Background(), &sdkgen.ConnectGitInput{
		Source:        sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "c1"},
		Language:      sdkgen.LanguageGo,
		RepositoryURL: "https://github.com/acme/sdk",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitConnection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sdk-git-connections/g1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "active", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": false, "pullRequests": [], "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.GitConnection(context.Background(), "g1")
	if err != nil {
		t.Fatalf("GitConnection: %v", err)
	}
	if got.ID != "g1" || got.RepositoryURL != "https://github.com/acme/sdk" {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestGitConnectionNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.GitConnection(context.Background(), "g1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateGitConnection(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/sdk-git-connections/g1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body api.UpdateSdkGitConnection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Status != api.UpdateSdkGitConnectionStatusDisconnected {
			t.Errorf("status = %q, want disconnected", body.Status)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "disconnected", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": false, "pullRequests": [], "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	got, err := svc.UpdateGitConnection(context.Background(), "g1", &sdkgen.UpdateGitConnectionInput{
		Status: sdkgen.GitConnectionStatusDisconnected,
	})
	if err != nil {
		t.Fatalf("UpdateGitConnection: %v", err)
	}
	if got.Status != sdkgen.GitConnectionStatusDisconnected {
		t.Errorf("result mismatch: %+v", got)
	}
}

func TestUpdateGitConnectionUnprocessableEntity(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status": 422, "title": "Unprocessable Entity"}`))
	})
	defer srv.Close()

	_, err := svc.UpdateGitConnection(context.Background(), "g1", &sdkgen.UpdateGitConnectionInput{
		Status: sdkgen.GitConnectionStatusActive,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 422 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitConnectionPullRequests(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sdk-git-connections/g1/pull-requests" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"number": 42, "url": "https://github.com/acme/sdk/pull/42", "status": "open", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"total": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.GitConnectionPullRequests(context.Background(), "g1", nil)
	if err != nil {
		t.Fatalf("GitConnectionPullRequests: %v", err)
	}
	if got.Total != 1 || len(got.PullRequests) != 1 {
		t.Fatalf("result mismatch: %+v", got)
	}
	pr := got.PullRequests[0]
	if pr.Number != "42" || pr.Status != "open" {
		t.Errorf("PullRequests[0] = %+v", pr)
	}
}

func TestGitConnectionPullRequestsNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.GitConnectionPullRequests(context.Background(), "g1", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateAllOptions exercises every per-language options struct and the
// retry options, to cover Generate's full request-building surface.
func TestGenerateAllOptions(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		var body api.CreateSdk
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		retry, ok := body.Retry.Get()
		if !ok || !retry.Enabled.Or(false) || retry.MaxAttempts.Or(0) != 3 || retry.RetryDelay != 100 {
			t.Errorf("retry mismatch: %+v", body.Retry)
		}
		if len(retry.HttpCodesToRetry) != 1 || retry.HttpCodesToRetry[0] != 503 {
			t.Errorf("httpCodesToRetry mismatch: %+v", retry.HttpCodesToRetry)
		}
		if len(retry.HttpMethodsToRetry) != 1 || retry.HttpMethodsToRetry[0] != api.HttpMethodsToRetryGET {
			t.Errorf("httpMethodsToRetry mismatch: %+v", retry.HttpMethodsToRetry)
		}
		if len(body.Authors) != 1 || body.Authors[0].Name != "Jane" {
			t.Errorf("authors mismatch: %+v", body.Authors)
		}
		ts, _ := body.TypescriptOptions.Get()
		if ts.NpmOrg.Or("") != "acme" {
			t.Errorf("typescriptOptions mismatch: %+v", ts)
		}
		py, _ := body.PythonOptions.Get()
		if py.PypiPackageName.Or("") != "acme-sdk" {
			t.Errorf("pythonOptions mismatch: %+v", py)
		}
		java, _ := body.JavaOptions.Get()
		if java.GroupId.Or("") != "com.acme" {
			t.Errorf("javaOptions mismatch: %+v", java)
		}
		cs, _ := body.CsharpOptions.Get()
		if cs.PackageId.Or("") != "Acme.Sdk" {
			t.Errorf("csharpOptions mismatch: %+v", cs)
		}
		rb, _ := body.RubyOptions.Get()
		if rb.GemName.Or("") != "acme_sdk" {
			t.Errorf("rubyOptions mismatch: %+v", rb)
		}
		php, _ := body.PhpOptions.Get()
		if php.PackageName.Or("") != "acme/sdk" {
			t.Errorf("phpOptions mismatch: %+v", php)
		}
		kt, _ := body.KotlinOptions.Get()
		if kt.GroupId.Or("") != "com.acme" {
			t.Errorf("kotlinOptions mismatch: %+v", kt)
		}
		rust, _ := body.RustOptions.Get()
		if rust.PackageName.Or("") != "acme-sdk" {
			t.Errorf("rustOptions mismatch: %+v", rust)
		}
		cli, _ := body.CliOptions.Get()
		if cli.GoModuleName.Or("") != "github.com/acme/cli" {
			t.Errorf("cliOptions mismatch: %+v", cli)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	err := svc.Generate(context.Background(), &sdkgen.GenerateInput{
		Source:   sdkgen.Source{Type: sdkgen.SourceTypeSpec, ID: "spec1"},
		Language: sdkgen.LanguageTypescript,
		Version:  "1.0.0",
		Authors:  []sdkgen.Author{{Name: "Jane", Email: "jane@example.com"}},
		Retry: &sdkgen.RetryOptions{
			Enabled:            true,
			MaxAttempts:        3,
			RetryDelay:         100,
			MaxDelay:           5000,
			BackOffFactor:      2,
			RetryDelayJitter:   0.1,
			HTTPCodesToRetry:   []int{503},
			HTTPMethodsToRetry: []string{"GET"},
		},
		TypescriptOptions: &sdkgen.TypescriptOptions{NpmOrg: "acme", NpmName: "sdk"},
		PythonOptions:     &sdkgen.PythonOptions{PypiPackageName: "acme-sdk"},
		JavaOptions:       &sdkgen.JavaOptions{GroupID: "com.acme", ArtifactID: "sdk"},
		CsharpOptions:     &sdkgen.CsharpOptions{PackageID: "Acme.Sdk"},
		RubyOptions:       &sdkgen.RubyOptions{GemName: "acme_sdk"},
		PhpOptions:        &sdkgen.PhpOptions{PackageName: "acme/sdk"},
		KotlinOptions:     &sdkgen.KotlinOptions{GroupID: "com.acme", ArtifactID: "sdk"},
		RustOptions:       &sdkgen.RustOptions{PackageName: "acme-sdk"},
		CliOptions:        &sdkgen.CliOptions{ModuleName: "github.com/acme/cli"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

// TestListWithPullRequest covers the SDK.PullRequest mapping branch.
func TestListWithPullRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "s1", "language": "go", "source": {"type": "collection", "id": "c1"}, "workspaceId": "w1", "buildStatus": "in_progress", "pullRequest": {"url": "https://github.com/acme/sdk/pull/1", "status": "open", "sdkId": "s1"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"total": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.List(context.Background(), &sdkgen.ListInput{
		WorkspaceID: "w1",
		SDKIDs:      []string{"s1"},
		BuildStatus: sdkgen.BuildStatusInProgress,
		Language:    sdkgen.LanguageGo,
		SourceID:    "c1",
		Cursor:      "abc",
		Limit:       5,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.SDKs) != 1 || got.SDKs[0].PullRequest == nil || got.SDKs[0].PullRequest.URL != "https://github.com/acme/sdk/pull/1" {
		t.Errorf("result mismatch: %+v", got.SDKs)
	}
}

// TestGitConnectionsWithSDK covers the GitConnection.SDK mapping branch.
func TestGitConnectionsWithSDK(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "active", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": false, "pullRequests": [{"url": "https://github.com/acme/sdk/pull/1", "status": "merged", "sdkId": "s1"}], "sdk": {"id": "s1", "language": "go", "source": {"type": "collection", "id": "c1"}, "workspaceId": "w1", "buildStatus": "succeeded", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}
			],
			"meta": {"total": 1}
		}`))
	})
	defer srv.Close()

	got, err := svc.GitConnections(context.Background(), &sdkgen.GitConnectionsInput{
		WorkspaceID:   "w1",
		SourceID:      "c1",
		Language:      sdkgen.LanguageGo,
		Status:        sdkgen.GitConnectionStatusActive,
		RepositoryURL: "https://github.com/acme/sdk",
		Cursor:        "abc",
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("GitConnections: %v", err)
	}
	if len(got.Connections) != 1 || got.Connections[0].SDK == nil || len(got.Connections[0].PullRequests) != 1 {
		t.Errorf("result mismatch: %+v", got.Connections)
	}
}

// TestUpdateGitConnectionAutoUpdateFalse covers explicitly clearing
// AutoUpdatePullRequestsEnabled.
func TestUpdateGitConnectionAutoUpdateFalse(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		var body api.UpdateSdkGitConnection
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if v, ok := body.AutoUpdatePullRequestsEnabled.Get(); !ok || v {
			t.Errorf("autoUpdatePullRequestsEnabled = %+v, want set to false", body.AutoUpdatePullRequestsEnabled)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sdkGitConnectionId": "g1", "source": {"type": "collection", "id": "c1"}, "language": "go", "status": "active", "repositoryUrl": "https://github.com/acme/sdk", "targetBranch": "main", "autoUpdatePullRequestsEnabled": false, "pullRequests": [], "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z"}`))
	})
	defer srv.Close()

	disabled := false
	_, err := svc.UpdateGitConnection(context.Background(), "g1", &sdkgen.UpdateGitConnectionInput{
		Status:                        sdkgen.GitConnectionStatusActive,
		AutoUpdatePullRequestsEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("UpdateGitConnection: %v", err)
	}
}

// TestDownloadNotFoundAndDeleteNotFound cover the remaining Download and
// Delete error branches.
func TestDownloadNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	_, err := svc.Download(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status": 404, "title": "Not Found"}`))
	})
	defer srv.Close()

	err := svc.Delete(context.Background(), "s1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("unexpected error: %v", err)
	}
}
