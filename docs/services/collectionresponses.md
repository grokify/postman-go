# Collection Responses (Comments)

Manages comments left on saved responses (examples) within a Postman
collection. Reachable via `client.CollectionResponses()`.

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
	"github.com/grokify/postman-go/collectionresponses"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List comments on a response.
comments, err := client.CollectionResponses().Get(ctx, collectionID, responseID)
for _, c := range comments.Comments {
	fmt.Println(c.ID, c.Body)
}

// Create a comment on a response.
created, err := client.CollectionResponses().Create(ctx, collectionID, responseID,
	&collectionresponses.CreateInput{Body: "Looks good to me."})
```

## Methods

| Method | Description |
|--------|-------------|
| `Get(ctx, collectionID, responseID)` | Returns all comments left by users on a response. |
| `Create(ctx, collectionID, responseID, *CreateInput)` | Creates a comment on a response, or a reply when `ThreadID` is set. Max 10,000 characters. |
| `Update(ctx, collectionID, responseID, commentID, *UpdateInput)` | Updates an existing comment on a response. Max 10,000 characters. |
| `Delete(ctx, collectionID, responseID, commentID)` | Deletes a comment. Deleting a thread's first comment deletes the whole thread. |

### Get

```go
result, err := client.CollectionResponses().Get(ctx, "COLLECTION_ID", "RESPONSE_ID")
for _, c := range result.Comments {
	fmt.Println(c.ID, c.ThreadID, c.CreatedBy, c.Body)
}
```

### Create

```go
result, err := client.CollectionResponses().Create(ctx, "COLLECTION_ID", "RESPONSE_ID",
	&collectionresponses.CreateInput{
		Body:     "Can we add an example for the error case?",
		ThreadID: 0, // set to reply within an existing thread
		Tag: &collectionresponses.Tag{
			Type: "user",
			ID:   "USER_ID",
		},
	})
// result.ID, result.ThreadID, result.CreatedBy, result.Body
```

### Update

```go
result, err := client.CollectionResponses().Update(ctx, "COLLECTION_ID", "RESPONSE_ID", 42,
	&collectionresponses.UpdateInput{Body: "Updated: this looks great now."})
```

### Delete

```go
err := client.CollectionResponses().Delete(ctx, "COLLECTION_ID", "RESPONSE_ID", 42)
```

## Reference

Source: [`collectionresponses/`](https://github.com/grokify/postman-go/tree/main/collectionresponses) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collectionresponses)
