// Package postmanerr defines error types shared across the postman-go SDK
// service packages.
//
// It is a leaf package (it imports nothing from the rest of the SDK) so that
// both the top-level postman package and the per-service packages can depend on
// it without creating an import cycle.
package postmanerr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents a non-2xx response from the Postman API. It mirrors the
// RFC 9457 problem-details object that Postman returns for error responses.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Type is a URI reference that identifies the problem type.
	Type string
	// Title is a short, human-readable summary of the problem type.
	Title string
	// Detail is a human-readable explanation specific to this occurrence.
	Detail string
	// Instance is a URI reference that identifies the specific occurrence.
	Instance string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := e.Title
	if msg == "" {
		msg = "request failed"
	}
	if e.Detail != "" {
		return fmt.Sprintf("postman: API error (status %d): %s: %s", e.StatusCode, msg, e.Detail)
	}
	return fmt.Sprintf("postman: API error (status %d): %s", e.StatusCode, msg)
}

// FromProblemDetails parses a raw RFC 9457 problem-details JSON body into an
// APIError. It is best-effort: a malformed or empty body still yields an
// APIError carrying the known HTTP status code.
//
// Use this for ogen error response types backed by raw JSON (i.e. whose
// underlying type resolves to jx.Raw / []byte), which is how ogen represents
// Postman's `ErrorTypeTitleStatusInstance`-shaped error schemas. Such a type
// converts to []byte directly, e.g. postmanerr.FromProblemDetails([]byte(*r), 401).
func FromProblemDetails(raw []byte, fallbackStatus int) *APIError {
	var v struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Detail   string `json:"detail"`
		Status   int    `json:"status"`
		Instance string `json:"instance"`
	}
	_ = json.Unmarshal(raw, &v)
	status := v.Status
	if status == 0 {
		status = fallbackStatus
	}
	title := v.Title
	if title == "" {
		title = http.StatusText(fallbackStatus)
	}
	return &APIError{
		StatusCode: status,
		Type:       v.Type,
		Title:      title,
		Detail:     v.Detail,
		Instance:   v.Instance,
	}
}

// Empty builds an APIError for response bodies whose schema is a union
// Postman's TypeScript SDK cannot statically resolve to a concrete shape (its
// generated OpenAPI collapses such unions to an empty object), so no error
// detail can be recovered beyond the known HTTP status code.
func Empty(status int) *APIError {
	return &APIError{StatusCode: status, Title: http.StatusText(status)}
}
