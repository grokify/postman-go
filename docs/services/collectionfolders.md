# Collection Folders (Comments)

Manages comments left on folders within a Postman collection. Reachable via
`client.CollectionFolders()`.

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
	"github.com/grokify/postman-go/collectionfolders"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List comments on a folder.
comments, err := client.CollectionFolders().Get(ctx, collectionID, folderID)
for _, c := range comments.Comments {
	fmt.Println(c.ID, c.Body)
}

// Create a comment on a folder.
created, err := client.CollectionFolders().Create(ctx, collectionID, folderID,
	&collectionfolders.CreateInput{Body: "Looks good to me."})
```

## Methods

| Method | Description |
|--------|-------------|
| `Get(ctx, collectionID, folderID)` | Returns all comments left by users on a folder. |
| `Create(ctx, collectionID, folderID, *CreateInput)` | Creates a comment on a folder, or a reply when `ThreadID` is set. Max 10,000 characters. |
| `Update(ctx, collectionID, folderID, commentID, *UpdateInput)` | Updates an existing comment on a folder. Max 10,000 characters. |
| `Delete(ctx, collectionID, folderID, commentID)` | Deletes a comment. Deleting a thread's first comment deletes the whole thread. |

### Get

```go
result, err := client.CollectionFolders().Get(ctx, "COLLECTION_ID", "FOLDER_ID")
for _, c := range result.Comments {
	fmt.Println(c.ID, c.ThreadID, c.CreatedBy, c.Body)
}
```

### Create

```go
result, err := client.CollectionFolders().Create(ctx, "COLLECTION_ID", "FOLDER_ID",
	&collectionfolders.CreateInput{
		Body:     "Can we split this folder by environment?",
		ThreadID: 0, // set to reply within an existing thread
		Tag: &collectionfolders.Tag{
			Type: "user",
			ID:   "USER_ID",
		},
	})
// result.ID, result.ThreadID, result.CreatedBy, result.Body
```

### Update

```go
result, err := client.CollectionFolders().Update(ctx, "COLLECTION_ID", "FOLDER_ID", 42,
	&collectionfolders.UpdateInput{Body: "Updated: done, folder split."})
```

### Delete

```go
err := client.CollectionFolders().Delete(ctx, "COLLECTION_ID", "FOLDER_ID", 42)
```

## Reference

Source: [`collectionfolders/`](https://github.com/grokify/postman-go/tree/main/collectionfolders) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collectionfolders)
