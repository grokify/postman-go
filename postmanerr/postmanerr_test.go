package postmanerr_test

import (
	"net/http"
	"testing"

	"github.com/grokify/postman-go/postmanerr"
)

func TestFromProblemDetails(t *testing.T) {
	raw := []byte(`{"type":"about:blank","title":"Bad Request","detail":"missing field","status":400,"instance":"/x"}`)
	err := postmanerr.FromProblemDetails(raw, http.StatusInternalServerError)
	if err.StatusCode != 400 || err.Title != "Bad Request" || err.Detail != "missing field" || err.Instance != "/x" {
		t.Errorf("unexpected error: %+v", err)
	}
}

func TestFromProblemDetailsFallback(t *testing.T) {
	err := postmanerr.FromProblemDetails(nil, http.StatusUnauthorized)
	if err.StatusCode != 401 || err.Title != "Unauthorized" {
		t.Errorf("unexpected error: %+v", err)
	}
}

func TestEmpty(t *testing.T) {
	err := postmanerr.Empty(http.StatusForbidden)
	if err.StatusCode != 403 || err.Title != "Forbidden" || err.Detail != "" {
		t.Errorf("unexpected error: %+v", err)
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &postmanerr.APIError{StatusCode: 500, Title: "Internal Server Error", Detail: "boom"}
	if got := err.Error(); got != "postman: API error (status 500): Internal Server Error: boom" {
		t.Errorf("Error() = %q", got)
	}
}
