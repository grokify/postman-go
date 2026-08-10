// Package billing provides a high-level client for Postman's Billing API.
//
// It reports Postman billing account details for a team and lists invoices
// for a billing account.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	account, _ := client.Billing().Accounts(ctx)
//	invoices, _ := client.Billing().Invoices(ctx, account.ID, &billing.InvoicesInput{Status: billing.AccountStatusPaid})
package billing

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// AccountStatus is the status of a Postman billing account.
type AccountStatus string

// Account status values.
const (
	// AccountStatusPaid indicates the account is on a paid plan.
	AccountStatusPaid AccountStatus = "PAID"
)

// SalesChannel is the sales channel through which a billing account was
// created.
type SalesChannel string

// Sales channel values.
const (
	SalesChannelSelfServe  SalesChannel = "SELF_SERVE"
	SalesChannelSalesServe SalesChannel = "SALES_SERVE"
)

// Service is the high-level Billing client. Obtain one via
// postman.Client.Billing.
type Service struct {
	api *api.Client
}

// New creates a Billing service over the given generated API client. Most
// callers should use postman.Client.Billing instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Accounts -----------------------------------------------------------

// Account describes a Postman billing account.
type Account struct {
	ID           int
	BillingEmail string
	State        string
	TeamID       int
	SalesChannel SalesChannel
	Slots        Slots
}

// Slots describes seat consumption for a billing account.
type Slots struct {
	Available int
	Consumed  int
	Total     int
	Unbilled  int
}

// Accounts returns the Postman billing account details for the caller's
// team.
func (s *Service) Accounts(ctx context.Context) (*Account, error) {
	res, err := s.api.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.AccountInformation:
		out := &Account{
			BillingEmail: r.BillingEmail.Or(""),
			ID:           r.ID.Or(0),
			State:        r.State.Or(""),
			TeamID:       r.TeamId.Or(0),
			SalesChannel: SalesChannel(r.SalesChannel.Or("")),
		}
		if slots, ok := r.Slots.Get(); ok {
			out.Slots = Slots{
				Available: slots.Available.Or(0),
				Consumed:  slots.Consumed.Or(0),
				Total:     slots.Total.Or(0),
				Unbilled:  slots.Unbilled.Or(0),
			}
		}
		return out, nil
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Invoices -------------------------------------------------------------

// InvoicesInput holds the filters for Invoices.
type InvoicesInput struct {
	// Status is required: the invoice status to filter by.
	Status AccountStatus
}

// InvoicesResult is the set of invoices for a billing account.
type InvoicesResult struct {
	Invoices []Invoice
}

// Invoice describes a single Postman billing account invoice.
type Invoice struct {
	ID          string
	Status      string
	IssuedAt    string
	TotalAmount InvoiceAmount
	WebURL      string
}

// InvoiceAmount is the total amount due for an invoice.
type InvoiceAmount struct {
	Value    int
	Currency string
}

// Invoices returns all invoices for a Postman billing account, filtered by
// the invoice status.
func (s *Service) Invoices(ctx context.Context, accountID string, in *InvoicesInput) (*InvoicesResult, error) {
	if in == nil {
		in = &InvoicesInput{}
	}
	params := api.GetAccountInvoicesParams{
		AccountId: accountID,
		Status:    api.BillingAccountStatus(in.Status),
	}

	res, err := s.api.GetAccountInvoices(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAccountInvoices:
		out := &InvoicesResult{}
		for _, inv := range r.Data {
			item := Invoice{
				ID:       inv.ID.Or(""),
				Status:   inv.Status.Or(""),
				IssuedAt: inv.IssuedAt.Or(""),
			}
			if total, ok := inv.TotalAmount.Get(); ok {
				item.TotalAmount = InvoiceAmount{
					Value:    total.Value.Or(0),
					Currency: total.Currency.Or(""),
				}
			}
			if links, ok := inv.Links.Get(); ok {
				if web, ok := links.Web.Get(); ok {
					item.WebURL = web.Href.Or("")
				}
			}
			out.Invoices = append(out.Invoices, item)
		}
		return out, nil
	case *api.GetAccountInvoicesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.GetAccountInvoicesForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
