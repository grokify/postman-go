# Webhooks

The Webhooks service creates webhooks that trigger a collection run with a
custom payload. Reachable via `client.Webhooks()`.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/webhooks"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

created, err := client.Webhooks().Create(ctx, &webhooks.CreateInput{
	Workspace:  "workspace-id",
	Name:       "my webhook",
	Collection: "collection-id",
})
fmt.Println(created.WebhookURL)
```

## Methods

| Method | Description |
|--------|-------------|
| `Create(ctx, *CreateInput)` | Create a webhook that triggers a collection run with a custom payload. |

### Create

```go
created, err := client.Webhooks().Create(ctx, &webhooks.CreateInput{
	Workspace:   "workspace-id",
	Name:        "my webhook",
	Collection:  "collection-id",
	Environment: "environment-id",
})
// created.ID, created.Name, created.Collection, created.WebhookURL, created.UID
```

If `Workspace` is empty, the webhook is created in the oldest personal
Internal workspace the caller owns. Trigger the webhook by sending an HTTP
request (with an optional custom JSON payload) to `created.WebhookURL`.

## Reference

Source: [`webhooks/`](https://github.com/grokify/postman-go/tree/main/webhooks) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/webhooks)
