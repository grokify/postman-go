# Tags

Tags let a team organize and search workspaces, APIs, and collections that
share a common label. Tagging is available on Postman Solo, Team, and
Enterprise plans. Reachable via `client.Tags()`.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Replace a workspace's tags.
tags, err := client.Tags().UpdateWorkspace(ctx, "WORKSPACE_ID", []string{"public-api", "billing"})

// Read them back.
got, err := client.Tags().Workspace(ctx, "WORKSPACE_ID")
fmt.Println(got) // ["public-api" "billing"]

// Find everything tagged "billing".
entities, err := client.Tags().Entities(ctx, "billing", nil)
for _, e := range entities.Entities {
	fmt.Println(e.ID, e.Type)
}
```

## Methods

| Method | Description |
|--------|-------------|
| `Collection(ctx, collectionID)` | Get the tags (slugs) associated with a collection. |
| `UpdateCollection(ctx, collectionID, slugs)` | Replace all of a collection's tags (up to 5). |
| `Workspace(ctx, workspaceID)` | Get the tags (slugs) associated with a workspace. |
| `UpdateWorkspace(ctx, workspaceID, slugs)` | Replace all of a workspace's tags (up to 5). |
| `Entities(ctx, slug, *EntitiesInput)` | Get elements (workspaces, APIs, collections) tagged with a given tag. Paginated. |

### Collection

```go
tags, err := client.Tags().Collection(ctx, "COLLECTION_ID")
fmt.Println(tags) // []string of slugs
```

### UpdateCollection

```go
tags, err := client.Tags().UpdateCollection(ctx, "COLLECTION_ID", []string{"public-api", "v2"})
```

Replaces all of the collection's tags with the given slugs; up to 5 tags
are allowed.

### Workspace

```go
tags, err := client.Tags().Workspace(ctx, "WORKSPACE_ID")
fmt.Println(tags) // []string of slugs
```

### UpdateWorkspace

```go
tags, err := client.Tags().UpdateWorkspace(ctx, "WORKSPACE_ID", []string{"public-api", "billing"})
```

Replaces all of the workspace's tags with the given slugs; up to 5 tags are
allowed.

### Entities

```go
entities, err := client.Tags().Entities(ctx, "billing", &tags.EntitiesInput{
	EntityType: tags.EntityTypeWorkspace,
	Limit:      50,
	Direction:  tags.SortDesc,
})
for _, e := range entities.Entities {
	fmt.Println(e.ID, e.Type)
}
```

`Slug` is the tag's ID within a team or individual (non-team) user scope. A
`nil` input returns all entity types.

## Reference

Source: [`tags/`](https://github.com/grokify/postman-go/tree/main/tags) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/tags)
