# Collection Requests (Comments)

Manages comments left on requests within a Postman collection. Reachable via
`client.CollectionRequests()`.

!!! note "Single tag per comment"
    The underlying generated API type only supports one `@mentioned` user per
    comment body: Postman's real schema keys tags by a `{{userName}}`
    placeholder map, which the SDK generator collapses to a single fixed
    `Tag` field.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/collectionrequests"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List comments on a request.
comments, err := client.CollectionRequests().Get(ctx, collectionID, requestID)
for _, c := range comments.Comments {
	fmt.Println(c.ID, c.Body)
}

// Create a comment on a request.
created, err := client.CollectionRequests().Create(ctx, collectionID, requestID,
	&collectionrequests.CreateInput{Body: "Looks good to me."})
```

## Methods

| Method | Description |
|--------|-------------|
| `Get(ctx, collectionID, requestID)` | Returns all comments left by users on a request. |
| `Create(ctx, collectionID, requestID, *CreateInput)` | Creates a comment on a request, or a reply when `ThreadID` is set. Max 10,000 characters. |
| `Update(ctx, collectionID, requestID, commentID, *UpdateInput)` | Updates an existing comment on a request. Max 10,000 characters. |
| `Delete(ctx, collectionID, requestID, commentID)` | Deletes a comment. Deleting a thread's first comment deletes the whole thread. |

### Get

```go
result, err := client.CollectionRequests().Get(ctx, "COLLECTION_ID", "REQUEST_ID")
for _, c := range result.Comments {
	fmt.Println(c.ID, c.ThreadID, c.CreatedBy, c.Body)
}
```

### Create

```go
result, err := client.CollectionRequests().Create(ctx, "COLLECTION_ID", "REQUEST_ID",
	&collectionrequests.CreateInput{
		Body:     "Should this header be required?",
		ThreadID: 0, // set to reply within an existing thread
		Tag: &collectionrequests.Tag{
			Type: "user",
			ID:   "USER_ID",
		},
	})
// result.ID, result.ThreadID, result.CreatedBy, result.Body
```

### Update

```go
result, err := client.CollectionRequests().Update(ctx, "COLLECTION_ID", "REQUEST_ID", 42,
	&collectionrequests.UpdateInput{Body: "Updated: resolved, thanks!"})
```

### Delete

```go
err := client.CollectionRequests().Delete(ctx, "COLLECTION_ID", "REQUEST_ID", 42)
```

## Reference

Source: [`collectionrequests/`](https://github.com/grokify/postman-go/tree/main/collectionrequests) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collectionrequests)
