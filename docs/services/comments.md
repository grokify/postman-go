# Comments

The Comments service resolves comment threads on collections, folders,
requests, and responses. Reachable via `client.Comments()`.

!!! note "Reading comments"
    This package currently wraps thread resolution only. Comment threads on
    collections and collection items are returned in each resource's own GET
    response, not through this package.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

err := client.Comments().ResolveThread(ctx, "THREAD_ID")
```

## Methods

| Method | Description |
|--------|-------------|
| `ResolveThread(ctx, threadID)` | Resolve a comment and any associated replies in the given thread. |

### ResolveThread

```go
err := client.Comments().ResolveThread(ctx, "THREAD_ID")
if err != nil {
	// handle error
}
```

Comment thread IDs are returned in the GET response for collections and
collection items. On success the API returns an empty body.

## Reference

Source: [`comments/`](https://github.com/grokify/postman-go/tree/main/comments) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/comments)
