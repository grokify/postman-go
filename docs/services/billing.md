# Billing

Reports Postman billing account details for a team and lists invoices for a
billing account. Reachable via `client.Billing()`.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/billing"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Get the team's billing account details.
account, err := client.Billing().Accounts(ctx)

// List paid invoices for that account.
invoices, err := client.Billing().Invoices(ctx, fmt.Sprint(account.ID), &billing.InvoicesInput{
	Status: billing.AccountStatusPaid,
})
```

## Methods

| Method | Description |
|--------|-------------|
| `Accounts(ctx)` | Get the Postman billing account details for the caller's team. |
| `Invoices(ctx, accountID, *InvoicesInput)` | List invoices for a billing account, filtered by status. |

### Accounts

```go
account, err := client.Billing().Accounts(ctx)
fmt.Println(account.ID, account.BillingEmail, account.State, account.SalesChannel)
fmt.Println(account.Slots.Available, account.Slots.Consumed, account.Slots.Total)
```

### Invoices

```go
result, err := client.Billing().Invoices(ctx, "12345", &billing.InvoicesInput{
	Status: billing.AccountStatusPaid, // required
})
for _, inv := range result.Invoices {
	fmt.Println(inv.ID, inv.IssuedAt, inv.TotalAmount.Value, inv.TotalAmount.Currency, inv.WebURL)
}
```

## Reference

Source: [`billing/`](https://github.com/grokify/postman-go/tree/main/billing) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/billing)
