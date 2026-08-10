# Collection Access Keys

Collection access keys let integrations authenticate requests scoped to a
single collection. Reachable via `client.CollectionAccessKeys()`. See
[Generate a collection access key](https://learning.postman.com/docs/developer/postman-api/authentication/#generate-a-collection-access-key)
for background.

!!! note "Key lifetime"
    Collection access keys are valid for 60 days; if unused, the key expires
    after 60 days, and each use extends the expiration by another 60 days.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/collectionaccesskeys"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List the authenticated user's collection access keys.
keys, err := client.CollectionAccessKeys().List(ctx, nil)

// Delete one.
err = client.CollectionAccessKeys().Delete(ctx, "key-id")
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List the authenticated user's personal and team collection access keys. Paginated. |
| `Delete(ctx, keyID)` | Delete a collection access key. |

### List

```go
result, err := client.CollectionAccessKeys().List(ctx, &collectionaccesskeys.ListInput{
	CollectionID: "COLLECTION_ID",
})
for _, k := range result.Keys {
	fmt.Println(k.ID, k.Status, k.ExpiresAfter, k.LastUsedAt)
}
// result.NextCursor pages further results.
```

A `nil` input returns all results.

### Delete

```go
err := client.CollectionAccessKeys().Delete(ctx, "KEY_ID")
```

## Reference

Source: [`collectionaccesskeys/`](https://github.com/grokify/postman-go/tree/main/collectionaccesskeys) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collectionaccesskeys)
