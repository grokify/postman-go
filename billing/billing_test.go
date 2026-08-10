package billing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grokify/postman-go/billing"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

func newService(t *testing.T, handler http.HandlerFunc) (*billing.Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	apiClient, err := api.NewClient(srv.URL, staticSec{key: "test-key"})
	if err != nil {
		t.Fatalf("api.NewClient: %v", err)
	}
	return billing.New(apiClient), srv
}

type staticSec struct{ key string }

func (s staticSec) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.key}, nil
}

func TestAccounts(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/accounts" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"billingEmail": "billing@example.com",
			"id": 42,
			"state": "active",
			"teamId": 7,
			"salesChannel": "SELF_SERVE",
			"slots": {"available": 1, "consumed": 2, "total": 3, "unbilled": 4}
		}`))
	})
	defer srv.Close()

	got, err := svc.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if got.ID != 42 || got.BillingEmail != "billing@example.com" || got.TeamID != 7 {
		t.Errorf("account mismatch: %+v", got)
	}
	if got.SalesChannel != billing.SalesChannelSelfServe {
		t.Errorf("SalesChannel = %q, want %q", got.SalesChannel, billing.SalesChannelSelfServe)
	}
	if got.Slots.Available != 1 || got.Slots.Consumed != 2 || got.Slots.Total != 3 || got.Slots.Unbilled != 4 {
		t.Errorf("slots mismatch: %+v", got.Slots)
	}
}

func TestAccountsError(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()

	_, err := svc.Accounts(context.Background())
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
}

func TestInvoices(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/accounts/acc1/invoices" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "PAID" {
			t.Errorf("status = %q, want PAID", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "inv1",
					"status": "PAID",
					"issuedAt": "2026-01-01T00:00:00Z",
					"totalAmount": {"value": 1000, "currency": "USD"},
					"links": {"web": {"href": "https://example.com/inv1"}}
				}
			]
		}`))
	})
	defer srv.Close()

	got, err := svc.Invoices(context.Background(), "acc1", &billing.InvoicesInput{Status: billing.AccountStatusPaid})
	if err != nil {
		t.Fatalf("Invoices: %v", err)
	}
	if len(got.Invoices) != 1 {
		t.Fatalf("len(Invoices) = %d, want 1", len(got.Invoices))
	}
	inv := got.Invoices[0]
	if inv.ID != "inv1" || inv.Status != "PAID" || inv.TotalAmount.Value != 1000 || inv.TotalAmount.Currency != "USD" {
		t.Errorf("invoice mismatch: %+v", inv)
	}
	if inv.WebURL != "https://example.com/inv1" {
		t.Errorf("WebURL = %q", inv.WebURL)
	}
}

func TestInvoicesForbidden(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status": 403, "title": "Forbidden", "type": "about:blank", "instance": "/accounts/acc1/invoices"}`))
	})
	defer srv.Close()

	_, err := svc.Invoices(context.Background(), "acc1", &billing.InvoicesInput{Status: billing.AccountStatusPaid})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 403 || apiErr.Title != "Forbidden" {
		t.Errorf("api error mismatch: %+v", apiErr)
	}
}

func TestInvoicesBadRequest(t *testing.T) {
	svc, srv := newService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status": 400, "title": "Bad Request"}`))
	})
	defer srv.Close()

	_, err := svc.Invoices(context.Background(), "acc1", &billing.InvoicesInput{Status: billing.AccountStatusPaid})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *postmanerr.APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("error is not *postmanerr.APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func asAPIError(err error, target **postmanerr.APIError) bool {
	e, ok := err.(*postmanerr.APIError)
	if ok {
		*target = e
	}
	return ok
}
