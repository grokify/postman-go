# Audit Logs

Lists a team's generated audit events and the catalog of audit log event
actions. Reachable via `client.AuditLogs()`.

!!! note "`orderBy` / `order_by` parameter collision"
    Postman's OpenAPI source defines two colliding query parameters for sort
    order, `orderBy` and `order_by`, that normalize to the same name. The
    generated client keeps only `orderBy` — set it via `ListInput.OrderBy`.
    `order_by` cannot be recovered (see `scripts/gen-openapi/README.md`,
    "Known approximations").

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/auditlogs"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List recent audit events.
logs, err := client.AuditLogs().List(ctx, &auditlogs.ListInput{Limit: 50})

// List the catalog of available actions.
actions, err := client.AuditLogs().Actions(ctx)
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List a team's generated audit events. Paginated. |
| `Actions(ctx)` | List the complete catalog of available audit log event actions. |

### List

```go
result, err := client.AuditLogs().List(ctx, &auditlogs.ListInput{
	UserID:  1234,
	Action:  "collection.viewed",
	Since:   "2026-01-01",
	Until:   "2026-01-31",
	Limit:   50,
	OrderBy: auditlogs.OrderDesc,
})
for _, ev := range result.Trails {
	fmt.Println(ev.Action, ev.Timestamp, ev.Data.Actor.Username)
}
```

A `nil` input returns all results.

### Actions

```go
actions, err := client.AuditLogs().Actions(ctx)
for _, a := range actions {
	fmt.Println(a.Name, a.DisplayName) // use Name in ListInput.Action
}
```

## Reference

Source: [`auditlogs/`](https://github.com/grokify/postman-go/tree/main/auditlogs) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/auditlogs)
